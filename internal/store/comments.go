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
	if err := sqliteInsertComments(noteID, items, keyFn); err != nil {
		return n, err
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
	if err := sqliteInsertComments(noteID, items, keyFn); err != nil {
		return n, err
	}
	return n, nil
}
