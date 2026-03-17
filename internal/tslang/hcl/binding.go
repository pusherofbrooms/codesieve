package hcl

// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../../third_party/tree-sitter-hcl/src/parser.c"
// #if __has_include("../../../third_party/tree-sitter-hcl/src/scanner.c")
// #include "../../../third_party/tree-sitter-hcl/src/scanner.c"
// #endif
import "C"

import "unsafe"

func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_hcl())
}
