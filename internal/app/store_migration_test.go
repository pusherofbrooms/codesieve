package app

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenStoreTracksSchemaVersionAndMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "schema.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore error: %v", err)
	}
	defer store.Close()

	var version int
	if err := store.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("expected user_version=%d, got %d", currentSchemaVersion, version)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != len(schemaMigrations) {
		t.Fatalf("expected %d schema_migrations rows, got %d", len(schemaMigrations), count)
	}
}

func TestOpenStoreMigratesLegacyFilesTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	legacySchema := `
CREATE TABLE repos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL UNIQUE,
  indexed_at TEXT NOT NULL
);
CREATE TABLE files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  language TEXT,
  hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  indexed_at TEXT NOT NULL,
  parse_status TEXT NOT NULL,
  UNIQUE(repo_id, path)
);
`
	if _, err := db.Exec(legacySchema); err != nil {
		db.Close()
		t.Fatalf("seed legacy schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore legacy db error: %v", err)
	}
	defer store.Close()

	hasModTime, err := tableHasColumn(store.db, "files", "mod_time_ns")
	if err != nil {
		t.Fatalf("check mod_time_ns column: %v", err)
	}
	if !hasModTime {
		t.Fatalf("expected files.mod_time_ns column after migration")
	}
	hasContent, err := tableHasColumn(store.db, "files", "content")
	if err != nil {
		t.Fatalf("check content column: %v", err)
	}
	if !hasContent {
		t.Fatalf("expected files.content column after migration")
	}
	hasParserVersion, err := tableHasColumn(store.db, "files", "parser_version")
	if err != nil {
		t.Fatalf("check parser_version column: %v", err)
	}
	if !hasParserVersion {
		t.Fatalf("expected files.parser_version column after migration")
	}

	var version int
	if err := store.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("expected user_version=%d after migration, got %d", currentSchemaVersion, version)
	}
}

func TestOpenStoreRejectsNewerSchemaVersion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "too-new.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 999`); err != nil {
		db.Close()
		t.Fatalf("set user_version: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	store, err := OpenStore(dbPath)
	if err == nil {
		store.Close()
		t.Fatalf("expected OpenStore to reject newer schema version")
	}
}

func tableHasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}
