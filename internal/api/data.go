package api

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"media-crawler-go/internal/config"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type dataFileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ModifiedAt  int64  `json:"modified_at"`
	RecordCount *int   `json:"record_count,omitempty"`
	Type        string `json:"type"`
}

func (s *Server) handleDataFilesList(w http.ResponseWriter, r *http.Request) {
	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	files, err := listDataFiles(dataDir, r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) handleDataFile(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/data/files/")
	if rel == "" || rel == "/" {
		s.handleDataFilesList(w, r)
		return
	}

	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	fullPath, err := safeDataPath(dataDir, rel)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "access denied"})
		return
	}

	preview := queryBoolDefault(r.URL.Query(), "preview", true)
	limit := queryIntDefault(r.URL.Query(), "limit", 100)
	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "not a file"})
		return
	}

	if preview {
		data, total, columns, err := previewDataFile(fullPath, limit)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		resp := map[string]any{"data": data, "total": total}
		if len(columns) > 0 {
			resp["columns"] = columns
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	serveDownload(w, r, fullPath, filepath.Base(fullPath))
}

func (s *Server) handleDataDownload(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/data/download/")
	if rel == "" || rel == "/" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing file path"})
		return
	}

	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	fullPath, err := safeDataPath(dataDir, rel)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "access denied"})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "not a file"})
		return
	}

	serveDownload(w, r, fullPath, filepath.Base(fullPath))
}

func (s *Server) handleDataStats(w http.ResponseWriter, r *http.Request) {
	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	stats, err := dataStats(dataDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func listDataFiles(dataDir string, q url.Values) ([]dataFileInfo, error) {
	platform := strings.TrimSpace(q.Get("platform"))
	fileType := strings.TrimSpace(q.Get("file_type"))
	supportedExt := map[string]struct{}{
		".json":  {},
		".jsonl": {},
		".csv":   {},
		".db":    {},
		".svg":   {},
		".png":   {},
		".xlsx":  {},
	}

	if _, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			return []dataFileInfo{}, nil
		}
		return nil, err
	}

	out := make([]dataFileInfo, 0, 64)
	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := supportedExt[ext]; !ok {
			return nil
		}
		if fileType != "" && strings.ToLower(strings.TrimPrefix(ext, ".")) != strings.ToLower(fileType) {
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if platform != "" && !strings.Contains(strings.ToLower(rel), strings.ToLower(platform)) {
			return nil
		}

		fi, err := os.Stat(path)
		if err != nil {
			return nil
		}

		rc, _ := tryCountRecords(path)
		item := dataFileInfo{
			Name:       d.Name(),
			Path:       rel,
			Size:       fi.Size(),
			ModifiedAt: fi.ModTime().Unix(),
			Type:       strings.TrimPrefix(ext, "."),
		}
		if rc != nil {
			item.RecordCount = rc
		}
		out = append(out, item)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortDataFiles(out)
	return out, nil
}

func sortDataFiles(files []dataFileInfo) {
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].ModifiedAt > files[i].ModifiedAt {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

func safeDataPath(dataDir, rel string) (string, error) {
	if strings.Contains(rel, "\x00") {
		return "", errors.New("invalid path")
	}
	rel = strings.TrimPrefix(rel, "/")
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || rel == "" {
		return "", errors.New("invalid path")
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("invalid path")
	}
	full := filepath.Join(dataDir, rel)
	dataAbs, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	relTo, err := filepath.Rel(dataAbs, fullAbs)
	if err != nil {
		return "", err
	}
	relTo = filepath.Clean(relTo)
	if relTo == "." {
		return "", errors.New("invalid path")
	}
	if strings.HasPrefix(relTo, ".."+string(filepath.Separator)) || relTo == ".." {
		return "", errors.New("access denied")
	}
	return fullAbs, nil
}

func queryBoolDefault(q url.Values, key string, defaultValue bool) bool {
	raw := strings.TrimSpace(q.Get(key))
	if raw == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}
	return b
}

func queryIntDefault(q url.Values, key string, defaultValue int) int {
	raw := strings.TrimSpace(q.Get(key))
	if raw == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return n
}

func previewDataFile(path string, limit int) (any, int, []string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, nil, err
		}
		defer f.Close()

		var v any
		dec := json.NewDecoder(f)
		if err := dec.Decode(&v); err != nil {
			return nil, 0, nil, err
		}
		switch vv := v.(type) {
		case []any:
			total := len(vv)
			if total > limit {
				vv = vv[:limit]
			}
			return vv, total, nil, nil
		default:
			return vv, 1, nil, nil
		}
	case ".jsonl":
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, nil, err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		data := make([]any, 0, minInt(limit, 64))
		total := 0
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			total++
			if len(data) < limit {
				var item any
				if err := json.Unmarshal([]byte(line), &item); err != nil {
					return nil, 0, nil, err
				}
				data = append(data, item)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, 0, nil, err
		}
		return data, total, nil, nil
	case ".csv":
		f, err := os.Open(path)
		if err != nil {
			return nil, 0, nil, err
		}
		defer f.Close()

		reader := csv.NewReader(f)
		header, err := reader.Read()
		if err != nil {
			return nil, 0, nil, err
		}
		if len(header) > 0 {
			header[0] = strings.TrimPrefix(header[0], "\uFEFF")
		}

		rows := make([]map[string]string, 0, minInt(limit, 64))
		total := 0
		for {
			rec, err := reader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, 0, nil, err
			}
			total++
			if len(rows) < limit {
				obj := map[string]string{}
				for i, k := range header {
					if i < len(rec) {
						obj[k] = rec[i]
					} else {
						obj[k] = ""
					}
				}
				rows = append(rows, obj)
			}
		}
		return rows, total, header, nil
	case ".xlsx":
		f, err := excelize.OpenFile(path)
		if err != nil {
			return nil, 0, nil, err
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return []any{}, 0, nil, nil
		}
		rows, err := f.GetRows(sheets[0])
		if err != nil {
			return nil, 0, nil, err
		}
		if len(rows) == 0 {
			return []any{}, 0, nil, nil
		}
		header := rows[0]
		total := len(rows) - 1
		if total < 0 {
			total = 0
		}
		data := make([]map[string]string, 0, minInt(limit, 64))
		for i := 1; i < len(rows) && len(data) < limit; i++ {
			rec := rows[i]
			obj := map[string]string{}
			for j, k := range header {
				if j < len(rec) {
					obj[k] = rec[j]
				} else {
					obj[k] = ""
				}
			}
			data = append(data, obj)
		}
		return data, total, header, nil
	default:
		return nil, 0, nil, errors.New("unsupported file type for preview")
	}
}

