package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func AppendUniqueXLSX(dir, dataFilename, indexFilename string, items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	indexPath := filepath.Join(dir, indexFilename)
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

	path := filepath.Join(dir, dataFilename)
	f, sheet, err := openOrCreateWorkbook(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	existing, err := f.GetRows(sheet)
	if err != nil {
		return 0, err
	}

	nextRow := len(existing) + 1
	if len(existing) == 0 {
		if err := writeRow(f, sheet, 1, header); err != nil {
			return 0, err
		}
		nextRow = 2
	} else {
		if !sameHeader(existing[0], header) {
			return 0, fmt.Errorf("xlsx header mismatch for %s", dataFilename)
		}
	}

	for _, r := range rows {
		if err := writeRow(f, sheet, nextRow, r); err != nil {
			return 0, err
		}
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
