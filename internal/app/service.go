package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

type Service struct {
	store *Store
}

func NewService(dbPath string) (*Service, error) {
	store, err := OpenStore(dbPath)
	if err != nil {
		return nil, err
	}
	return &Service{store: store}, nil
}

func (s *Service) Close() error { return s.store.Close() }

func (s *Service) Index(ctx context.Context, path string, opt IndexOptions) (IndexResult, error) {
	start := time.Now()
	repoPath, err := filepath.Abs(path)
	if err != nil {
		return IndexResult{}, err
	}
	repoID, err := s.store.upsertRepo(ctx, repoPath)
	if err != nil {
		return IndexResult{}, err
	}
	if err := s.store.clearDiagnostics(ctx, repoID); err != nil {
		return IndexResult{}, err
	}
	existing, err := s.store.listIndexedFiles(ctx, repoID)
	if err != nil {
		return IndexResult{}, err
	}

	ig, _ := loadGitignore(repoPath, opt.NoGitignore)
	res := IndexResult{RepoPath: repoPath}
	count := 0
	seen := map[string]struct{}{}
	err = filepath.WalkDir(repoPath, func(fullPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(repoPath, fullPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".direnv" {
				return filepath.SkipDir
			}
			if rel != "." && ig != nil && (ig.MatchesPath(rel) || ig.MatchesPath(rel+"/")) {
				d := Diagnostic{Code: "SKIPPED_IGNORED", Path: rel}
				res.FilesSkipped = append(res.FilesSkipped, d)
				_ = s.store.addDiagnostic(ctx, repoID, d)
				return filepath.SkipDir
			}
			return nil
		}
		if opt.MaxFiles > 0 && count >= opt.MaxFiles {
			return io.EOF
		}
		if ig != nil && ig.MatchesPath(rel) {
			d := Diagnostic{Code: "SKIPPED_IGNORED", Path: rel}
			res.FilesSkipped = append(res.FilesSkipped, d)
			_ = s.store.addDiagnostic(ctx, repoID, d)
			return nil
		}
		lang := DetectLanguage(fullPath)
		if lang == "" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if opt.MaxSize > 0 && info.Size() > opt.MaxSize {
			d := Diagnostic{Code: "SKIPPED_TOO_LARGE", Path: rel}
			res.FilesSkipped = append(res.FilesSkipped, d)
			_ = s.store.addDiagnostic(ctx, repoID, d)
			return nil
		}
		seen[rel] = struct{}{}
		if !opt.Force {
			if prev, ok := existing[rel]; ok && prev.SizeBytes == info.Size() && prev.ModTimeNS == info.ModTime().UnixNano() && prev.ParseStatus == "ok" {
				res.FilesUnchanged++
				return nil
			}
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		if isBinary(content) {
			delete(seen, rel)
			d := Diagnostic{Code: "SKIPPED_BINARY", Path: rel}
			res.FilesSkipped = append(res.FilesSkipped, d)
			_ = s.store.addDiagnostic(ctx, repoID, d)
			return nil
		}
		hash := sha256hex(content)
		if !opt.Force {
			if prev, ok := existing[rel]; ok && prev.Hash == hash && prev.ParseStatus == "ok" {
				res.FilesUnchanged++
				return nil
			}
		}
		parsed, detectedLang, err := ParseSymbols(fullPath, content)
		parseStatus := "ok"
		if err != nil {
			parseStatus = "parse_failed"
			d := Diagnostic{Code: "PARSE_FAILED", Path: rel, Message: err.Error()}
			res.Warnings = append(res.Warnings, d)
			_ = s.store.addDiagnostic(ctx, repoID, d)
			parsed = nil
			detectedLang = lang
		}
		for i := range parsed {
			parsed[i].Language = detectedLang
			parsed[i].FilePath = rel
			parsed[i].ID = symbolID(repoPath, rel, parsed[i])
			if parsed[i].QualifiedName == "" {
				parsed[i].QualifiedName = parsed[i].Name
			}
		}
		if err := s.store.replaceFileSymbols(ctx, repoID, rel, detectedLang, hash, info.Size(), info.ModTime().UnixNano(), parseStatus, parsed); err != nil {
			return err
		}
		count++
		res.FilesIndexed++
		res.FilesUpdated++
		res.SymbolsExtracted += len(parsed)
		return nil
	})
	if err != nil && err != io.EOF {
		return IndexResult{}, err
	}
	if err == nil || err == io.EOF {
		deleted, derr := s.store.deleteMissingFiles(ctx, repoID, seen)
		if derr != nil {
			return IndexResult{}, derr
		}
		res.FilesDeleted = deleted
	}
	res.DurationMS = time.Since(start).Milliseconds()
	return res, nil
}

func (s *Service) SearchSymbols(ctx context.Context, opt SearchSymbolOptions) (SymbolSearchResult, error) {
	repoPath, err := currentRepoRoot()
	if err != nil {
		return SymbolSearchResult{}, err
	}
	items, err := s.store.searchSymbols(ctx, repoPath, opt)
	if err != nil {
		return SymbolSearchResult{}, err
	}
	result := SymbolSearchResult{Results: make([]SymbolSearchItem, 0, len(items))}
	for _, item := range items {
		result.Results = append(result.Results, SymbolSearchItem{
			ID:            item.ID,
			Name:          item.Name,
			QualifiedName: item.QualifiedName,
			Kind:          item.Kind,
			FilePath:      item.FilePath,
			Line:          item.StartLine,
			Signature:     item.Signature,
			Score:         item.Score,
		})
	}
	return result, nil
}

func (s *Service) SearchText(ctx context.Context, opt SearchTextOptions) (TextSearchResult, error) {
	_ = ctx
	repoPath, err := currentRepoRoot()
	if err != nil {
		return TextSearchResult{}, err
	}
	query := strings.ToLower(opt.Query)
	if opt.Limit <= 0 {
		opt.Limit = 20
	}
	var results []TextSearchItem
	_ = filepath.WalkDir(repoPath, func(fullPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || len(results) >= opt.Limit {
			return walkErr
		}
		lang := DetectLanguage(fullPath)
		if lang == "" || (opt.Lang != "" && lang != opt.Lang) {
			return nil
		}
		content, err := os.ReadFile(fullPath)
		if err != nil || isBinary(content) {
			return nil
		}
		rel, _ := filepath.Rel(repoPath, fullPath)
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		lineNo := 0
		for scanner.Scan() && len(results) < opt.Limit {
			lineNo++
			line := scanner.Text()
			idx := strings.Index(strings.ToLower(line), query)
			if idx >= 0 {
				results = append(results, TextSearchItem{FilePath: filepath.ToSlash(rel), Line: lineNo, Snippet: strings.TrimSpace(line), StartCol: idx + 1, EndCol: idx + len(opt.Query) + 1})
			}
		}
		return nil
	})
	return TextSearchResult{Results: results}, nil
}

func (s *Service) Outline(ctx context.Context, path string) (OutlineResult, error) {
	_, relPath, fullPath, err := resolvePathInRepo(path)
	if err != nil {
		return OutlineResult{}, err
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return OutlineResult{}, err
	}
	parsed, lang, err := ParseSymbols(fullPath, content)
	if err != nil {
		return OutlineResult{}, err
	}
	result := OutlineResult{FilePath: relPath, Language: lang}
	repoPath, err := currentRepoRoot()
	if err != nil {
		return OutlineResult{}, err
	}
	for _, sym := range parsed {
		sym.ID = symbolID(repoPath, relPath, sym)
		result.Symbols = append(result.Symbols, OutlineSymbol{ID: sym.ID, Name: sym.Name, Kind: sym.Kind, ParentID: sym.ParentID, StartLine: sym.StartLine, EndLine: sym.EndLine, Signature: sym.Signature, Language: lang})
	}
	_ = ctx
	return result, nil
}

func (s *Service) ShowSymbol(ctx context.Context, id string, contextLines int) (ShowSymbolResult, error) {
	rec, err := s.store.getSymbol(ctx, id)
	if err != nil {
		return ShowSymbolResult{}, err
	}
	fullPath := filepath.Join(rec.RepoPath, filepath.FromSlash(rec.FilePath))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return ShowSymbolResult{}, err
	}
	startLine := rec.StartLine - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := rec.EndLine + contextLines
	chunk, _, _, err := SliceLines(string(content), startLine, endLine)
	if err != nil {
		return ShowSymbolResult{}, err
	}
	return ShowSymbolResult{ID: rec.ID, Name: rec.Name, Kind: rec.Kind, FilePath: rec.FilePath, Language: rec.Language, QualifiedName: rec.QualifiedName, Signature: rec.Signature, StartLine: rec.StartLine, EndLine: rec.EndLine, Content: chunk}, nil
}

