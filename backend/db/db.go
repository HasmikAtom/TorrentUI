package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Database struct {
	DB *sqlx.DB
}

func Open(path string) (*Database, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sqlx.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	d := &Database{DB: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) migrate() error {
	_, err := d.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_integrations (
			user_id      TEXT PRIMARY KEY,
			plex_enabled INTEGER NOT NULL DEFAULT 0,
			plex_token   TEXT    NOT NULL DEFAULT '',
			created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at   INTEGER NOT NULL DEFAULT (unixepoch())
		);
	`)
	return err
}
