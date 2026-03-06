package app

import "strings"

func SliceLines(content string, startLine, endLine int) (string, int, int, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", 0, 0, nil
	}
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 || endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine || startLine > len(lines) {
		return "", 0, 0, ErrNotFound("INVALID_RANGE", "requested line range is invalid")
	}
	chunk := strings.Join(lines[startLine-1:endLine], "\n")
	return chunk, startLine, endLine, nil
}