func (s *Service) ShowFile(ctx context.Context, path string, startLine, endLine int) (ShowFileResult, error) {
	_, relPath, fullPath, err := resolvePathInRepo(path)
	if err != nil {
		return ShowFileResult{}, err
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return ShowFileResult{}, err
	}
	chunk, startLine, endLine, err := SliceLines(string(content), startLine, endLine)
	if err != nil {
		return ShowFileResult{}, err
	}
	_ = ctx
	return ShowFileResult{FilePath: relPath, StartLine: startLine, EndLine: endLine, Content: chunk}, nil
}

func currentRepoRoot() (string, error) {
	return os.Getwd()
}

func resolvePathInRepo(path string) (repoPath, relPath, fullPath string, err error) {
	repoPath, err = currentRepoRoot()
	if err != nil {
		return "", "", "", err
	}
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(repoPath, path)
	}
	repoPath, relPath, err = repoAndRel(repoPath, fullPath)
	if err != nil {
		return "", "", "", ErrNotFound("FILE_NOT_INDEXED", err.Error())
	}
	fullPath = filepath.Join(repoPath, filepath.FromSlash(relPath))
	return repoPath, relPath, fullPath, nil
}

func loadGitignore(repoPath string, disabled bool) (*ignore.GitIgnore, error) {
	if disabled {
		return nil, nil
	}
	path := filepath.Join(repoPath, ".gitignore")
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	return ignore.CompileIgnoreFile(path)
}

func isBinary(content []byte) bool {
	limit := len(content)
	if limit > 1024 {
		limit = 1024
	}
	for i := 0; i < limit; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

func sha256hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func symbolID(repoPath, relPath string, sym Symbol) string {
	name := sym.QualifiedName
	if name == "" {
		name = sym.Name
	}
	repoKey := sha256hex([]byte(repoPath))[:12]
	return repoKey + ":" + filepath.ToSlash(relPath) + "::" + name + "#" + sym.Kind + ":" + strconv.Itoa(sym.StartLine)
}
