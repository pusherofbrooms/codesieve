package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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

	normalized, err := normalizeRepoPath(repoPath)
	if err != nil {
		t.Fatalf("normalize repo path %s: %v", repoPath, err)
	}

	var id int64
	if err := db.QueryRow(`SELECT id FROM repos WHERE path = ?`, normalized).Scan(&id); err != nil {
		t.Fatalf("lookup repo id for %s: %v", normalized, err)
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

func TestIndexPersistsRunStats(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("first Index error: %v", err)
	}
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("second Index error: %v", err)
	}

	repoID := repoIDForPath(t, svc.store.db, repoPath)

	var runCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM index_runs WHERE repo_id = ?`, repoID).Scan(&runCount); err != nil {
		t.Fatalf("count index_runs: %v", err)
	}
	if runCount != 2 {
		t.Fatalf("expected 2 index_runs rows, got %d", runCount)
	}

	var status string
	var filesIndexed, filesUpdated, filesUnchanged, filesDeleted, filesSkipped int
	var symbolsExtracted, warningsCount int
	var durationMS int64
	if err := svc.store.db.QueryRow(`SELECT status, files_indexed, files_updated, files_unchanged, files_deleted, files_skipped, symbols_extracted, warnings_count, duration_ms
		FROM index_runs
		WHERE repo_id = ?
		ORDER BY id DESC
		LIMIT 1`, repoID).Scan(&status, &filesIndexed, &filesUpdated, &filesUnchanged, &filesDeleted, &filesSkipped, &symbolsExtracted, &warningsCount, &durationMS); err != nil {
		t.Fatalf("query latest index_run: %v", err)
	}

	if status != "success" {
		t.Fatalf("expected status=success, got %q", status)
	}
	if filesIndexed != 0 || filesUpdated != 0 || filesUnchanged != 3 || filesDeleted != 0 {
		t.Fatalf("unexpected latest run file stats: indexed=%d updated=%d unchanged=%d deleted=%d", filesIndexed, filesUpdated, filesUnchanged, filesDeleted)
	}
	if filesSkipped != 0 {
		t.Fatalf("expected files_skipped=0 for fixture, got %d", filesSkipped)
	}
	if symbolsExtracted != 0 {
		t.Fatalf("expected symbols_extracted=0 for unchanged reindex, got %d", symbolsExtracted)
	}
	if warningsCount != 0 {
		t.Fatalf("expected warnings_count=0, got %d", warningsCount)
	}
	if durationMS < 0 {
		t.Fatalf("expected non-negative duration_ms, got %d", durationMS)
	}
}

func TestIncrementalReindexSkipsUnchangedParseFailures(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	workdir := filepath.Join(t.TempDir(), "workrepo-parse-fail")
	if err := os.MkdirAll(filepath.Join(workdir, "src"), 0o755); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	broken := "package src\n\nfunc Broken( {\n"
	if err := os.WriteFile(filepath.Join(workdir, "src", "broken.go"), []byte(broken), 0o644); err != nil {
		t.Fatalf("WriteFile broken.go: %v", err)
	}

	first, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("first Index error: %v", err)
	}
	if first.FilesUpdated != 1 || first.FilesUnchanged != 0 {
		t.Fatalf("unexpected first index result: %+v", first)
	}
	if len(first.Warnings) == 0 || first.Warnings[0].Code != "PARSE_FAILED" {
		t.Fatalf("expected PARSE_FAILED warning on first index, got %+v", first.Warnings)
	}

	second, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("second Index error: %v", err)
	}
	if second.FilesUpdated != 0 || second.FilesUnchanged != 1 {
		t.Fatalf("expected unchanged parse-failed file to be skipped, got %+v", second)
	}
	if len(second.Warnings) == 0 || second.Warnings[0].Code != "PARSE_FAILED" {
		t.Fatalf("expected cached PARSE_FAILED warning on second index, got %+v", second.Warnings)
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
	expectedRepoPath, err := normalizeRepoPath(repoPath)
	if err != nil {
		t.Fatalf("normalize repo path: %v", err)
	}
	if rec.RepoPath != expectedRepoPath {
		t.Fatalf("repo path mismatch: %q vs %q", rec.RepoPath, expectedRepoPath)
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

func TestSearchTextUsesIndexedContent(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	srcRepo := fixtureRepo(t)
	workdir := filepath.Join(t.TempDir(), "workrepo-text-indexed")
	copyDir(t, srcRepo, workdir)

	workReal, err := filepath.EvalSymlinks(workdir)
	if err != nil {
		t.Fatalf("EvalSymlinks(workdir): %v", err)
	}
	if _, err := svc.Index(ctx, workReal, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(workReal); err != nil {
		t.Fatalf("Chdir(%s): %v", workReal, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// Mutate an indexed file after indexing. Search results should still reflect indexed content.
	mutated := "export class Client {\n  login(token: string) {\n    return token;\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(workdir, "src", "client.ts"), []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile mutated client.ts: %v", err)
	}

	res, err := svc.SearchText(ctx, SearchTextOptions{Query: "AUTH_HEADER", Limit: 10})
	if err != nil {
		t.Fatalf("SearchText error: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatalf("expected indexed AUTH_HEADER match, got none")
	}

	// Add a new unindexed file; it must not appear until reindex.
	if err := os.WriteFile(filepath.Join(workdir, "src", "late_added.go"), []byte("package src\n\nconst LATE_ADDED_TOKEN = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile late_added.go: %v", err)
	}
	res, err = svc.SearchText(ctx, SearchTextOptions{Query: "LATE_ADDED_TOKEN", Limit: 10})
	if err != nil {
		t.Fatalf("SearchText late-added error: %v", err)
	}
	if len(res.Results) != 0 {
		t.Fatalf("expected no matches from unindexed file, got %+v", res.Results)
	}
}

func TestSearchTextRegexAndContext(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Chdir(%s): %v", repoPath, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	res, err := svc.SearchText(ctx, SearchTextOptions{Query: `return\s+id;`, Regex: true, ContextLines: 1, Limit: 10})
	if err != nil {
		t.Fatalf("SearchText regex error: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatalf("expected regex match, got none")
	}

	item := res.Results[0]
	if item.FilePath != "src/client.ts" {
		t.Fatalf("expected src/client.ts result, got %q", item.FilePath)
	}
	if len(item.ContextBefore) == 0 || !strings.Contains(item.ContextBefore[len(item.ContextBefore)-1], "fetchUser") {
		t.Fatalf("expected context_before to include fetchUser declaration, got %+v", item.ContextBefore)
	}
	if len(item.ContextAfter) == 0 || strings.TrimSpace(item.ContextAfter[0]) != "}" {
		t.Fatalf("expected context_after to include closing brace, got %+v", item.ContextAfter)
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

	// Ensure there is an ignored path in the copied fixture.
	// The repository fixture intentionally keeps ignored files out of version control,
	// so create one here to make this test deterministic in clean source builds.
	if err := os.MkdirAll(filepath.Join(workdir, "ignored"), 0o755); err != nil {
		t.Fatalf("MkdirAll ignored: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "ignored", "tmp.go"), []byte("package ignored\n\nfunc Hidden() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile ignored/tmp.go: %v", err)
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

	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM diagnostics WHERE repo_id = ? AND code = 'SKIPPED_IGNORED'`, repoID).Scan(&diagCount); err != nil {
		t.Fatalf("count SKIPPED_IGNORED diagnostics after second index: %v", err)
	}
	if diagCount != 0 {
		t.Fatalf("expected SKIPPED_IGNORED diagnostics to be cleared when conditions change, got %d rows", diagCount)
	}
}

