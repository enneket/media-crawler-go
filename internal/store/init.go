package store

import (
	"context"
	"time"
)

func Init(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch backendKind() {
	case backendSQLite:
		db, err := sqliteDB()
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	case backendMySQL:
		db, err := mysqlDB()
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	case backendPostgres:
		db, err := postgresDB()
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	case backendMongoDB:
		_, err := mongoClient()
		return err
	default:
		return nil
	}
}
