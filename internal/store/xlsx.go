package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xuri/excelize/v2"
)

type XlsxStore struct {
	Dir string
	mu  sync.Mutex
}

func NewXlsxStore(dir string) *XlsxStore {
	return &XlsxStore{Dir: dir}
}

func (s *XlsxStore) Save(data interface{}, filename string) error {
	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, filename)

	header, row, err := toTabular(data)
	if err != nil {
		return err
	}
	if len(header) == 0 {
		return errors.New("empty header")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, sheet, err := openOrCreateWorkbook(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}

	nextRow := len(rows) + 1
	if len(rows) == 0 {
		if err := writeRow(f, sheet, 1, header); err != nil {
			return err
		}
		nextRow = 2
	} else if len(rows) >= 1 {
		existingHeader := rows[0]
		if !sameHeader(existingHeader, header) {
			return fmt.Errorf("xlsx header mismatch for %s", filename)
		}
	}

	if err := writeRow(f, sheet, nextRow, row); err != nil {
		return err
	}

	return f.SaveAs(path)
}

func openOrCreateWorkbook(path string) (*excelize.File, string, error) {
	if _, err := os.Stat(path); err == nil {
		f, err := excelize.OpenFile(path)
		if err != nil {
			return nil, "", err
		}
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			sheet := "Sheet1"
			f.NewSheet(sheet)
			f.DeleteSheet("Sheet1")
			return f, sheet, nil
		}
		return f, sheets[0], nil
	}

	f := excelize.NewFile()
	sheets := f.GetSheetList()
	sheet := "Sheet1"
	if len(sheets) > 0 {
		sheet = sheets[0]
	} else {
		f.NewSheet(sheet)
	}
	return f, sheet, nil
}

func writeRow(f *excelize.File, sheet string, rowIndex int, values []string) error {
	for i, v := range values {
		cell, err := excelize.CoordinatesToCellName(i+1, rowIndex)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			return err
		}
	}
	return nil
}

func sameHeader(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func toTabular(data any) (header []string, row []string, err error) {
	if v, ok := data.(CSVer); ok {
		return v.CSVHeader(), v.ToCSV(), nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}
	return []string{"json"}, []string{string(b)}, nil
}

