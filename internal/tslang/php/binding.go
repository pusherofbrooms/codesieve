package php

// #cgo CPPFLAGS: -I../../../third_party/tree-sitter-php/php/src
// #cgo CFLAGS: -std=c11 -fPIC
// #include "../../../third_party/tree-sitter-php/php/src/parser.c"
// #include "../../../third_party/tree-sitter-php/php/src/scanner.c"
import "C"

import "unsafe"

func Language() unsafe.Pointer {
	return unsafe.Pointer(C.tree_sitter_php())
}
