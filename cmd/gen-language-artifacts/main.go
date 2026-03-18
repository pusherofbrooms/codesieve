package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pusherofbrooms/codesieve/internal/languages"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	docPath := filepath.Join(root, "docs", "supported_languages.md")
	if err := os.WriteFile(docPath, []byte(languages.RenderSupportedLanguagesMarkdown()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", docPath, err)
	}

	shellPath := filepath.Join(root, "scripts", "language-map.sh")
	if err := os.WriteFile(shellPath, []byte(languages.RenderLanguageMapShell()), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", shellPath, err)
	}

	fmt.Printf("updated %s\n", docPath)
	fmt.Printf("updated %s\n", shellPath)
	return nil
}
