package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesFileAndRunsMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("expected database file to exist")
	}

	var tableName string
	err = d.DB.Get(&tableName, "SELECT name FROM sqlite_master WHERE type='table' AND name='user_integrations'")
	if err != nil {
		t.Fatalf("user_integrations table not found: %v", err)
	}
	if tableName != "user_integrations" {
		t.Fatalf("expected 'user_integrations', got %q", tableName)
	}
}
