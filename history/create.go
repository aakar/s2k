package history

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

var schema = sqlitemigration.Schema{
	Migrations: []string{
		`CREATE TABLE "identifiers" (
			"value" TEXT NOT NULL UNIQUE,
			PRIMARY KEY("value")
		);`,
		`CREATE TABLE "steps" (
			"step_id"     INTEGER NOT NULL UNIQUE,
			"source"      TEXT NOT NULL,
			"destination" TEXT NOT NULL,
			"created"     INTEGER NOT NULL, -- Unix timestamp (epoch seconds)
			PRIMARY KEY("step_id" AUTOINCREMENT)
		);`,
		`CREATE TABLE "objects" (
			"step_id" INTEGER NOT NULL,
			"path"    TEXT NOT NULL,
			"data"    JSON,
			PRIMARY KEY("step_id","path"),
			FOREIGN KEY(step_id) REFERENCES steps(step_id)
		);`,
	},
}

func Create(path string, log *zap.Logger, values ...string) error {
	log = log.Named("history-migration")

	// async - will return immediately
	pool := sqlitemigration.NewPool(path, schema, sqlitemigration.Options{
		Flags: sqlite.OpenReadWrite | sqlite.OpenCreate,
		PrepareConn: func(conn *sqlite.Conn) error {
			// Enable foreign keys. See https://sqlite.org/foreignkeys.html
			return sqlitex.ExecuteTransient(conn, "PRAGMA foreign_keys = ON;", nil)
		},
		OnError: func(e error) {
			log.Error("Problems creating history database", zap.String("path", path), zap.Error(e))
		},
	})
	defer pool.Close()

	// Get a connection. This blocks until the migration completes.
	conn, err := pool.Get(context.TODO())
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	for _, value := range values {
		if err := sqlitex.Execute(conn, `INSERT INTO identifiers (value) VALUES (?);`, &sqlitex.ExecOptions{
			Args: []any{value},
		}); err != nil {
			return fmt.Errorf("unable to save identifier value '%s' in history: %w", value, err)
		}
	}

	return nil
}
