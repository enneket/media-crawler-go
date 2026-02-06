package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"media-crawler-go/internal/config"

	"github.com/xuri/excelize/v2"
)

var (
	bookMu       sync.Mutex
	bookFilename string
)

func BeginRunWorkbook() {
	bookMu.Lock()
	defer bookMu.Unlock()

	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	mode := strings.TrimSpace(config.AppConfig.CrawlerType)
	if mode == "" {
		mode = "search"
	}
	ts := time.Now().Format("20060102_150405")
	bookFilename = fmt.Sprintf("%s_%s_%s.xlsx", platform, mode, ts)
}

func workbookPath() string {
	bookMu.Lock()
	fn := bookFilename
	bookMu.Unlock()

	if fn == "" {
		BeginRunWorkbook()
		bookMu.Lock()
		fn = bookFilename
		bookMu.Unlock()
	}
	return filepath.Join(PlatformDir(), fn)
}

func AppendBookContents(noteID string, note any) error {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return nil
	}
	header, row, err := toTabular(note)
	if err != nil {
		return err
	}
	if len(header) == 0 {
		return errors.New("empty header")
	}
	return appendUniqueBookRow("Contents", "contents.book.idx", noteID, header, row)
}

func AppendBookCreator(userID string, creator any) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	header, row, err := toTabular(creator)
	if err != nil {
		return err
	}
	if len(header) == 0 {
		return errors.New("empty header")
	}
	return appendUniqueBookRow("Creators", "creators.book.idx", userID, header, row)
}

func AppendUniqueBookSheetRows(sheet string, indexFilename string, items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	if err := os.MkdirAll(PlatformDir(), 0755); err != nil {
		return 0, err
	}
	indexPath := filepath.Join(PlatformDir(), indexFilename)
	seen, err := loadIndex(indexPath)
	if err != nil {
		return 0, err
	}

	rows := make([][]string, 0, len(items))
	newKeys := make([]string, 0, len(items))
	for _, item := range items {
		k, err := keyFn(item)
		if err != nil {
			return 0, err
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		r, err := rowFn(item)
		if err != nil {
			return 0, err
		}
		seen[k] = struct{}{}
		rows = append(rows, r)
		newKeys = append(newKeys, k)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	bookMu.Lock()
	defer bookMu.Unlock()

	path := workbookPath()
	sheets := []string{"Contents", "Comments", "Creators"}
	found := false
	for _, s := range sheets {
		if s == sheet {
			found = true
			break
		}
	}
	if !found {
		sheets = append(sheets, sheet)
	}
	f, err := openOrCreateBook(path, sheets)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	if err := ensureHeader(f, sheet, header); err != nil {
		return 0, err
	}

	existing, err := f.GetRows(sheet)
	if err != nil {
		return 0, err
	}
	nextRow := len(existing) + 1
	if nextRow < 2 {
		nextRow = 2
	}

	for _, r := range rows {
		if err := writeRow(f, sheet, nextRow, r); err != nil {
			return 0, err
		}
		applyAutoWidth(f, sheet, header, r)
		nextRow++
	}

	if err := f.SaveAs(path); err != nil {
		return 0, err
	}
	if err := appendIndex(indexPath, newKeys); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func appendUniqueBookRow(sheet string, indexFilename string, key string, header []string, row []string) error {
	if err := os.MkdirAll(PlatformDir(), 0755); err != nil {
		return err
	}
	indexPath := filepath.Join(PlatformDir(), indexFilename)
	seen, err := loadIndex(indexPath)
	if err != nil {
		return err
	}
	if _, ok := seen[key]; ok {
		return nil
	}

	bookMu.Lock()
	defer bookMu.Unlock()

	path := workbookPath()
	sheets := []string{"Contents", "Comments", "Creators"}
	found := false
	for _, s := range sheets {
		if s == sheet {
			found = true
			break
		}
	}
	if !found {
		sheets = append(sheets, sheet)
	}
	f, err := openOrCreateBook(path, sheets)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := ensureHeader(f, sheet, header); err != nil {
		return err
	}

	existing, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	nextRow := len(existing) + 1
	if nextRow < 2 {
		nextRow = 2
	}
	if err := writeRow(f, sheet, nextRow, row); err != nil {
		return err
	}
	applyAutoWidth(f, sheet, header, row)

	if err := f.SaveAs(path); err != nil {
		return err
	}
	return appendIndex(indexPath, []string{key})
}

func openOrCreateBook(path string, sheets []string) (*excelize.File, error) {
	if _, err := os.Stat(path); err == nil {
		f, err := excelize.OpenFile(path)
		if err != nil {
			return nil, err
		}
		for _, sh := range sheets {
			ensureSheet(f, sh)
		}
		return f, nil
	}
	f := excelize.NewFile()
	for _, sh := range sheets {
		ensureSheet(f, sh)
	}
	if idx := f.GetActiveSheetIndex(); idx >= 0 {
		f.SetActiveSheet(idx)
	}
	return f, nil
}

func ensureSheet(f *excelize.File, sheet string) {
	if f == nil || strings.TrimSpace(sheet) == "" {
		return
	}
	for _, s := range f.GetSheetList() {
		if s == sheet {
			return
		}
	}
	f.NewSheet(sheet)
	if idx, err := f.GetSheetIndex("Sheet1"); err == nil && idx != -1 && sheet != "Sheet1" {
		_ = f.DeleteSheet("Sheet1")
	}
}

func ensureHeader(f *excelize.File, sheet string, header []string) error {
	existing, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		if err := writeRow(f, sheet, 1, header); err != nil {
			return err
		}
		applyHeaderStyle(f, sheet, len(header))
		applyAutoWidth(f, sheet, header, header)
		return nil
	}
	if len(existing) >= 1 && !sameHeader(existing[0], header) {
		return fmt.Errorf("xlsx header mismatch for sheet %s", sheet)
	}
	return nil
}

func applyHeaderStyle(f *excelize.File, sheet string, cols int) {
	if f == nil || cols <= 0 {
		return
	}
	styleID, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})
	if err != nil {
		return
	}
	start, _ := excelize.CoordinatesToCellName(1, 1)
	end, _ := excelize.CoordinatesToCellName(cols, 1)
	_ = f.SetCellStyle(sheet, start, end, styleID)
	_ = f.SetRowHeight(sheet, 1, 20)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       true,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
}

func applyAutoWidth(f *excelize.File, sheet string, header []string, row []string) {
	if f == nil {
		return
	}
	n := len(header)
	if len(row) > n {
		n = len(row)
	}
	for i := 0; i < n; i++ {
		maxLen := 0
		if i < len(header) {
			maxLen = len([]rune(header[i]))
		}
		if i < len(row) {
			l := len([]rune(row[i]))
			if l > maxLen {
				maxLen = l
			}
		}
		w := float64(maxLen + 2)
		if w < 10 {
			w = 10
		}
		if w > 60 {
			w = 60
		}
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			continue
		}
		_ = f.SetColWidth(sheet, col, col, w)
	}
}
