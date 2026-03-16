package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	repoPath, err := normalizeRepoPath(path)
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
	pending := make([]FileIndexUpdate, 0, 128)
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		if err := s.store.replaceFilesSymbolsBatch(ctx, repoID, pending); err != nil {
			return err
		}
		pending = pending[:0]
		return nil
	}
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
		if isSecretPath(rel) {
			d := Diagnostic{Code: "SKIPPED_SECRET", Path: rel}
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
			if prev, ok := existing[rel]; ok && prev.SizeBytes == info.Size() && prev.ModTimeNS == info.ModTime().UnixNano() {
				if prev.ParseStatus == "parse_failed" {
					d := Diagnostic{Code: "PARSE_FAILED", Path: rel, Message: "parse previously failed (unchanged file)"}
					res.Warnings = append(res.Warnings, d)
					_ = s.store.addDiagnostic(ctx, repoID, d)
				}
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
			if prev, ok := existing[rel]; ok && prev.Hash == hash {
				if prev.ParseStatus == "parse_failed" {
					d := Diagnostic{Code: "PARSE_FAILED", Path: rel, Message: "parse previously failed (unchanged file)"}
					res.Warnings = append(res.Warnings, d)
					_ = s.store.addDiagnostic(ctx, repoID, d)
				}
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
		parsed = finalizeSymbols(repoPath, rel, detectedLang, parsed)
		pending = append(pending, FileIndexUpdate{
			RelPath:     rel,
			Language:    detectedLang,
			Hash:        hash,
			SizeBytes:   info.Size(),
			ModTimeNS:   info.ModTime().UnixNano(),
			ParseStatus: parseStatus,
			Content:     string(content),
			Symbols:     parsed,
		})
		if len(pending) >= 128 {
			if err := flush(); err != nil {
				return err
			}
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
	if ferr := flush(); ferr != nil {
		return IndexResult{}, ferr
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
	repoPath, err := currentRepoRoot()
	if err != nil {
		return TextSearchResult{}, err
	}
	items, err := s.store.searchText(ctx, repoPath, opt)
	if err != nil {
		return TextSearchResult{}, err
	}
	return TextSearchResult{Results: items}, nil
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
	parsed = finalizeSymbols(repoPath, relPath, lang, parsed)
	flat := make([]OutlineSymbol, 0, len(parsed))
	for _, sym := range parsed {
		flat = append(flat, OutlineSymbol{ID: sym.ID, Name: sym.Name, Kind: sym.Kind, ParentID: sym.ParentID, StartLine: sym.StartLine, EndLine: sym.EndLine, Signature: sym.Signature, Language: lang})
	}
	result.Symbols = buildOutlineHierarchy(flat)
	_ = ctx
	return result, nil
}

func (s *Service) ShowSymbol(ctx context.Context, id string, contextLines int, verify bool) (ShowSymbolResult, error) {
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
	result := ShowSymbolResult{ID: rec.ID, Name: rec.Name, Kind: rec.Kind, FilePath: rec.FilePath, Language: rec.Language, QualifiedName: rec.QualifiedName, Signature: rec.Signature, StartLine: rec.StartLine, EndLine: rec.EndLine, Content: chunk}
	if verify {
		result.Verification = verifyStoredSymbol(rec, fullPath, content)
	}
	return result, nil
}

func (s *Service) ShowSymbols(ctx context.Context, ids []string) (ShowSymbolsResult, error) {
	result := ShowSymbolsResult{Symbols: make([]ShowSymbolResult, 0, len(ids))}
	for _, id := range ids {
		sym, err := s.ShowSymbol(ctx, id, 0, false)
		if err != nil {
			var coded *CodedError
			if errors.As(err, &coded) {
				result.Errors = append(result.Errors, BatchError{ID: id, Code: coded.Code, Message: coded.Message})
				continue
			}
			return ShowSymbolsResult{}, err
		}
		result.Symbols = append(result.Symbols, sym)
	}
	return result, nil
}

func verifyStoredSymbol(rec *symbolRecord, fullPath string, content []byte) *SymbolVerification {
	parsed, _, err := ParseSymbols(fullPath, content)
	if err != nil {
		return &SymbolVerification{Verified: false, Reason: "unable to parse file for verification"}
	}
	parsed = finalizeSymbols(rec.RepoPath, rec.FilePath, rec.Language, parsed)
	for i := range parsed {
		if parsed[i].ID == rec.ID {
			return &SymbolVerification{Verified: true}
		}
	}
	return &SymbolVerification{Verified: false, Reason: "symbol no longer matches indexed location"}
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

func (s *Service) RepoOutline(ctx context.Context) (RepoOutlineResult, error) {
	repoPath, err := currentRepoRoot()
	if err != nil {
		return RepoOutlineResult{}, err
	}
	return s.store.repoSummary(ctx, repoPath)
}

func currentRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return normalizeRepoPath(cwd)
}

func normalizeRepoPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return real, nil
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

func finalizeSymbols(repoPath, relPath, language string, symbols []Symbol) []Symbol {
	byQualified := make(map[string]string, len(symbols))
	byName := make(map[string]string, len(symbols))
	for i := range symbols {
		symbols[i].Language = language
		symbols[i].FilePath = relPath
		if symbols[i].QualifiedName == "" {
			symbols[i].QualifiedName = symbols[i].Name
		}
		symbols[i].ID = symbolID(repoPath, relPath, symbols[i])
		if symbols[i].QualifiedName != "" {
			if _, ok := byQualified[symbols[i].QualifiedName]; !ok {
				byQualified[symbols[i].QualifiedName] = symbols[i].ID
			}
		}
		if symbols[i].Name != "" {
			if _, ok := byName[symbols[i].Name]; !ok {
				byName[symbols[i].Name] = symbols[i].ID
			}
		}
	}
	for i := range symbols {
		if symbols[i].ParentID == "" {
			continue
		}
		if id, ok := byQualified[symbols[i].ParentID]; ok {
			symbols[i].ParentID = id
			continue
		}
		if id, ok := byName[symbols[i].ParentID]; ok {
			symbols[i].ParentID = id
			continue
		}
		symbols[i].ParentID = ""
	}
	return symbols
}

func buildOutlineHierarchy(flat []OutlineSymbol) []OutlineSymbol {
	if len(flat) == 0 {
		return nil
	}
	byID := make(map[string]OutlineSymbol, len(flat))
	children := make(map[string][]string, len(flat))
	roots := make([]string, 0, len(flat))
	for _, sym := range flat {
		sym.Children = nil
		byID[sym.ID] = sym
	}
	for _, sym := range flat {
		if sym.ParentID != "" {
			if _, ok := byID[sym.ParentID]; ok {
				children[sym.ParentID] = append(children[sym.ParentID], sym.ID)
				continue
			}
		}
		roots = append(roots, sym.ID)
	}
	var expand func(id string) OutlineSymbol
	expand = func(id string) OutlineSymbol {
		node := byID[id]
		for _, childID := range children[id] {
			node.Children = append(node.Children, expand(childID))
		}
		return node
	}
	out := make([]OutlineSymbol, 0, len(roots))
	for _, id := range roots {
		out = append(out, expand(id))
	}
	return out
}

func symbolID(repoPath, relPath string, sym Symbol) string {
	name := sym.QualifiedName
	if name == "" {
		name = sym.Name
	}
	repoKey := sha256hex([]byte(repoPath))[:12]
	return repoKey + ":" + filepath.ToSlash(relPath) + "::" + name + "#" + sym.Kind + ":" + strconv.Itoa(sym.StartByte) + "-" + strconv.Itoa(sym.EndByte)
}