func TestIndexSkipsSecretFiles(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	srcRepo := fixtureRepo(t)
	workdir := filepath.Join(t.TempDir(), "workrepo-secrets")
	copyDir(t, srcRepo, workdir)

	if err := os.WriteFile(filepath.Join(workdir, "src", "secrets.py"), []byte("def leaked():\n    return 'AKIA_TEST_VALUE'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile secrets.py: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, ".env"), []byte("API_KEY=AKIA_ENV_VALUE\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .env: %v", err)
	}

	res, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("Index error: %v", err)
	}

	secretSkips := 0
	for _, d := range res.FilesSkipped {
		if d.Code == "SKIPPED_SECRET" {
			secretSkips++
			if strings.Contains(d.Message, "AKIA_") {
				t.Fatalf("secret diagnostic message leaked secret-like value: %+v", d)
			}
		}
	}
	if secretSkips < 2 {
		t.Fatalf("expected at least two SKIPPED_SECRET diagnostics, got %d (%+v)", secretSkips, res.FilesSkipped)
	}

	workAbs, err := filepath.Abs(workdir)
	if err != nil {
		t.Fatalf("Abs(workdir): %v", err)
	}

	items, err := svc.store.searchSymbols(ctx, workAbs, SearchSymbolOptions{Query: "leaked", Limit: 10})
	if err != nil {
		t.Fatalf("searchSymbols error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no symbols from secret files, got %d", len(items))
	}

	repoID := repoIDForPath(t, svc.store.db, workAbs)
	var secretDiagCount int
	if err := svc.store.db.QueryRow(`SELECT COUNT(*) FROM diagnostics WHERE repo_id = ? AND code = 'SKIPPED_SECRET'`, repoID).Scan(&secretDiagCount); err != nil {
		t.Fatalf("count SKIPPED_SECRET diagnostics: %v", err)
	}
	if secretDiagCount < 2 {
		t.Fatalf("expected at least two SKIPPED_SECRET diagnostics in DB, got %d", secretDiagCount)
	}
}

