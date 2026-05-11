package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchemaV1(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	assertTableExists(t, db, "migrations")
	assertTableExists(t, db, "projects")
	assertTableExists(t, db, "agents")
	assertTableExists(t, db, "tasks")
	assertTableExists(t, db, "sessions")

	if count := migrationCount(t, db, migrationSchemaV1); count != 1 {
		t.Fatalf("migration count = %d, want 1", count)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sessions.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open failed: %v", err)
	}
	db.Close()

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	defer db.Close()

	if count := migrationCount(t, db, migrationSchemaV1); count != 1 {
		t.Fatalf("migration count after reopen = %d, want 1", count)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for %q failed: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("table %q count = %d, want 1", table, count)
	}
}

func migrationCount(t *testing.T, db *sql.DB, id string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM migrations WHERE id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("count migration %q failed: %v", id, err)
	}

	return count
}
