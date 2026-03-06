package typescript

// #cgo CPPFLAGS: -I../../../third_party/tree-sitter-typescript/typescript/src
// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../../third_party/tree-sitter-typescript/typescript/src/parser.c"
// #include "../../../third_party/tree-sitter-typescript/typescript/src/scanner.c"
import "C"

import "unsafe"

func LanguageTypescript() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_typescript())
}
