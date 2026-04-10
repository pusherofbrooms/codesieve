package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type languageFixtureCase struct {
	name             string
	fixtureDir       string
	filePath         string
	expectedLanguage string
	topName          string
	topKind          string
	childName        string
	childKind        string
	query            string
	queryKind        string
	queryQualified   string
	showContains     string
}

func TestLanguageFixturesFollowStandardContract(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	cases := []languageFixtureCase{
		{
			name:             "go",
			fixtureDir:       "tests/testdata/languages/go",
			filePath:         "basic.go",
			expectedLanguage: "go",
			topName:          "AuthService",
			topKind:          "struct",
			childName:        "Logout",
			childKind:        "method",
			query:            "Login",
			queryKind:        "function",
			queryQualified:   "Login",
			showContains:     "func Login(user string) error",
		},
		{
			name:             "python",
			fixtureDir:       "tests/testdata/languages/python",
			filePath:         "basic.py",
			expectedLanguage: "python",
			topName:          "Auth",
			topKind:          "class",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "Auth.login",
			showContains:     "def login(self, user):",
		},
		{
			name:             "ruby",
			fixtureDir:       "tests/testdata/languages/ruby",
			filePath:         "basic.rb",
			expectedLanguage: "ruby",
			topName:          "Example",
			topKind:          "module",
			childName:        "Client",
			childKind:        "class",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "Example.Client.login",
			showContains:     "def login(user)",
		},
		{
			name:             "typescript",
			fixtureDir:       "tests/testdata/languages/typescript",
			filePath:         "basic.ts",
			expectedLanguage: "typescript",
			topName:          "Client",
			topKind:          "class",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "Client.login",
			showContains:     "login(token: string)",
		},
		{
			name:             "javascript",
			fixtureDir:       "tests/testdata/languages/javascript",
			filePath:         "basic.js",
			expectedLanguage: "javascript",
			topName:          "Client",
			topKind:          "class",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "Client.login",
			showContains:     "login(token)",
		},
		{
			name:             "rust",
			fixtureDir:       "tests/testdata/languages/rust",
			filePath:         "basic.rs",
			expectedLanguage: "rust",
			topName:          "AuthService",
			topKind:          "struct",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "AuthService.login",
			showContains:     "fn login(&self, user: &str) -> bool",
		},
		{
			name:             "java",
			fixtureDir:       "tests/testdata/languages/java",
			filePath:         "basic.java",
			expectedLanguage: "java",
			topName:          "AuthService",
			topKind:          "class",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "AuthService.login(String)",
			showContains:     "public boolean login(String user)",
		},
		{
			name:             "zig",
			fixtureDir:       "tests/testdata/languages/zig",
			filePath:         "basic.zig",
			expectedLanguage: "zig",
			topName:          "Client",
			topKind:          "struct",
			childName:        "login",
			childKind:        "method",
			query:            "login",
			queryKind:        "method",
			queryQualified:   "Client.login",
			showContains:     "pub fn login(self: *Client, user: []const u8) bool",
		},
		{
			name:             "csharp",
			fixtureDir:       "tests/testdata/languages/csharp",
			filePath:         "basic.cs",
			expectedLanguage: "csharp",
			topName:          "AuthService",
			topKind:          "class",
			childName:        "Login",
			childKind:        "method",
			query:            "Login",
			queryKind:        "method",
			queryQualified:   "AuthService.Login(string)",
			showContains:     "public bool Login(string user)",
		},
		{
			name:             "php",
			fixtureDir:       "tests/testdata/languages/php",
			filePath:         "basic.php",
			expectedLanguage: "php",
			topName:          "Example.App",
			topKind:          "namespace",
			childName:        "Service",
			childKind:        "class",
			query:            "run",
			queryKind:        "method",
			queryQualified:   "Example.App.Service.run",
			showContains:     "public function run(): void",
		},
		{
			name:             "bash",
			fixtureDir:       "tests/testdata/languages/bash",
			filePath:         "basic.sh",
			expectedLanguage: "bash",
			topName:          "script:basic.sh",
			topKind:          "script",
			childName:        "login",
			childKind:        "function",
			query:            "AUTH_HEADER",
			queryKind:        "variable",
			queryQualified:   "AUTH_HEADER",
			showContains:     "export AUTH_HEADER",
		},
		{
			name:             "hcl",
			fixtureDir:       "tests/testdata/languages/hcl",
			filePath:         "basic.tf",
			expectedLanguage: "hcl",
			topName:          "file:basic.tf",
			topKind:          "file",
			childName:        "app",
			childKind:        "resource",
			query:            "app",
			queryKind:        "resource",
			queryQualified:   "resource.aws_s3_bucket.app",
			showContains:     "resource \"aws_s3_bucket\" \"app\"",
		},
		{
			name:             "yaml",
			fixtureDir:       "tests/testdata/languages/yaml",
			filePath:         "basic.yaml",
			expectedLanguage: "yaml",
			topName:          "template:basic.yaml",
			topKind:          "template",
			childName:        "Resources",
			childKind:        "section",
			query:            "AppBucket",
			queryKind:        "resource",
			queryQualified:   "Resources.AppBucket",
			showContains:     "AppBucket:",
		},
		{
			name:             "json",
			fixtureDir:       "tests/testdata/languages/json",
			filePath:         "basic.json",
			expectedLanguage: "json",
			topName:          "template:basic.json",
			topKind:          "template",
			childName:        "Resources",
			childKind:        "section",
			query:            "AppBucket",
			queryKind:        "resource",
			queryQualified:   "Resources.AppBucket",
			showContains:     "\"AppBucket\"",
		},
		{
			name:             "nix",
			fixtureDir:       "tests/testdata/languages/nix",
			filePath:         "basic.nix",
			expectedLanguage: "nix",
			topName:          "file:basic.nix",
			topKind:          "file",
			childName:        "packages",
			childKind:        "attrset",
			query:            "default",
			queryKind:        "binding",
			queryQualified:   "packages.${system}.default",
			showContains:     "packages.${system}.default = pkgs.hello;",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixtureSrc := fixturePath(t, tc.fixtureDir)
			workdir := filepath.Join(t.TempDir(), "work")
			copyDir(t, fixtureSrc, workdir)

			res, err := svc.Index(ctx, workdir, IndexOptions{})
			if err != nil {
				t.Fatalf("Index error: %v", err)
			}
			if res.FilesIndexed != 1 || res.FilesUpdated != 1 || res.FilesUnchanged != 0 {
				t.Fatalf("unexpected index result: %+v", res)
			}

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Getwd: %v", err)
			}
			if err := os.Chdir(workdir); err != nil {
				t.Fatalf("Chdir(%s): %v", workdir, err)
			}
			t.Cleanup(func() { _ = os.Chdir(cwd) })

			outline, err := svc.Outline(ctx, tc.filePath)
			if err != nil {
				t.Fatalf("Outline error: %v", err)
			}
			if outline.Language != tc.expectedLanguage {
				t.Fatalf("outline language = %q, want %q", outline.Language, tc.expectedLanguage)
			}

			top := findOutlineSymbol(outline.Symbols, tc.topName, tc.topKind)
			if top == nil {
				t.Fatalf("missing top-level symbol %s/%s in outline: %+v", tc.topName, tc.topKind, outline.Symbols)
			}
			child := findOutlineSymbol(top.Children, tc.childName, tc.childKind)
			if child == nil {
				t.Fatalf("missing nested child symbol %s/%s under %s: %+v", tc.childName, tc.childKind, tc.topName, top.Children)
			}

			search, err := svc.SearchSymbols(ctx, SearchSymbolOptions{Query: tc.query, Kind: tc.queryKind, Limit: 10})
			if err != nil {
				t.Fatalf("SearchSymbols error: %v", err)
			}
			id := ""
			for _, item := range search.Results {
				if item.QualifiedName == tc.queryQualified && item.Kind == tc.queryKind {
					id = item.ID
					break
				}
			}
			if id == "" {
				t.Fatalf("missing expected search symbol %s/%s in %+v", tc.queryQualified, tc.queryKind, search.Results)
			}

			shown, err := svc.ShowSymbol(ctx, id, 0, false)
			if err != nil {
				t.Fatalf("ShowSymbol error: %v", err)
			}
			if !strings.Contains(shown.Content, tc.showContains) {
				t.Fatalf("show symbol content missing %q:\n%s", tc.showContains, shown.Content)
			}
		})
	}
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	path := filepath.Join(cwd, "..", "..", rel)
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs(%s): %v", path, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture path %s: %v", abs, err)
	}
	return abs
}

func findOutlineSymbol(symbols []OutlineSymbol, name, kind string) *OutlineSymbol {
	for i := range symbols {
		s := &symbols[i]
		if s.Name == name && s.Kind == kind {
			return s
		}
	}
	return nil
}
