package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func DefaultDBPath() (string, error) {
	if v := os.Getenv("CODESIEVE_DB_PATH"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codesieve", "index.db"), nil
}

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS repos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL UNIQUE,
  indexed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  language TEXT,
  hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  mod_time_ns INTEGER NOT NULL DEFAULT 0,
  indexed_at TEXT NOT NULL,
  parse_status TEXT NOT NULL,
  UNIQUE(repo_id, path)
);
CREATE TABLE IF NOT EXISTS symbols (
  id TEXT PRIMARY KEY,
  repo_id INTEGER NOT NULL,
  file_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  qualified_name TEXT,
  kind TEXT NOT NULL,
  parent_symbol_id TEXT,
  signature TEXT,
  documentation TEXT,
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  start_byte INTEGER NOT NULL,
  end_byte INTEGER NOT NULL,
  language TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS diagnostics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  path TEXT,
  code TEXT NOT NULL,
  message TEXT
);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_qname ON symbols(qualified_name);
CREATE INDEX IF NOT EXISTS idx_files_repo_path ON files(repo_id, path);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	_, err := s.db.Exec(`ALTER TABLE files ADD COLUMN mod_time_ns INTEGER NOT NULL DEFAULT 0`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	return nil
}

func (s *Store) upsertRepo(ctx context.Context, path string) (int64, error) {
	_, err := s.db.ExecContext(ctx, `INSERT INTO repos(path, indexed_at) VALUES(?, datetime('now')) ON CONFLICT(path) DO UPDATE SET indexed_at=datetime('now')`, path)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM repos WHERE path = ?`, path).Scan(&id)
	return id, err
}

func (s *Store) replaceFileSymbols(ctx context.Context, repoID int64, relPath, language, hash string, size int64, modTimeNS int64, parseStatus string, symbols []Symbol) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO files(repo_id, path, language, hash, size_bytes, mod_time_ns, indexed_at, parse_status) VALUES(?, ?, ?, ?, ?, ?, datetime('now'), ?)
		ON CONFLICT(repo_id, path) DO UPDATE SET language=excluded.language, hash=excluded.hash, size_bytes=excluded.size_bytes, mod_time_ns=excluded.mod_time_ns, indexed_at=datetime('now'), parse_status=excluded.parse_status`, repoID, relPath, language, hash, size, modTimeNS, parseStatus)
	if err != nil {
		return err
	}
	var fileID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM files WHERE repo_id = ? AND path = ?`, repoID, relPath).Scan(&fileID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM symbols WHERE file_id = ?`, fileID); err != nil {
		return err
	}
	for _, sym := range symbols {
		_, err := tx.ExecContext(ctx, `INSERT INTO symbols(id, repo_id, file_id, name, qualified_name, kind, parent_symbol_id, signature, documentation, start_line, end_line, start_byte, end_byte, language)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, sym.ID, repoID, fileID, sym.Name, sym.QualifiedName, sym.Kind, nullable(sym.ParentID), nullable(sym.Signature), nullable(sym.Documentation), sym.StartLine, sym.EndLine, sym.StartByte, sym.EndByte, sym.Language)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) clearDiagnostics(ctx context.Context, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM diagnostics WHERE repo_id = ?`, repoID)
	return err
}

func (s *Store) addDiagnostic(ctx context.Context, repoID int64, d Diagnostic) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO diagnostics(repo_id, path, code, message) VALUES(?, ?, ?, ?)`, repoID, nullable(d.Path), d.Code, nullable(d.Message))
	return err
}

type indexedFile struct {
	Path        string
	Hash        string
	SizeBytes   int64
	ModTimeNS   int64
	Language    string
	ParseStatus string
}

type storedSymbol struct {
	ID            string
	Name          string
	QualifiedName string
	Kind          string
	Signature     string
	FilePath      string
	StartLine     int
	EndLine       int
	Language      string
	Score         float64
}

func (s *Store) listIndexedFiles(ctx context.Context, repoID int64) (map[string]indexedFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, hash, size_bytes, mod_time_ns, COALESCE(language,''), parse_status FROM files WHERE repo_id = ?`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]indexedFile{}
	for rows.Next() {
		var item indexedFile
		if err := rows.Scan(&item.Path, &item.Hash, &item.SizeBytes, &item.ModTimeNS, &item.Language, &item.ParseStatus); err != nil {
			return nil, err
		}
		out[item.Path] = item
	}
	return out, rows.Err()
}

