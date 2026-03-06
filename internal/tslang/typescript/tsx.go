package typescript

// #cgo CPPFLAGS: -I../../../third_party/tree-sitter-typescript/tsx/src
// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../../third_party/tree-sitter-typescript/tsx/src/parser.c"
// #include "../../../third_party/tree-sitter-typescript/tsx/src/scanner.c"
import "C"

import "unsafe"

func LanguageTSX() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_tsx())
}
