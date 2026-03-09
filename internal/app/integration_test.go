package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// newTestService creates a Service backed by a fresh SQLite DB in a temp dir.
func newTestService(t *testing.T) (*Service, string) {
	t.Helper()

	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")

	svc, err := NewService(dbPath)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	// Ensure DB is closed at the end of the test.
	t.Cleanup(func() { _ = svc.Close() })

	return svc, dbPath
}

// fixtureRepo returns the absolute path to the basic test fixture repository.
func fixtureRepo(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	path := filepath.Join(cwd, "..", "..", "tests", "testdata", "basicrepo")
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	return abs
}

// copyDir copies a directory tree from src to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", dst, err)
	}

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, info.Mode()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("copyDir %s -> %s: %v", src, dst, err)
	}
}

func repoIDForPath(t *testing.T, db *sql.DB, repoPath string) int64 {
	t.Helper()

	var id int64
	if err := db.QueryRow(`SELECT id FROM repos WHERE path = ?`, repoPath).Scan(&id); err != nil {
		t.Fatalf("lookup repo id for %s: %v", repoPath, err)
	}
	return id
}

func TestIndexStoresFilesAndSymbols(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)

	res, err := svc.Index(ctx, repoPath, IndexOptions{})
	if err != nil {
		t.Fatalf("Index error: %v", err)
	}

	if res.FilesIndexed != 3 || res.FilesUpdated != 3 || res.FilesUnchanged != 0 {
		t.Fatalf("unexpected index result: %+v", res)
	}

	// Validate DB contents.
	repoID := repoIDForPath(t, svc.store.db, repoPath)

	var fileCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM files WHERE repo_id = ?`, repoID).Scan(&fileCount); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if fileCount != 3 {
		t.Fatalf("expected 3 files, got %d", fileCount)
	}

	var symbolCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE repo_id = ?`, repoID).Scan(&symbolCount); err != nil {
		t.Fatalf("count symbols: %v", err)
	}
	if symbolCount < 6 {
		t.Fatalf("expected at least 6 symbols, got %d", symbolCount)
	}

	var okFiles int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM files WHERE repo_id = ? AND parse_status = 'ok'`, repoID).Scan(&okFiles); err != nil {
		t.Fatalf("count ok files: %v", err)
	}
	if okFiles != 3 {
		t.Fatalf("expected 3 ok files, got %d", okFiles)
	}
}

func TestIncrementalReindexSkipsUnchanged(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)

	first, err := svc.Index(ctx, repoPath, IndexOptions{})
	if err != nil {
		t.Fatalf("first Index error: %v", err)
	}
	if first.FilesIndexed != 3 || first.FilesUpdated != 3 || first.FilesUnchanged != 0 || first.FilesDeleted != 0 {
		t.Fatalf("unexpected first index result: %+v", first)
	}

	second, err := svc.Index(ctx, repoPath, IndexOptions{})
	if err != nil {
		t.Fatalf("second Index error: %v", err)
	}
	if second.FilesIndexed != 0 || second.FilesUpdated != 0 || second.FilesUnchanged != 3 || second.FilesDeleted != 0 {
		t.Fatalf("unexpected second index result: %+v", second)
	}
}

func TestDeleteMissingFileRemovesFromIndex(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	srcRepo := fixtureRepo(t)
	workdir := filepath.Join(t.TempDir(), "workrepo")
	copyDir(t, srcRepo, workdir)

	// Index the working copy.
	first, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("first Index error: %v", err)
	}
	if first.FilesIndexed != 3 || first.FilesDeleted != 0 {
		t.Fatalf("unexpected first index result: %+v", first)
	}

	workAbs, err := filepath.Abs(workdir)
	if err != nil {
		t.Fatalf("Abs(workdir): %v", err)
	}
	repoID := repoIDForPath(t, svc.store.db, workAbs)

	// Ensure helpers.py is present in files table.
	var count int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM files WHERE repo_id = ? AND path = 'src/helpers.py'`, repoID).Scan(&count); err != nil {
		t.Fatalf("query helpers.py presence: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected helpers.py to be indexed, got count=%d", count)
	}

	// Remove helpers.py from the worktree and reindex.
	if err := os.Remove(filepath.Join(workdir, "src", "helpers.py")); err != nil {
		t.Fatalf("Remove helpers.py: %v", err)
	}

	second, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("second Index error: %v", err)
	}
	if second.FilesDeleted != 1 {
		t.Fatalf("expected 1 file deleted, got %+v", second)
	}

	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM files WHERE repo_id = ? AND path = 'src/helpers.py'`, repoID).Scan(&count); err != nil {
		t.Fatalf("query helpers.py after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected helpers.py to be removed from index, got count=%d", count)
	}

	// Any symbols that used to belong to helpers.py should also be gone.
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM symbols s JOIN files f ON s.file_id = f.id WHERE f.repo_id = ? AND f.path = 'src/helpers.py'`, repoID).Scan(&count); err != nil {
		t.Fatalf("query helpers.py symbols after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected symbols for helpers.py to be removed, got count=%d", count)
	}

	// searchSymbols should not find auth_helper anymore.
	items, err := svc.store.searchSymbols(ctx, workAbs, SearchSymbolOptions{Query: "auth_helper", Limit: 10})
	if err != nil {
		t.Fatalf("searchSymbols error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no results for auth_helper after deletion, got %d", len(items))
	}
}

func TestSearchSymbolsAndGetSymbolRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	items, err := svc.store.searchSymbols(ctx, repoPath, SearchSymbolOptions{
		Query: "Login",
		Kind:  "function",
		Lang:  "go",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("searchSymbols error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one Login symbol, got 0")
	}

	var target storedSymbol
	found := false
	for _, it := range items {
		if it.FilePath == "src/auth.go" {
			found = true
			target = it
			break
		}
	}
	if !found {
		t.Fatalf("expected a Login symbol in src/auth.go, got: %+v", items)
	}

	rec, err := svc.store.getSymbol(ctx, target.ID)
	if err != nil {
		t.Fatalf("getSymbol error: %v", err)
	}
	if rec.ID != target.ID {
		t.Fatalf("ID mismatch: %q vs %q", rec.ID, target.ID)
	}
	if rec.FilePath != target.FilePath {
		t.Fatalf("file path mismatch: %q vs %q", rec.FilePath, target.FilePath)
	}
	if rec.RepoPath != repoPath {
		t.Fatalf("repo path mismatch: %q vs %q", rec.RepoPath, repoPath)
	}

	// Non-existent ID should yield a coded SYMBOL_NOT_FOUND error.
	_, err = svc.store.getSymbol(ctx, "no-such-id")
	if err == nil {
		t.Fatalf("expected error for missing symbol id")
	}
	ce, ok := err.(*CodedError)
	if !ok {
		t.Fatalf("expected *CodedError, got %T", err)
	}
	if ce.Code != "SYMBOL_NOT_FOUND" {
		t.Fatalf("expected code SYMBOL_NOT_FOUND, got %q", ce.Code)
	}
}

func TestDiagnosticsPersistAndClear(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	srcRepo := fixtureRepo(t)
	workdir := filepath.Join(t.TempDir(), "workrepo-diags")
	copyDir(t, srcRepo, workdir)

	workAbs, err := filepath.Abs(workdir)
	if err != nil {
		t.Fatalf("Abs(workdir): %v", err)
	}

	// First index with .gitignore present -> expect SKIPPED_IGNORED diagnostics.
	if _, err := svc.Index(ctx, workdir, IndexOptions{}); err != nil {
		t.Fatalf("first Index error: %v", err)
	}

	repoID := repoIDForPath(t, svc.store.db, workAbs)

	var diagCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM diagnostics WHERE repo_id = ?`, repoID).Scan(&diagCount); err != nil {
		t.Fatalf("count diagnostics: %v", err)
	}
	if diagCount == 0 {
		t.Fatalf("expected at least one diagnostic for ignored paths, got 0")
	}

	var ignoredCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM diagnostics WHERE repo_id = ? AND code = 'SKIPPED_IGNORED'`, repoID).Scan(&ignoredCount); err != nil {
		t.Fatalf("count SKIPPED_IGNORED diagnostics: %v", err)
	}
	if ignoredCount == 0 {
		t.Fatalf("expected at least one SKIPPED_IGNORED diagnostic, got 0")
	}

	// Remove .gitignore so subsequent indexing should not emit SKIPPED_IGNORED.
	if err := os.Remove(filepath.Join(workdir, ".gitignore")); err != nil {
		t.Fatalf("Remove .gitignore: %v", err)
	}

	if _, err := svc.Index(ctx, workdir, IndexOptions{}); err != nil {
		t.Fatalf("second Index error: %v", err)
	}

	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM diagnostics WHERE repo_id = ?`, repoID).Scan(&diagCount); err != nil {
		t.Fatalf("count diagnostics after second index: %v", err)
	}
	if diagCount != 0 {
		t.Fatalf("expected diagnostics to be cleared when conditions change, got %d rows", diagCount)
	}
}
