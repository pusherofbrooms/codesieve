package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pusherofbrooms/codesieve/internal/app"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	ctx := context.Background()
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return 1
	}

	dbPath, err := app.DefaultDBPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	svc, err := app.NewService(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer svc.Close()

	start := time.Now()
	switch args[0] {
	case "help", "--help", "-h":
		printUsage()
		return 0
	case "version", "--version", "-v":
		fmt.Println(version)
		return 0
	case "index":
		return handleIndex(ctx, svc, args[1:], start)
	case "search":
		return handleSearch(ctx, svc, args[1:], start)
	case "outline":
		return handleOutline(ctx, svc, args[1:], start)
	case "repo":
		return handleRepo(ctx, svc, args[1:], start)
	case "show":
		return handleShow(ctx, svc, args[1:], start)
	default:
		printUsage()
		return 1
	}
}

func handleIndex(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) == 0 {
		return printError(start, false, app.ErrInvalidArgs("missing path"))
	}
	if isHelpArg(args[0]) {
		printIndexUsage()
		return 0
	}

	path := args[0]
	opt := app.IndexOptions{MaxFiles: 10000, MaxSize: 1024 * 1024}
	jsonMode := false
	for _, arg := range args[1:] {
		switch {
		case isHelpArg(arg):
			printIndexUsage()
			return 0
		case arg == "--json":
			jsonMode = true
		case arg == "--force":
			opt.Force = true
		case arg == "--no-gitignore":
			opt.NoGitignore = true
		case strings.HasPrefix(arg, "--max-files="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--max-files="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --max-files"))
			}
			opt.MaxFiles = v
		case strings.HasPrefix(arg, "--max-size="):
			v, err := strconv.ParseInt(strings.TrimPrefix(arg, "--max-size="), 10, 64)
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --max-size"))
			}
			opt.MaxSize = v
		default:
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}

	result, err := svc.Index(ctx, path, opt)
	if err != nil {
		return printError(start, jsonMode, err)
	}
	return printSuccess(start, jsonMode, result)
}

func handleSearch(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) == 0 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve search <symbol|text> <query>"))
	}
	if isHelpArg(args[0]) {
		printSearchUsage()
		return 0
	}

	subcommand := args[0]
	if subcommand != "symbol" && subcommand != "text" {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve search <symbol|text> <query>"))
	}

	if len(args) < 2 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve search <symbol|text> <query>"))
	}
	if isHelpArg(args[1]) {
		if subcommand == "symbol" {
			printSearchSymbolUsage()
		} else {
			printSearchTextUsage()
		}
		return 0
	}

	jsonMode := false
	limit := 20
	lang := ""
	kind := ""
	pathSubstr := ""
	caseSensitive := false
	regexMode := false
	contextLines := 0
	query := args[1]
	for _, arg := range args[2:] {
		switch {
		case isHelpArg(arg):
			if subcommand == "symbol" {
				printSearchSymbolUsage()
			} else {
				printSearchTextUsage()
			}
			return 0
		case arg == "--json":
			jsonMode = true
		case strings.HasPrefix(arg, "--limit="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --limit"))
			}
			limit = v
		case strings.HasPrefix(arg, "--lang="):
			lang = strings.TrimPrefix(arg, "--lang=")
		case strings.HasPrefix(arg, "--kind="):
			kind = strings.TrimPrefix(arg, "--kind=")
		case strings.HasPrefix(arg, "--path-substr="):
			pathSubstr = strings.TrimPrefix(arg, "--path-substr=")
		case arg == "--case-sensitive":
			caseSensitive = true
		case arg == "--regex":
			regexMode = true
		case strings.HasPrefix(arg, "--context-lines="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--context-lines="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --context-lines"))
			}
			contextLines = v
		default:
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}

	switch subcommand {
	case "symbol":
		result, err := svc.SearchSymbols(ctx, app.SearchSymbolOptions{Query: query, Limit: limit, Lang: lang, Kind: kind, PathSubstr: pathSubstr, CaseSensitive: caseSensitive})
		if err != nil {
			return printError(start, jsonMode, err)
		}
		return printSuccess(start, jsonMode, result)
	case "text":
		result, err := svc.SearchText(ctx, app.SearchTextOptions{Query: query, Limit: limit, Lang: lang, PathSubstr: pathSubstr, CaseSensitive: caseSensitive, Regex: regexMode, ContextLines: contextLines})
		if err != nil {
			return printError(start, jsonMode, err)
		}
		return printSuccess(start, jsonMode, result)
	default:
		return printError(start, jsonMode, app.ErrInvalidArgs("usage: codesieve search <symbol|text> <query>"))
	}
}