func tryCountRecords(path string) (*int, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var v any
		dec := json.NewDecoder(f)
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		switch vv := v.(type) {
		case []any:
			n := len(vv)
			return &n, nil
		default:
			n := 1
			return &n, nil
		}
	case ".jsonl":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		n := 0
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == "" {
				continue
			}
			n++
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return &n, nil
	case ".csv":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		reader := bufio.NewReader(f)
		n := 0
		for {
			_, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, err
			}
			n++
		}
		if n <= 0 {
			return nil, nil
		}
		n--
		if n < 0 {
			n = 0
		}
		return &n, nil
	case ".xlsx":
		f, err := excelize.OpenFile(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			n := 0
			return &n, nil
		}
		rows, err := f.GetRows(sheets[0])
		if err != nil {
			return nil, err
		}
		n := len(rows) - 1
		if n < 0 {
			n = 0
		}
		return &n, nil
	default:
		return nil, nil
	}
}

func dataStats(dataDir string) (map[string]any, error) {
	if _, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"total_files": 0, "total_size": 0, "by_platform": map[string]int{}, "by_type": map[string]int{}}, nil
		}
		return nil, err
	}

	byPlatform := map[string]int{}
	byType := map[string]int{}
	totalFiles := 0
	var totalSize int64

	platformKeys := []string{"xhs", "dy", "ks", "bili", "wb", "tieba", "zhihu", "douyin", "bilibili", "weibo"}

	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == "" {
			return nil
		}

		fi, err := os.Stat(path)
		if err != nil {
			return nil
		}
		totalFiles++
		totalSize += fi.Size()

		typ := strings.TrimPrefix(ext, ".")
		byType[typ]++

		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return nil
		}
		relLower := strings.ToLower(filepath.ToSlash(rel))
		for _, k := range platformKeys {
			if strings.Contains(relLower, k) {
				byPlatform[k]++
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"total_files":  totalFiles,
		"total_size":   totalSize,
		"by_platform":  byPlatform,
		"by_type":      byType,
		"generated_at": nowUnix(),
	}, nil
}

func serveDownload(w http.ResponseWriter, r *http.Request, path, filename string) {
	f, err := os.Open(path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer f.Close()

	w.Header().Set("content-type", "application/octet-stream")
	w.Header().Set("content-disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
