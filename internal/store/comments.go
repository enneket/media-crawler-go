package store

import "strings"

func AppendUniqueCommentsJSONL(noteID string, items []any, keyFn func(any) (string, error)) (int, error) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return 0, nil
	}
	n, err := AppendUniqueJSONL(NoteDir(noteID), "comments.jsonl", "comments.idx", items, keyFn)
	if err != nil {
		return n, err
	}
	if err := sqlInsertComments(noteID, items, keyFn); err != nil {
		return n, err
	}
	for _, it := range items {
		_ = pythonCompatAppendJSON("comments", it)
	}
	return n, nil
}

func AppendUniqueCommentsCSV(noteID string, items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return 0, nil
	}
	n, err := AppendUniqueCSV(NoteDir(noteID), "comments.csv", "comments.idx", items, keyFn, header, rowFn)
	if err != nil {
		return n, err
	}
	if err := sqlInsertComments(noteID, items, keyFn); err != nil {
		return n, err
	}
	return n, nil
}

func AppendUniqueCommentsXLSX(noteID string, items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return 0, nil
	}
	n, err := AppendUniqueXLSX(NoteDir(noteID), "comments.xlsx", "comments.idx", items, keyFn, header, rowFn)
	if err != nil {
		return n, err
	}
	if err := sqlInsertComments(noteID, items, keyFn); err != nil {
		return n, err
	}
	return n, nil
}

func AppendUniqueGlobalCommentsJSONL(items []any, keyFn func(any) (string, error)) (int, error) {
	n, err := AppendUniqueJSONL(PlatformDir(), "comments.jsonl", "comments.global.idx", items, keyFn)
	if err != nil {
		return n, err
	}
	for _, it := range items {
		_ = pythonCompatAppendJSON("comments", it)
	}
	return n, nil
}

func AppendUniqueGlobalCommentsCSV(items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	return AppendUniqueCSV(PlatformDir(), "comments.csv", "comments.global.idx", items, keyFn, header, rowFn)
}

func AppendUniqueGlobalCommentsXLSX(items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	return AppendUniqueXLSX(PlatformDir(), "comments.xlsx", "comments.global.idx", items, keyFn, header, rowFn)
}

func AppendUniqueGlobalCommentsBook(items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	return AppendUniqueBookSheetRows("Comments", "comments.book.idx", items, keyFn, header, rowFn)
}