func handleOutline(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) == 0 {
		return printError(start, false, app.ErrInvalidArgs("missing file path"))
	}
	if isHelpArg(args[0]) {
		printOutlineUsage()
		return 0
	}

	jsonMode := false
	for _, arg := range args[1:] {
		if isHelpArg(arg) {
			printOutlineUsage()
			return 0
		}
		if arg == "--json" {
			jsonMode = true
		} else {
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}
	result, err := svc.Outline(ctx, args[0])
	if err != nil {
		return printError(start, jsonMode, err)
	}
	return printSuccess(start, jsonMode, result)
}

func handleRepo(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) == 0 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve repo outline"))
	}
	if isHelpArg(args[0]) {
		printRepoUsage()
		return 0
	}
	if args[0] != "outline" {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve repo outline"))
	}

	jsonMode := false
	for _, arg := range args[1:] {
		switch {
		case isHelpArg(arg):
			printRepoOutlineUsage()
			return 0
		case arg == "--json":
			jsonMode = true
		default:
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}

	result, err := svc.RepoOutline(ctx)
	if err != nil {
		return printError(start, jsonMode, err)
	}
	return printSuccess(start, jsonMode, result)
}

func handleShow(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) == 0 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve show <symbol|symbols|file> <target>"))
	}
	if isHelpArg(args[0]) {
		printShowUsage()
		return 0
	}

	subcommand := args[0]
	if subcommand != "symbol" && subcommand != "symbols" && subcommand != "file" {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve show <symbol|symbols|file> <target>"))
	}

	if len(args) < 2 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve show <symbol|symbols|file> <target>"))
	}
	if isHelpArg(args[1]) {
		switch subcommand {
		case "symbol":
			printShowSymbolUsage()
		case "symbols":
			printShowSymbolsUsage()
		default:
			printShowFileUsage()
		}
		return 0
	}
	if subcommand == "symbols" && strings.HasPrefix(args[1], "-") {
		return printError(start, false, app.ErrInvalidArgs("missing symbol ids"))
	}

	jsonMode := false
	contentOnly := false
	verify := false
	contextLines := 0
	startLine := 0
	endLine := 0
	symbolIDs := []string{}
	if subcommand == "symbols" {
		symbolIDs = append(symbolIDs, args[1])
	}
	for _, arg := range args[2:] {
		if subcommand == "symbols" && !strings.HasPrefix(arg, "-") {
			symbolIDs = append(symbolIDs, arg)
			continue
		}
		switch {
		case isHelpArg(arg):
			switch subcommand {
			case "symbol":
				printShowSymbolUsage()
			case "symbols":
				printShowSymbolsUsage()
			default:
				printShowFileUsage()
			}
			return 0
		case arg == "--json":
			jsonMode = true
		case arg == "--content-only":
			contentOnly = true
		case arg == "--verify":
			verify = true
		case strings.HasPrefix(arg, "--context="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--context="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --context"))
			}
			contextLines = v
		case strings.HasPrefix(arg, "--start-line="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--start-line="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --start-line"))
			}
			startLine = v
		case strings.HasPrefix(arg, "--end-line="):
			v, err := strconv.Atoi(strings.TrimPrefix(arg, "--end-line="))
			if err != nil {
				return printError(start, jsonMode, app.ErrInvalidArgs("invalid --end-line"))
			}
			endLine = v
		default:
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}

	switch subcommand {
	case "symbol":
		result, err := svc.ShowSymbol(ctx, args[1], contextLines, verify)
		if err != nil {
			return printError(start, jsonMode, err)
		}
		if contentOnly {
			fmt.Print(result.Content)
			if !strings.HasSuffix(result.Content, "\n") {
				fmt.Println()
			}
			return 0
		}
		return printSuccess(start, jsonMode, result)
	case "symbols":
		if contextLines != 0 || startLine != 0 || endLine != 0 || verify {
			return printError(start, jsonMode, app.ErrInvalidArgs("--context, --start-line, --end-line, and --verify are only valid for specific show subcommands"))
		}
		result, err := svc.ShowSymbols(ctx, symbolIDs)
		if err != nil {
			return printError(start, jsonMode, err)
		}
		if contentOnly {
			for _, sym := range result.Symbols {
				fmt.Print(sym.Content)
				if !strings.HasSuffix(sym.Content, "\n") {
					fmt.Println()
				}
			}
			return 0
		}
		return printSuccess(start, jsonMode, result)
	case "file":
		if contextLines != 0 || verify {
			return printError(start, jsonMode, app.ErrInvalidArgs("--context and --verify are only valid for 'show symbol'"))
		}
		result, err := svc.ShowFile(ctx, args[1], startLine, endLine)
		if err != nil {
			return printError(start, jsonMode, err)
		}
		if contentOnly {
			fmt.Print(result.Content)
			if !strings.HasSuffix(result.Content, "\n") {
				fmt.Println()
			}
			return 0
		}
		return printSuccess(start, jsonMode, result)
	default:
		return printError(start, jsonMode, app.ErrInvalidArgs("usage: codesieve show <symbol|symbols|file> <target>"))
	}
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func printUsage() {
	fmt.Println("codesieve <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  help")
	fmt.Println("  version")
	fmt.Println("  index <path>")
	fmt.Println("  search symbol <query>")
	fmt.Println("  search text <query>")
	fmt.Println("  outline <file>")
	fmt.Println("  repo outline")
	fmt.Println("  show symbol <id>")
	fmt.Println("  show symbols <id...>")
	fmt.Println("  show file <path>")
	fmt.Println("")
	fmt.Println("Run 'codesieve <command> --help' for command-specific help.")
}