func TestOutlineReturnsHierarchicalSymbols(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Chdir(%s): %v", repoPath, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	result, err := svc.Outline(ctx, "src/auth.go")
	if err != nil {
		t.Fatalf("Outline error: %v", err)
	}

	var authType OutlineSymbol
	foundType := false
	for _, sym := range result.Symbols {
		if sym.Name == "AuthService" && sym.Kind == "struct" {
			authType = sym
			foundType = true
			break
		}
	}
	if !foundType {
		t.Fatalf("expected top-level AuthService struct, got %+v", result.Symbols)
	}

	foundLogoutChild := false
	for _, child := range authType.Children {
		if child.Name == "Logout" && child.Kind == "method" {
			foundLogoutChild = true
			if child.ParentID != authType.ID {
				t.Fatalf("expected Logout parent_id=%q, got %q", authType.ID, child.ParentID)
			}
		}
	}
	if !foundLogoutChild {
		t.Fatalf("expected Logout nested under AuthService, got %+v", authType.Children)
	}

	for _, sym := range result.Symbols {
		if sym.Name == "Logout" {
			t.Fatalf("expected Logout to be nested, found top-level symbol: %+v", sym)
		}
	}
}

func TestRepoOutlineSummarizesIndexedRepo(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Chdir(%s): %v", repoPath, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	result, err := svc.RepoOutline(ctx)
	if err != nil {
		t.Fatalf("RepoOutline error: %v", err)
	}

	expectedRepoPath, err := normalizeRepoPath(repoPath)
	if err != nil {
		t.Fatalf("normalize repo path: %v", err)
	}
	if result.RepoPath != expectedRepoPath {
		t.Fatalf("expected RepoPath %q, got %q", expectedRepoPath, result.RepoPath)
	}
	if result.TotalFiles != 3 {
		t.Fatalf("expected TotalFiles=3, got %d", result.TotalFiles)
	}
	if result.TotalSymbols < 6 {
		t.Fatalf("expected TotalSymbols>=6, got %d", result.TotalSymbols)
	}
	if result.LanguageBreakdown["go"] != 1 || result.LanguageBreakdown["python"] != 1 || result.LanguageBreakdown["typescript"] != 1 {
		t.Fatalf("unexpected language breakdown: %+v", result.LanguageBreakdown)
	}
	if result.TopLevelDirectoryCounts["src"] != 3 {
		t.Fatalf("unexpected top-level directory counts: %+v", result.TopLevelDirectoryCounts)
	}
	if len(result.SymbolKindCounts) == 0 {
		t.Fatalf("expected non-empty symbol kind counts")
	}
	if result.IndexAgeSeconds < 0 {
		t.Fatalf("expected non-negative index age, got %d", result.IndexAgeSeconds)
	}
	if result.IndexedAt == "" {
		t.Fatalf("expected non-empty indexed_at")
	}
	if result.LatestIndexRun == nil {
		t.Fatalf("expected latest_index_run in repo outline")
	}
	if result.LatestIndexRun.Status != "success" {
		t.Fatalf("expected latest_index_run.status=success, got %+v", result.LatestIndexRun)
	}
	if result.LatestIndexRun.FilesIndexed != 3 || result.LatestIndexRun.FilesUpdated != 3 {
		t.Fatalf("unexpected latest_index_run file stats: %+v", result.LatestIndexRun)
	}
}

