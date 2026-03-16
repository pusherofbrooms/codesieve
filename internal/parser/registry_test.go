package parser

import "testing"

func TestSupportedLanguagesIncludesBuiltins(t *testing.T) {
	names := SupportedLanguages()
	want := map[string]bool{
		"go":         false,
		"python":     false,
		"typescript": false,
		"javascript": false,
	}

	for _, name := range names {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("missing language %q in SupportedLanguages: %v", name, names)
		}
	}
}

func TestDetectLanguageByExtension(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "main.go", want: "go"},
		{path: "script.py", want: "python"},
		{path: "file.ts", want: "typescript"},
		{path: "file.tsx", want: "typescript"},
		{path: "file.js", want: "javascript"},
		{path: "file.jsx", want: "javascript"},
		{path: "README.md", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := DetectLanguage(tc.path); got != tc.want {
				t.Fatalf("DetectLanguage(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}