func printIndexUsage() {
	fmt.Println("Usage: codesieve index <path> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --force")
	fmt.Println("  --no-gitignore")
	fmt.Println("  --max-files=<n>")
	fmt.Println("  --max-size=<bytes>")
}

func printSearchUsage() {
	fmt.Println("Usage: codesieve search <symbol|text> <query> [flags]")
	fmt.Println("")
	fmt.Println("Run 'codesieve search symbol --help' or 'codesieve search text --help' for details.")
}

func printSearchSymbolUsage() {
	fmt.Println("Usage: codesieve search symbol <query> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --limit=<n>")
	fmt.Println("  --lang=<language>")
	fmt.Println("  --kind=<kind>")
	fmt.Println("  --path-substr=<substring>")
	fmt.Println("  --case-sensitive")
}

func printSearchTextUsage() {
	fmt.Println("Usage: codesieve search text <query> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --limit=<n>")
	fmt.Println("  --lang=<language>")
	fmt.Println("  --path-substr=<substring>")
	fmt.Println("  --case-sensitive")
	fmt.Println("  --regex")
	fmt.Println("  --context-lines=<n>")
}

func printOutlineUsage() {
	fmt.Println("Usage: codesieve outline <file> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
}

func printRepoUsage() {
	fmt.Println("Usage: codesieve repo outline [flags]")
	fmt.Println("")
	fmt.Println("Run 'codesieve repo outline --help' for details.")
}

func printRepoOutlineUsage() {
	fmt.Println("Usage: codesieve repo outline [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
}

func printShowUsage() {
	fmt.Println("Usage: codesieve show <symbol|symbols|file> <target> [flags]")
	fmt.Println("")
	fmt.Println("Run 'codesieve show symbol --help', 'codesieve show symbols --help', or 'codesieve show file --help' for details.")
}

func printShowSymbolUsage() {
	fmt.Println("Usage: codesieve show symbol <id> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --content-only")
	fmt.Println("  --context=<n>")
	fmt.Println("  --verify")
}

func printShowSymbolsUsage() {
	fmt.Println("Usage: codesieve show symbols <id...> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --content-only")
}

func printShowFileUsage() {
	fmt.Println("Usage: codesieve show file <path> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json")
	fmt.Println("  --content-only")
	fmt.Println("  --start-line=<n>")
	fmt.Println("  --end-line=<n>")
}

func printOutlineSymbols(symbols []app.OutlineSymbol, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, s := range symbols {
		fmt.Printf("%s- %s %s [%d-%d]\n", indent, s.Kind, s.Name, s.StartLine, s.EndLine)
		if len(s.Children) > 0 {
			printOutlineSymbols(s.Children, depth+1)
		}
	}
}

func printSuccess(start time.Time, jsonMode bool, data any) int {
	if jsonMode {
		payload := map[string]any{
			"ok":   true,
			"data": data,
			"meta": map[string]any{
				"timing_ms": time.Since(start).Milliseconds(),
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
		return 0
	}

	switch v := data.(type) {
	case app.IndexResult:
		fmt.Printf("Indexed %d files (%d updated, %d unchanged, %d deleted), %d symbols in %dms\n", v.FilesIndexed, v.FilesUpdated, v.FilesUnchanged, v.FilesDeleted, v.SymbolsExtracted, time.Since(start).Milliseconds())
		for _, d := range v.Warnings {
			fmt.Printf("- %s: %s\n", d.Code, d.Path)
		}
	case app.SymbolSearchResult:
		for _, item := range v.Results {
			fmt.Printf("%s\t%s\t%s:%d\n", item.ID, item.Kind, item.FilePath, item.Line)
		}
	case app.TextSearchResult:
		for _, item := range v.Results {
			fmt.Printf("%s:%d\t%s\n", item.FilePath, item.Line, item.Snippet)
		}
	case app.OutlineResult:
		fmt.Printf("%s (%s)\n", v.FilePath, v.Language)
		printOutlineSymbols(v.Symbols, 0)
	case app.RepoOutlineResult:
		fmt.Printf("%s\n", v.RepoPath)
		fmt.Printf("files: %d  symbols: %d  indexed_at: %s  stale: %t\n", v.TotalFiles, v.TotalSymbols, v.IndexedAt, v.Stale)
		fmt.Println("languages:")
		for k, c := range v.LanguageBreakdown {
			fmt.Printf("- %s: %d\n", k, c)
		}
		fmt.Println("top-level directories:")
		for k, c := range v.TopLevelDirectoryCounts {
			fmt.Printf("- %s: %d\n", k, c)
		}
		fmt.Println("symbol kinds:")
		for k, c := range v.SymbolKindCounts {
			fmt.Printf("- %s: %d\n", k, c)
		}
	case app.ShowSymbolResult:
		fmt.Printf("%s %s %s:%d-%d\n", v.ID, v.Kind, v.FilePath, v.StartLine, v.EndLine)
		if v.Verification != nil {
			status := "failed"
			if v.Verification.Verified {
				status = "ok"
			}
			fmt.Printf("verification: %s", status)
			if v.Verification.Reason != "" {
				fmt.Printf(" (%s)", v.Verification.Reason)
			}
			fmt.Println()
		}
		fmt.Printf("\n%s", v.Content)
	case app.ShowSymbolsResult:
		for _, sym := range v.Symbols {
			fmt.Printf("%s %s %s:%d-%d\n\n%s", sym.ID, sym.Kind, sym.FilePath, sym.StartLine, sym.EndLine, sym.Content)
			if !strings.HasSuffix(sym.Content, "\n") {
				fmt.Println()
			}
		}
		for _, e := range v.Errors {
			fmt.Printf("%s: %s (%s)\n", e.ID, e.Message, e.Code)
		}
	case app.ShowFileResult:
		fmt.Printf("%s:%d-%d\n\n%s", v.FilePath, v.StartLine, v.EndLine, v.Content)
	default:
		fmt.Printf("%v\n", data)
	}
	return 0
}

func printError(start time.Time, jsonMode bool, err error) int {
	code := "INTERNAL"
	message := err.Error()
	var cerr *app.CodedError
	if errors.As(err, &cerr) {
		code = cerr.Code
		message = cerr.Message
	}
	if jsonMode {
		payload := map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":    code,
				"message": message,
			},
			"meta": map[string]any{
				"timing_ms": time.Since(start).Milliseconds(),
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
		return 1
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", code, message)
	return 1
}
