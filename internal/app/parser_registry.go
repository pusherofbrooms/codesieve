package app

import "github.com/pusherofbrooms/codesieve/internal/parser"

func DetectLanguage(path string) string {
	return parser.DetectLanguage(path)
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	syms, lang, err := parser.ParseSymbols(path, content)
	if err != nil || len(syms) == 0 {
		return nil, lang, err
	}
	out := make([]Symbol, 0, len(syms))
	for _, sym := range syms {
		out = append(out, Symbol{
			Name:          sym.Name,
			QualifiedName: sym.QualifiedName,
			Kind:          sym.Kind,
			ParentID:      sym.ParentID,
			Signature:     sym.Signature,
			Documentation: sym.Documentation,
			StartLine:     sym.StartLine,
			EndLine:       sym.EndLine,
			StartByte:     sym.StartByte,
			EndByte:       sym.EndByte,
			Language:      sym.Language,
			FilePath:      sym.FilePath,
		})
	}
	return out, lang, nil
}
