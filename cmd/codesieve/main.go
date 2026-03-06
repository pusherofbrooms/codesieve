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

	"github.com/jorgensen/codesieve/internal/app"
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
	case "version", "--version", "-v":
		fmt.Println(version)
		return 0
	case "index":
		return handleIndex(ctx, svc, args[1:], start)
	case "search":
		return handleSearch(ctx, svc, args[1:], start)
	case "outline":
		return handleOutline(ctx, svc, args[1:], start)
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
	path := args[0]
	opt := app.IndexOptions{MaxFiles: 10000, MaxSize: 1024 * 1024}
	jsonMode := false
	for _, arg := range args[1:] {
		switch {
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
	if len(args) < 2 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve search <symbol|text> <query>"))
	}
	jsonMode := false
	limit := 20
	lang := ""
	kind := ""
	query := args[1]
	for _, arg := range args[2:] {
		switch {
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
		default:
			return printError(start, jsonMode, app.ErrInvalidArgs("unknown flag: "+arg))
		}
	}

	switch args[0] {
	case "symbol":
		result, err := svc.SearchSymbols(ctx, app.SearchSymbolOptions{Query: query, Limit: limit, Lang: lang, Kind: kind})
		if err != nil {
			return printError(start, jsonMode, err)
		}
		return printSuccess(start, jsonMode, result)
	case "text":
		result, err := svc.SearchText(ctx, app.SearchTextOptions{Query: query, Limit: limit, Lang: lang})
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
	jsonMode := false
	for _, arg := range args[1:] {
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

func handleShow(ctx context.Context, svc *app.Service, args []string, start time.Time) int {
	if len(args) < 2 {
		return printError(start, false, app.ErrInvalidArgs("usage: codesieve show <symbol|file> <target>"))
	}
	jsonMode := false
	contentOnly := false
	contextLines := 0
	startLine := 0
	endLine := 0
	for _, arg := range args[2:] {
		switch {
		case arg == "--json":
			jsonMode = true
		case arg == "--content-only":
			contentOnly = true
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

	switch args[0] {
	case "symbol":
		result, err := svc.ShowSymbol(ctx, args[1], contextLines)
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
	case "file":
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
		return printError(start, jsonMode, app.ErrInvalidArgs("usage: codesieve show <symbol|file> <target>"))
	}
}

func printUsage() {
	fmt.Println("codesieve <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  index <path>")
	fmt.Println("  search symbol <query>")
	fmt.Println("  search text <query>")
	fmt.Println("  outline <file>")
	fmt.Println("  show symbol <id>")
	fmt.Println("  show file <path>")
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
		for _, s := range v.Symbols {
			fmt.Printf("- %s %s [%d-%d]\n", s.Kind, s.Name, s.StartLine, s.EndLine)
		}
	case app.ShowSymbolResult:
		fmt.Printf("%s %s %s:%d-%d\n\n%s", v.ID, v.Kind, v.FilePath, v.StartLine, v.EndLine, v.Content)
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
