package app

import "github.com/pusherofbrooms/codesieve/internal/parser"

func DetectLanguage(path string) string {
	return parser.DetectLanguage(path)
}

func DetectLanguageWithContent(path string, content []byte) string {
	return parser.DetectLanguageWithContent(path, content)
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	return parser.ParseSymbols(path, content)
}

func LanguageVersion(language string) string {
	return parser.LanguageVersion(language)
}
