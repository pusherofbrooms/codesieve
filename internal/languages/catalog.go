package languages

import "slices"

type Metadata struct {
	Name          string
	DisplayName   string
	Parser        string
	Version       string
	Extensions    []string
	GrammarRepo   string
	GrammarDir    string
	Notes         string
	SupportsGlobs []string
}

var catalog = []Metadata{
	{
		Name:        "go",
		DisplayName: "Go",
		Parser:      "go/parser",
		Version:     "1",
		Extensions:  []string{".go"},
	},
	{
		Name:        "python",
		DisplayName: "Python",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".py"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-python.git",
		GrammarDir:  "third_party/tree-sitter-python",
	},
	{
		Name:        "rust",
		DisplayName: "Rust",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".rs"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-rust.git",
		GrammarDir:  "third_party/tree-sitter-rust",
	},
	{
		Name:        "zig",
		DisplayName: "Zig",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".zig"},
		GrammarRepo: "https://github.com/tree-sitter-grammars/tree-sitter-zig.git",
		GrammarDir:  "third_party/tree-sitter-zig",
	},
	{
		Name:        "php",
		DisplayName: "PHP",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".php"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-php.git",
		GrammarDir:  "third_party/tree-sitter-php",
	},
	{
		Name:        "typescript",
		DisplayName: "TypeScript / TSX",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".ts", ".tsx"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-typescript.git",
		GrammarDir:  "third_party/tree-sitter-typescript",
	},
	{
		Name:        "javascript",
		DisplayName: "JavaScript",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".js", ".jsx", ".mjs", ".cjs"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-javascript.git",
		GrammarDir:  "third_party/tree-sitter-javascript",
	},
	{
		Name:        "java",
		DisplayName: "Java",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".java"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-java.git",
		GrammarDir:  "third_party/tree-sitter-java",
	},
	{
		Name:        "csharp",
		DisplayName: "C#",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".cs", ".csx"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-c-sharp.git",
		GrammarDir:  "third_party/tree-sitter-csharp",
	},
	{
		Name:          "hcl",
		DisplayName:   "Terraform/OpenTofu (HCL)",
		Parser:        "tree-sitter",
		Version:       "1",
		Extensions:    []string{".tf", ".tfvars", ".hcl"},
		SupportsGlobs: []string{"*.tf.json", "*.tfvars.json"},
		GrammarRepo:   "https://github.com/tree-sitter-grammars/tree-sitter-hcl.git",
		GrammarDir:    "third_party/tree-sitter-hcl",
	},
	{
		Name:        "json",
		DisplayName: "JSON",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".json"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-json.git",
		GrammarDir:  "third_party/tree-sitter-json",
	},
	{
		Name:        "bash",
		DisplayName: "Bash",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".sh", ".bash"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-bash.git",
		GrammarDir:  "third_party/tree-sitter-bash",
		Notes:       "Also detected via bash shebang for extensionless scripts",
	},
	{
		Name:        "yaml",
		DisplayName: "YAML",
		Parser:      "tree-sitter",
		Version:     "1",
		Extensions:  []string{".yaml", ".yml"},
		GrammarRepo: "https://github.com/tree-sitter/tree-sitter-yaml.git",
		GrammarDir:  "third_party/tree-sitter-yaml",
	},
}

func All() []Metadata {
	out := make([]Metadata, len(catalog))
	for i := range catalog {
		out[i] = catalog[i]
		out[i].Extensions = slices.Clone(catalog[i].Extensions)
		out[i].SupportsGlobs = slices.Clone(catalog[i].SupportsGlobs)
	}
	return out
}

func ByName(name string) (Metadata, bool) {
	for _, item := range catalog {
		if item.Name == name {
			copy := item
			copy.Extensions = slices.Clone(item.Extensions)
			copy.SupportsGlobs = slices.Clone(item.SupportsGlobs)
			return copy, true
		}
	}
	return Metadata{}, false
}

func TreeSitter() []Metadata {
	items := make([]Metadata, 0, len(catalog))
	for _, item := range catalog {
		if item.Parser != "tree-sitter" {
			continue
		}
		copy := item
		copy.Extensions = slices.Clone(item.Extensions)
		copy.SupportsGlobs = slices.Clone(item.SupportsGlobs)
		items = append(items, copy)
	}
	return items
}
