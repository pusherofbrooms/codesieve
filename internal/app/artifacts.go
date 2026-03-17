package app

import (
	"path/filepath"
	"strings"
)

func isGeneratedArtifactPath(relPath string, isDir bool) bool {
	rel := strings.ToLower(filepath.ToSlash(relPath))
	base := strings.ToLower(filepath.Base(rel))

	if isDir {
		if base == ".terraform" {
			return true
		}
		for _, part := range strings.Split(rel, "/") {
			if part == ".terraform" {
				return true
			}
		}
		return false
	}

	if strings.HasSuffix(base, ".tfstate") || strings.HasSuffix(base, ".tfstate.backup") {
		return true
	}
	return false
}