func (s *Store) deleteMissingFiles(ctx context.Context, repoID int64, seen map[string]struct{}) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, path FROM files WHERE repo_id = ?`, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	deleted := 0
	for rows.Next() {
		var fileID int64
		var path string
		if err := rows.Scan(&fileID, &path); err != nil {
			return 0, err
		}
		if _, ok := seen[path]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM symbols WHERE file_id = ?`, fileID); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, fileID); err != nil {
			return 0, err
		}
		deleted++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Store) searchSymbols(ctx context.Context, repoPath string, opt SearchSymbolOptions) ([]storedSymbol, error) {
	if opt.Limit <= 0 {
		opt.Limit = 20
	}
	qFold := strings.ToLower(opt.Query)
	pathLike := "%" + opt.PathSubstr + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, COALESCE(s.qualified_name,''), s.kind, COALESCE(s.signature,''), f.path, s.start_line, s.end_line, s.language
		FROM symbols s
		JOIN repos r ON r.id = s.repo_id
		JOIN files f ON f.id = s.file_id
		WHERE r.path = ?
		AND (lower(s.name) LIKE ? OR lower(s.qualified_name) LIKE ?)
		AND (? = '' OR s.language = ?)
		AND (? = '' OR s.kind = ?)
		AND (? = '' OR f.path LIKE ?)
		LIMIT ?`, repoPath, "%"+qFold+"%", "%"+qFold+"%", opt.Lang, opt.Lang, opt.Kind, opt.Kind, opt.PathSubstr, pathLike, opt.Limit*5)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedSymbol
	for rows.Next() {
		var item storedSymbol
		if err := rows.Scan(&item.ID, &item.Name, &item.QualifiedName, &item.Kind, &item.Signature, &item.FilePath, &item.StartLine, &item.EndLine, &item.Language); err != nil {
			return nil, err
		}
		item.Score = rankSymbol(opt, item)
		if item.Score > 0 {
			out = append(out, item)
		}
	}
	sortStoredSymbols(out)
	if len(out) > opt.Limit {
		out = out[:opt.Limit]
	}
	return out, rows.Err()
}

func sortStoredSymbols(items []storedSymbol) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Score > items[i].Score || (items[j].Score == items[i].Score && items[j].FilePath < items[i].FilePath) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func rankSymbol(opt SearchSymbolOptions, item storedSymbol) float64 {
	q := opt.Query
	if strings.TrimSpace(q) == "" {
		return 0
	}

	name := item.Name
	qname := item.QualifiedName
	container := ""
	if dot := strings.LastIndex(qname, "."); dot > 0 {
		container = qname[:dot]
	}

	// Case handling
	nameFold := strings.ToLower(name)
	qnameFold := strings.ToLower(qname)
	qFold := strings.ToLower(q)

	matchScore := 0.0

	// Helper for equality under case rules
	eq := func(a, b string) bool {
		if opt.CaseSensitive {
			return a == b
		}
		return strings.EqualFold(a, b)
	}

	// Name / qualified name matching tiers
	switch {
	case eq(name, q):
		matchScore = 100
	case eq(qname, q):
		matchScore = 95
	case !opt.CaseSensitive && strings.HasPrefix(nameFold, qFold):
		matchScore = 80
	case !opt.CaseSensitive && strings.HasPrefix(qnameFold, qFold):
		matchScore = 70
	case opt.CaseSensitive && strings.HasPrefix(name, q):
		matchScore = 80
	case opt.CaseSensitive && strings.HasPrefix(qname, q):
		matchScore = 70
	case !opt.CaseSensitive && strings.Contains(nameFold, qFold):
		matchScore = 60
	case !opt.CaseSensitive && strings.Contains(qnameFold, qFold):
		matchScore = 50
	case opt.CaseSensitive && strings.Contains(name, q):
		matchScore = 60
	case opt.CaseSensitive && strings.Contains(qname, q):
		matchScore = 50
	}

	if matchScore == 0 {
		return 0
	}

	score := matchScore

	// Kind weighting
	switch strings.ToLower(item.Kind) {
	case "function", "func", "method":
		score += 15
	case "class", "struct", "interface":
		score += 10
	case "enum", "type", "type_alias":
		score += 5
	}

	// Container/context hints
	if container != "" {
		if eq(container, q) {
			score += 10
		} else if !opt.CaseSensitive && strings.HasPrefix(strings.ToLower(container), qFold) {
			score += 5
		} else if opt.CaseSensitive && strings.HasPrefix(container, q) {
			score += 5
		}
	}

	// Path heuristics
	path := item.FilePath
	pathFold := strings.ToLower(path)
	if strings.HasPrefix(path, "vendor/") || strings.HasPrefix(path, "third_party/") {
		score -= 15
	}
	if strings.HasSuffix(path, "_test.go") || strings.Contains(pathFold, "/test/") || strings.Contains(pathFold, "/tests/") {
		score -= 10
	}
	if strings.HasPrefix(path, "src/") || strings.HasPrefix(path, "internal/") {
		score += 5
	}

	// Slight preference for earlier definitions
	score += 1.0 / float64(item.StartLine+1)
	return score
}

func (s *Store) listFileSymbols(ctx context.Context, repoPath, relPath string) ([]storedSymbol, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.name, COALESCE(s.qualified_name,''), s.kind, COALESCE(s.signature,''), f.path, s.start_line, s.end_line, s.language
		FROM symbols s
		JOIN repos r ON r.id = s.repo_id
		JOIN files f ON f.id = s.file_id
		WHERE r.path = ? AND f.path = ?
		ORDER BY s.start_line`, repoPath, relPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedSymbol
	for rows.Next() {
		var item storedSymbol
		if err := rows.Scan(&item.ID, &item.Name, &item.QualifiedName, &item.Kind, &item.Signature, &item.FilePath, &item.StartLine, &item.EndLine, &item.Language); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type symbolRecord struct {
	storedSymbol
	RepoPath string
}

func (s *Store) getSymbol(ctx context.Context, id string) (*symbolRecord, error) {
	var rec symbolRecord
	err := s.db.QueryRowContext(ctx, `SELECT s.id, s.name, COALESCE(s.qualified_name,''), s.kind, COALESCE(s.signature,''), f.path, s.start_line, s.end_line, s.language, r.path
		FROM symbols s
		JOIN files f ON f.id = s.file_id
		JOIN repos r ON r.id = s.repo_id
		WHERE s.id = ?`, id).Scan(&rec.ID, &rec.Name, &rec.QualifiedName, &rec.Kind, &rec.Signature, &rec.FilePath, &rec.StartLine, &rec.EndLine, &rec.Language, &rec.RepoPath)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("SYMBOL_NOT_FOUND", "no symbol matched the provided id")
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) repoSummary(ctx context.Context, repoPath string) (RepoOutlineResult, error) {
	var repoID int64
	var indexedAt string
	var ageSeconds int64
	err := s.db.QueryRowContext(ctx, `SELECT id, indexed_at, CAST(strftime('%s','now') AS INTEGER) - CAST(strftime('%s', indexed_at) AS INTEGER)
		FROM repos
		WHERE path = ?`, repoPath).Scan(&repoID, &indexedAt, &ageSeconds)
	if err == sql.ErrNoRows {
		return RepoOutlineResult{}, ErrNotFound("FILE_NOT_INDEXED", "repository is not indexed: run 'codesieve index .' first")
	}
	if err != nil {
		return RepoOutlineResult{}, err
	}

	result := RepoOutlineResult{
		RepoPath:                repoPath,
		LanguageBreakdown:       map[string]int{},
		TopLevelDirectoryCounts: map[string]int{},
		SymbolKindCounts:        map[string]int{},
		IndexedAt:               indexedAt,
		IndexAgeSeconds:         ageSeconds,
		Stale:                   ageSeconds > 24*60*60,
	}

	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE repo_id = ?`, repoID).Scan(&result.TotalFiles); err != nil {
		return RepoOutlineResult{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbols WHERE repo_id = ?`, repoID).Scan(&result.TotalSymbols); err != nil {
		return RepoOutlineResult{}, err
	}

	langRows, err := s.db.QueryContext(ctx, `SELECT COALESCE(language,''), COUNT(*) FROM files WHERE repo_id = ? GROUP BY language`, repoID)
	if err != nil {
		return RepoOutlineResult{}, err
	}
	defer langRows.Close()
	for langRows.Next() {
		var lang string
		var count int
		if err := langRows.Scan(&lang, &count); err != nil {
			return RepoOutlineResult{}, err
		}
		if strings.TrimSpace(lang) == "" {
			lang = "unknown"
		}
		result.LanguageBreakdown[lang] = count
	}
	if err := langRows.Err(); err != nil {
		return RepoOutlineResult{}, err
	}

	pathRows, err := s.db.QueryContext(ctx, `SELECT path FROM files WHERE repo_id = ?`, repoID)
	if err != nil {
		return RepoOutlineResult{}, err
	}
	defer pathRows.Close()
	for pathRows.Next() {
		var path string
		if err := pathRows.Scan(&path); err != nil {
			return RepoOutlineResult{}, err
		}
		top := topLevelSegment(path)
		result.TopLevelDirectoryCounts[top]++
	}
	if err := pathRows.Err(); err != nil {
		return RepoOutlineResult{}, err
	}

	kindRows, err := s.db.QueryContext(ctx, `SELECT kind, COUNT(*) FROM symbols WHERE repo_id = ? GROUP BY kind`, repoID)
	if err != nil {
		return RepoOutlineResult{}, err
	}
	defer kindRows.Close()
	for kindRows.Next() {
		var kind string
		var count int
		if err := kindRows.Scan(&kind, &count); err != nil {
			return RepoOutlineResult{}, err
		}
		result.SymbolKindCounts[kind] = count
	}
	if err := kindRows.Err(); err != nil {
		return RepoOutlineResult{}, err
	}

	return result, nil
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func topLevelSegment(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || path == "." {
		return "."
	}
	if idx := strings.Index(path, "/"); idx >= 0 {
		if idx == 0 {
			return "."
		}
		return path[:idx]
	}
	return "."
}

func repoAndRel(base, file string) (string, string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", "", err
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(absBase, absFile)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("path is outside repository: %s", file)
	}
	return absBase, filepath.ToSlash(rel), nil
}
