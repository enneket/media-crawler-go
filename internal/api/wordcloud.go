package api

import (
	"bufio"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"media-crawler-go/internal/config"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type wordCount struct {
	Word  string
	Count int
}

func (s *Server) handleDataWordcloud(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	platform := strings.TrimSpace(q.Get("platform"))
	if platform == "" {
		platform = strings.TrimSpace(config.AppConfig.Platform)
	}
	if platform == "" {
		platform = "xhs"
	}
	noteID := strings.TrimSpace(q.Get("note_id"))

	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}

	maxComments := queryIntDefault(q, "max_comments", 5000)
	if maxComments < 1 {
		maxComments = 1
	}
	if maxComments > 200000 {
		maxComments = 200000
	}
	maxWords := queryIntDefault(q, "max_words", 200)
	if maxWords < 10 {
		maxWords = 10
	}
	if maxWords > 1000 {
		maxWords = 1000
	}
	minCount := queryIntDefault(q, "min_count", 2)
	if minCount < 1 {
		minCount = 1
	}
	width := queryIntDefault(q, "width", 1200)
	height := queryIntDefault(q, "height", 800)
	if width < 300 {
		width = 300
	}
	if height < 200 {
		height = 200
	}
	save := queryBoolDefault(q, "save", true)
	useCache := queryBoolDefault(q, "cache", true)

	if !save && useCache && s.cache != nil {
		key := fmt.Sprintf("wordcloud:comments:%s:%s:%d:%d:%d:%d:%d", platform, noteID, maxComments, maxWords, minCount, width, height)
		if b, ok, err := s.cache.Get(r.Context(), key); err == nil && ok && len(b) > 0 {
			w.Header().Set("content-type", "image/svg+xml; charset=utf-8")
			w.Header().Set("content-disposition", `inline; filename="wordcloud.svg"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
			return
		}
	}

	ctx := r.Context()
	texts, err := collectCommentTexts(ctx, dataDir, platform, noteID, maxComments)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if len(texts) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no comments found"})
		return
	}

	counts := countWords(texts, maxWords, minCount)
	if len(counts) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "no words after filtering"})
		return
	}

	seed := seedFor(platform + ":" + noteID)
	svg := renderWordcloudSVG(counts, width, height, seed)
	if !save && useCache && s.cache != nil {
		ttlSec := config.AppConfig.CacheDefaultTTLSec
		if ttlSec <= 0 {
			ttlSec = 600
		}
		key := fmt.Sprintf("wordcloud:comments:%s:%s:%d:%d:%d:%d:%d", platform, noteID, maxComments, maxWords, minCount, width, height)
		_ = s.cache.Set(r.Context(), key, []byte(svg), time.Duration(ttlSec)*time.Second)
	}

	var relPath string
	if save {
		fn := "wordcloud_comments_" + time.Now().Format("20060102_150405") + ".svg"
		if noteID != "" {
			fn = "wordcloud_" + sanitizeFilename(noteID) + "_" + time.Now().Format("20060102_150405") + ".svg"
		}
		outDir := filepath.Join(dataDir, platform)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		full := filepath.Join(outDir, fn)
		if err := os.WriteFile(full, []byte(svg), 0644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if r, err := filepath.Rel(dataDir, full); err == nil {
			relPath = filepath.ToSlash(r)
			w.Header().Set("x-generated-file", relPath)
		}
	}

	w.Header().Set("content-type", "image/svg+xml; charset=utf-8")
	if relPath != "" {
		w.Header().Set("content-disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(relPath)))
	} else {
		w.Header().Set("content-disposition", `inline; filename="wordcloud.svg"`)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

func collectCommentTexts(ctx context.Context, dataDir, platform, noteID string, max int) ([]string, error) {
	if max <= 0 {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(config.AppConfig.StoreBackend)) {
	case "sqlite":
		return collectCommentTextsFromSQLite(ctx, platform, noteID, max)
	case "mysql":
		return collectCommentTextsFromMySQLDSN(ctx, platform, noteID, max, config.AppConfig.MySQLDSN)
	case "postgres", "postgresql":
		return collectCommentTextsFromPostgresDSN(ctx, platform, noteID, max, config.AppConfig.PostgresDSN)
	case "mongodb", "mongo":
		return collectCommentTextsFromMongo(ctx, platform, noteID, max, config.AppConfig.MongoURI, config.AppConfig.MongoDB)
	}
	return collectCommentTextsFromFiles(ctx, dataDir, platform, noteID, max)
}

func collectCommentTextsFromSQLite(ctx context.Context, platform, noteID string, max int) ([]string, error) {
	p := strings.TrimSpace(config.AppConfig.SQLitePath)
	if p == "" {
		p = "data/media_crawler.db"
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var rows *sql.Rows
	if noteID != "" {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=? AND note_id=? ORDER BY created_at DESC LIMIT ?`, platform, noteID, max)
	} else {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=? ORDER BY created_at DESC LIMIT ?`, platform, max)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0, 256)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		text := extractCommentTextJSON([]byte(raw))
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

func collectCommentTextsFromFiles(ctx context.Context, dataDir, platform, noteID string, max int) ([]string, error) {
	root := filepath.Join(dataDir, platform, "notes")
	if noteID != "" {
		root = filepath.Join(root, noteID)
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]string, 0, 256)
	walkRoot := root
	if noteID != "" {
		walkRoot = root
	}
	err := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(d.Name()) != "comments.jsonl" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 1024*1024)
		sc.Buffer(buf, 4*1024*1024)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			text := extractCommentTextJSON([]byte(line))
			if strings.TrimSpace(text) == "" {
				continue
			}
			out = append(out, text)
			if len(out) >= max {
				return context.Canceled
			}
		}
		return nil
	})
	if err != nil && !errorsIsCanceled(err) {
		return nil, err
	}
	if len(out) > max {
		out = out[:max]
	}
	return out, nil
}

func errorsIsCanceled(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled)
}

func extractCommentTextJSON(b []byte) string {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	return extractCommentTextAny(m)
}

func extractCommentTextAny(v any) string {
	switch vv := v.(type) {
	case map[string]any:
		for _, k := range []string{"content", "text", "comment", "message", "desc"} {
			if s, ok := vv[k].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
		for _, k := range []string{"data", "comment"} {
			if inner, ok := vv[k]; ok {
				if s := extractCommentTextAny(inner); strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
	case []any:
		for _, it := range vv {
			if s := extractCommentTextAny(it); strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func countWords(texts []string, maxWords int, minCount int) []wordCount {
	stop := defaultStopwords()
	m := make(map[string]int, 2048)
	for _, t := range texts {
		for _, tok := range tokenize(t) {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			if _, ok := stop[tok]; ok {
				continue
			}
			if isASCIIWord(tok) && len(tok) < 2 {
				continue
			}
			m[tok]++
		}
	}
	out := make([]wordCount, 0, len(m))
	for w, c := range m {
		if c < minCount {
			continue
		}
		out = append(out, wordCount{Word: w, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Word < out[j].Word
	})
	if len(out) > maxWords {
		out = out[:maxWords]
	}
	return out
}

func tokenize(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	var out []string
	var buf []rune
	mode := 0
	flush := func() {
		if len(buf) == 0 {
			return
		}
		out = append(out, string(buf))
		buf = buf[:0]
		mode = 0
	}
	for _, r := range []rune(s) {
		if isASCIIAlphaNum(r) {
			if mode != 1 {
				flush()
				mode = 1
			}
			buf = append(buf, r)
			continue
		}
		if isHan(r) {
			flush()
			out = append(out, string(r))
			continue
		}
		flush()
	}
	flush()
	return out
}

func isASCIIAlphaNum(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if r == '_' {
		return true
	}
	return false
}

func isASCIIWord(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func isHan(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}

func defaultStopwords() map[string]struct{} {
	words := []string{
		"的", "了", "和", "是", "我", "你", "他", "她", "它", "也", "就", "都", "而", "及", "与", "着", "或",
		"在", "对", "很", "吗", "吧", "啊", "呢", "呀", "哦", "么", "一个", "我们", "你们", "他们", "她们",
		"the", "a", "an", "and", "or", "to", "of", "in", "on", "for", "is", "are", "was", "were", "be", "been", "it",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

type box struct {
	x0 float64
	y0 float64
	x1 float64
	y1 float64
}

func (b box) overlaps(o box) bool {
	return !(b.x1 <= o.x0 || b.x0 >= o.x1 || b.y1 <= o.y0 || b.y0 >= o.y1)
}

func renderWordcloudSVG(words []wordCount, width, height int, seed int64) string {
	maxC := words[0].Count
	minC := words[len(words)-1].Count
	if minC <= 0 {
		minC = 1
	}
	rnd := rand.New(rand.NewSource(seed))
	palette := []string{"#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd", "#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf"}

	canvasW := float64(width)
	canvasH := float64(height)
	cx := canvasW / 2
	cy := canvasH / 2

	placed := make([]box, 0, len(words))
	var sb strings.Builder
	sb.Grow(4096)
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height))

	for i, wc := range words {
		fontSize := scaleFont(wc.Count, minC, maxC, 16, 84)
		txt := escapeXML(wc.Word)
		w := estimateTextWidth(wc.Word, fontSize)
		h := float64(fontSize)

		var chosen box
		ok := false
		for step := 0; step < 1600; step++ {
			t := float64(step) / 6
			ang := t * 0.7
			rad := 6 + t*3.2
			x := cx + rad*math.Cos(ang) + (rnd.Float64()-0.5)*6
			y := cy + rad*math.Sin(ang) + (rnd.Float64()-0.5)*6
			b := box{x0: x - w/2, y0: y - h/2, x1: x + w/2, y1: y + h/2}
			if b.x0 < 6 || b.y0 < 6 || b.x1 > canvasW-6 || b.y1 > canvasH-6 {
				continue
			}
			collide := false
			for _, p := range placed {
				if b.overlaps(p) {
					collide = true
					break
				}
			}
			if collide {
				continue
			}
			chosen = b
			ok = true
			break
		}
		if !ok {
			continue
		}

		placed = append(placed, chosen)
		x := (chosen.x0 + chosen.x1) / 2
		y := (chosen.y0 + chosen.y1) / 2
		fill := palette[i%len(palette)]
		sb.WriteString(fmt.Sprintf(`<text x="%s" y="%s" text-anchor="middle" dominant-baseline="central" font-size="%d" fill="%s" font-family="system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial">%s</text>`,
			formatFloat(x), formatFloat(y), fontSize, fill, txt))
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

func scaleFont(v, minV, maxV, minSize, maxSize int) int {
	if v <= minV {
		return minSize
	}
	if v >= maxV {
		return maxSize
	}
	if maxV == minV {
		return minSize
	}
	t := float64(v-minV) / float64(maxV-minV)
	t = math.Sqrt(t)
	return int(math.Round(float64(minSize) + t*float64(maxSize-minSize)))
}

func estimateTextWidth(s string, fontSize int) float64 {
	runes := []rune(s)
	w := 0.0
	for _, r := range runes {
		if r <= 127 {
			w += 0.55
		} else {
			w += 1.0
		}
	}
	return w * float64(fontSize)
}

func seedFor(s string) int64 {
	h := sha1.Sum([]byte(strings.TrimSpace(s)))
	var v int64
	for i := 0; i < 8; i++ {
		v = (v << 8) | int64(h[i])
	}
	return v
}

func sanitizeFilename(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return "x"
	}
	var b strings.Builder
	for _, r := range []rune(v) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
			continue
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	out := b.String()
	if len([]rune(out)) > 64 {
		h := sha1.Sum([]byte(out))
		return hex.EncodeToString(h[:8])
	}
	return out
}

func escapeXML(s string) string {
	repl := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return repl.Replace(s)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
