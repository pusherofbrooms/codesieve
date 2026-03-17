package filetype

import (
	"path/filepath"
	"strings"
)

func IsTerraformJSONPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, ".tf.json") || strings.HasSuffix(base, ".tfvars.json")
}