func TestRepoOutlineErrorsWhenRepoNotIndexed(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	workdir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir(%s): %v", workdir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	_, err = svc.RepoOutline(ctx)
	if err == nil {
		t.Fatalf("expected error for non-indexed repo")
	}
	ce, ok := err.(*CodedError)
	if !ok {
		t.Fatalf("expected *CodedError, got %T", err)
	}
	if ce.Code != "FILE_NOT_INDEXED" {
		t.Fatalf("expected FILE_NOT_INDEXED, got %q", ce.Code)
	}
}

func TestShowSymbolsIncludesPerIDErrors(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	repoPath := fixtureRepo(t)
	if _, err := svc.Index(ctx, repoPath, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	items, err := svc.store.searchSymbols(ctx, repoPath, SearchSymbolOptions{Query: "Login", Limit: 10})
	if err != nil {
		t.Fatalf("searchSymbols error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one symbol")
	}

	result, err := svc.ShowSymbols(ctx, []string{items[0].ID, "missing-symbol-id"})
	if err != nil {
		t.Fatalf("ShowSymbols error: %v", err)
	}
	if len(result.Symbols) != 1 {
		t.Fatalf("expected one resolved symbol, got %d", len(result.Symbols))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one per-id error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != "SYMBOL_NOT_FOUND" {
		t.Fatalf("expected SYMBOL_NOT_FOUND error code, got %+v", result.Errors[0])
	}
}

func TestShowSymbolVerifyDetectsDrift(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	srcRepo := fixtureRepo(t)
	workdir := filepath.Join(t.TempDir(), "workrepo-verify")
	copyDir(t, srcRepo, workdir)
	if _, err := svc.Index(ctx, workdir, IndexOptions{}); err != nil {
		t.Fatalf("Index error: %v", err)
	}

	workAbs, err := normalizeRepoPath(workdir)
	if err != nil {
		t.Fatalf("normalizeRepoPath(workdir): %v", err)
	}
	items, err := svc.store.searchSymbols(ctx, workAbs, SearchSymbolOptions{Query: "Login", Limit: 10})
	if err != nil {
		t.Fatalf("searchSymbols error: %v", err)
	}
	var loginID string
	for _, item := range items {
		if item.FilePath == "src/auth.go" && item.Name == "Login" {
			loginID = item.ID
			break
		}
	}
	if loginID == "" {
		t.Fatalf("expected Login symbol in src/auth.go, got %+v", items)
	}

	first, err := svc.ShowSymbol(ctx, loginID, 0, true)
	if err != nil {
		t.Fatalf("ShowSymbol verify=true first call: %v", err)
	}
	if first.Verification == nil || !first.Verification.Verified {
		t.Fatalf("expected verification success, got %+v", first.Verification)
	}

	mutated := "package main\n\ntype AuthService struct{}\n\nfunc LoginUser(user string) error {\n\treturn nil\n}\n\nfunc (a *AuthService) Logout() error {\n\treturn nil\n}\n\nconst AuthHeader = \"X-Auth-Header\"\n"
	if err := os.WriteFile(filepath.Join(workdir, "src", "auth.go"), []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile mutated auth.go: %v", err)
	}

	second, err := svc.ShowSymbol(ctx, loginID, 0, true)
	if err != nil {
		t.Fatalf("ShowSymbol verify=true second call: %v", err)
	}
	if second.Verification == nil || second.Verification.Verified {
		t.Fatalf("expected verification failure after drift, got %+v", second.Verification)
	}
}
