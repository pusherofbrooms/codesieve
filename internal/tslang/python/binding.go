package python

// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../../third_party/tree-sitter-python/src/parser.c"
// #if __has_include("../../../third_party/tree-sitter-python/src/scanner.c")
// #include "../../../third_party/tree-sitter-python/src/scanner.c"
// #endif
import "C"

import "unsafe"

func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_python())
}
