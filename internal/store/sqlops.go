package store

func sqlUpsertNote(noteID string, note any) error {
	switch backendKind() {
	case backendSQLite:
		return sqliteUpsertNote(noteID, note)
	case backendMySQL:
		return mysqlUpsertNote(noteID, note)
	case backendPostgres:
		return postgresUpsertNote(noteID, note)
	case backendMongoDB:
		return mongoUpsertNote(noteID, note)
	default:
		return nil
	}
}

func sqlUpsertCreator(creatorID string, data any) error {
	switch backendKind() {
	case backendSQLite:
		return sqliteUpsertCreator(creatorID, data)
	case backendMySQL:
		return mysqlUpsertCreator(creatorID, data)
	case backendPostgres:
		return postgresUpsertCreator(creatorID, data)
	case backendMongoDB:
		return mongoUpsertCreator(creatorID, data)
	default:
		return nil
	}
}

func sqlInsertComments(noteID string, items []any, keyFn func(any) (string, error)) error {
	switch backendKind() {
	case backendSQLite:
		return sqliteInsertComments(noteID, items, keyFn)
	case backendMySQL:
		return mysqlInsertComments(noteID, items, keyFn)
	case backendPostgres:
		return postgresInsertComments(noteID, items, keyFn)
	case backendMongoDB:
		return mongoInsertComments(noteID, items, keyFn)
	default:
		return nil
	}
}
