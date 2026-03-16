package app

type CodedError struct {
	Code    string
	Message string
}

func (e *CodedError) Error() string { return e.Message }

func ErrInvalidArgs(message string) error {
	return &CodedError{Code: "INVALID_ARGS", Message: message}
}

func ErrNotFound(code, message string) error {
	return &CodedError{Code: code, Message: message}
}

type Diagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

type IndexOptions struct {
	Force       bool
	NoGitignore bool
	MaxFiles    int
	MaxSize     int64
}

type IndexResult struct {
	RepoPath         string       `json:"repo_path"`
	FilesIndexed     int          `json:"files_indexed"`
	FilesUpdated     int          `json:"files_updated"`
	FilesUnchanged   int          `json:"files_unchanged"`
	FilesDeleted     int          `json:"files_deleted"`
	SymbolsExtracted int          `json:"symbols_extracted"`
	FilesSkipped     []Diagnostic `json:"files_skipped"`
	Warnings         []Diagnostic `json:"warnings"`
	DurationMS       int64        `json:"duration_ms"`
}

type FileIndexUpdate struct {
	RelPath     string
	Language    string
	Hash        string
	SizeBytes   int64
	ModTimeNS   int64
	ParseStatus string
	Content     string
	Symbols     []Symbol
}

type SearchSymbolOptions struct {
	Query         string
	Limit         int
	Lang          string
	Kind          string
	PathSubstr    string
	CaseSensitive bool
}

type SymbolSearchItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	QualifiedName string  `json:"qualified_name,omitempty"`
	Kind          string  `json:"kind"`
	FilePath      string  `json:"file_path"`
	Line          int     `json:"line"`
	Signature     string  `json:"signature,omitempty"`
	Score         float64 `json:"score"`
}

type SymbolSearchResult struct {
	Results []SymbolSearchItem `json:"results"`
}

type SearchTextOptions struct {
	Query         string
	Limit         int
	Lang          string
	PathSubstr    string
	CaseSensitive bool
	Regex         bool
	ContextLines  int
}

type TextSearchItem struct {
	FilePath      string   `json:"file_path"`
	Line          int      `json:"line"`
	Snippet       string   `json:"snippet"`
	StartCol      int      `json:"start_col,omitempty"`
	EndCol        int      `json:"end_col,omitempty"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
}

type TextSearchResult struct {
	Results []TextSearchItem `json:"results"`
}

type OutlineSymbol struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	ParentID  string          `json:"parent_id,omitempty"`
	StartLine int             `json:"start_line"`
	EndLine   int             `json:"end_line"`
	Signature string          `json:"signature,omitempty"`
	Language  string          `json:"language,omitempty"`
	Children  []OutlineSymbol `json:"children,omitempty"`
}

type OutlineResult struct {
	FilePath string          `json:"file_path"`
	Language string          `json:"language"`
	Symbols  []OutlineSymbol `json:"symbols"`
}

type IndexRunSummary struct {
	StartedAt        string `json:"started_at"`
	FinishedAt       string `json:"finished_at"`
	DurationMS       int64  `json:"duration_ms"`
	Status           string `json:"status"`
	FilesIndexed     int    `json:"files_indexed"`
	FilesUpdated     int    `json:"files_updated"`
	FilesUnchanged   int    `json:"files_unchanged"`
	FilesDeleted     int    `json:"files_deleted"`
	FilesSkipped     int    `json:"files_skipped"`
	SymbolsExtracted int    `json:"symbols_extracted"`
	WarningsCount    int    `json:"warnings_count"`
	ErrorCode        string `json:"error_code,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
}

type RepoOutlineResult struct {
	RepoPath                string           `json:"repo_path"`
	TotalFiles              int              `json:"total_files"`
	TotalSymbols            int              `json:"total_symbols"`
	LanguageBreakdown       map[string]int   `json:"language_breakdown"`
	TopLevelDirectoryCounts map[string]int   `json:"top_level_directory_counts"`
	SymbolKindCounts        map[string]int   `json:"symbol_kind_counts"`
	IndexedAt               string           `json:"indexed_at"`
	IndexAgeSeconds         int64            `json:"index_age_seconds"`
	Stale                   bool             `json:"stale"`
	LatestIndexRun          *IndexRunSummary `json:"latest_index_run,omitempty"`
}

type SymbolVerification struct {
	Verified bool   `json:"verified"`
	Reason   string `json:"reason,omitempty"`
}

type ShowSymbolResult struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Kind          string              `json:"kind"`
	FilePath      string              `json:"file_path"`
	Language      string              `json:"language"`
	QualifiedName string              `json:"qualified_name,omitempty"`
	Signature     string              `json:"signature,omitempty"`
	StartLine     int                 `json:"start_line"`
	EndLine       int                 `json:"end_line"`
	Content       string              `json:"content"`
	Verification  *SymbolVerification `json:"verification,omitempty"`
}

type BatchError struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ShowSymbolsResult struct {
	Symbols []ShowSymbolResult `json:"symbols"`
	Errors  []BatchError       `json:"errors,omitempty"`
}

type ShowFileResult struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Content   string `json:"content"`
}

type Symbol struct {
	ID            string
	Name          string
	QualifiedName string
	Kind          string
	ParentID      string
	Signature     string
	Documentation string
	StartLine     int
	EndLine       int
	StartByte     int
	EndByte       int
	Language      string
	FilePath      string
}
