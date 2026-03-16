package app

import "github.com/pusherofbrooms/codesieve/internal/parser"

func DetectLanguage(path string) string {
	return parser.DetectLanguage(path)
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	return parser.ParseSymbols(path, content)
}
