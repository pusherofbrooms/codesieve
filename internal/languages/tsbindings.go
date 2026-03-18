package languages

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type TreeSitterBinding struct {
	Package    string
	GrammarDir string
	Variants   []TreeSitterBindingVariant
}

type TreeSitterBindingVariant struct {
	FileName        string
	SourceSubdir    string
	FunctionName    string
	CSymbol         string
	CPPIncludeDir   string
	OptionalScanner bool
}

func TreeSitterBindings() []TreeSitterBinding {
	out := make([]TreeSitterBinding, 0, len(TreeSitter()))
	for _, item := range TreeSitter() {
		if item.Name == "typescript" {
			out = append(out, TreeSitterBinding{
				Package:    item.Name,
				GrammarDir: item.GrammarDir,
				Variants: []TreeSitterBindingVariant{
					{
						FileName:        "binding.go",
						SourceSubdir:    "typescript/src",
						FunctionName:    "LanguageTypescript",
						CSymbol:         "tree_sitter_typescript",
						CPPIncludeDir:   "typescript/src",
						OptionalScanner: false,
					},
					{
						FileName:        "tsx.go",
						SourceSubdir:    "tsx/src",
						FunctionName:    "LanguageTSX",
						CSymbol:         "tree_sitter_tsx",
						CPPIncludeDir:   "tsx/src",
						OptionalScanner: false,
					},
				},
			})
			continue
		}
		if item.Name == "php" {
			out = append(out, TreeSitterBinding{
				Package:    item.Name,
				GrammarDir: item.GrammarDir,
				Variants: []TreeSitterBindingVariant{
					{
						FileName:        "binding.go",
						SourceSubdir:    "php/src",
						FunctionName:    "Language",
						CSymbol:         "tree_sitter_php",
						CPPIncludeDir:   "php/src",
						OptionalScanner: false,
					},
				},
			})
			continue
		}

		out = append(out, TreeSitterBinding{
			Package:    item.Name,
			GrammarDir: item.GrammarDir,
			Variants: []TreeSitterBindingVariant{
				{
					FileName:        "binding.go",
					SourceSubdir:    "src",
					FunctionName:    "Language",
					CSymbol:         defaultCSymbol(item.Name),
					OptionalScanner: true,
				},
			},
		})
	}
	return out
}

func RenderTSLangBindings() map[string]string {
	out := map[string]string{}
	for _, b := range TreeSitterBindings() {
		for _, v := range b.Variants {
			rel := filepath.ToSlash(filepath.Join("internal", "tslang", b.Package, v.FileName))
			out[rel] = RenderTSLangBinding(b.Package, b.GrammarDir, v)
		}
	}
	return out
}

func RenderTSLangBinding(pkg, grammarDir string, v TreeSitterBindingVariant) string {
	prefix := "../../../" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(grammarDir, v.SourceSubdir)), "./")

	var b strings.Builder
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\n")

	if v.CPPIncludeDir != "" {
		b.WriteString("// #cgo CPPFLAGS: -I")
		b.WriteString("../../../")
		b.WriteString(strings.TrimPrefix(filepath.ToSlash(filepath.Join(grammarDir, v.CPPIncludeDir)), "./"))
		b.WriteString("\n")
	}
	b.WriteString("// #cgo CFLAGS: -std=c11 -fPIC\n")
	b.WriteString("// #include \"")
	b.WriteString(prefix)
	b.WriteString("/parser.c\"\n")
	if v.OptionalScanner {
		b.WriteString("// #if __has_include(\"")
		b.WriteString(prefix)
		b.WriteString("/scanner.c\")\n")
		b.WriteString("// #include \"")
		b.WriteString(prefix)
		b.WriteString("/scanner.c\"\n")
		b.WriteString("// #endif\n")
	} else {
		b.WriteString("// #include \"")
		b.WriteString(prefix)
		b.WriteString("/scanner.c\"\n")
	}
	b.WriteString("import \"C\"\n\n")
	b.WriteString("import \"unsafe\"\n\n")
	b.WriteString("func ")
	b.WriteString(v.FunctionName)
	b.WriteString("() unsafe.Pointer {\n")
	b.WriteString("\treturn unsafe.Pointer(C.")
	b.WriteString(v.CSymbol)
	b.WriteString("())\n")
	b.WriteString("}\n")
	return b.String()
}

func SortedTSBindingPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func defaultCSymbol(name string) string {
	switch name {
	case "csharp":
		return "tree_sitter_c_sharp"
	default:
		return fmt.Sprintf("tree_sitter_%s", name)
	}
}
