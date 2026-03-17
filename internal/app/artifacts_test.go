package app

import "testing"

func TestIsGeneratedArtifactPath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		isDir bool
		want  bool
	}{
		{name: "terraform dir", path: ".terraform", isDir: true, want: true},
		{name: "nested terraform dir", path: "env/.terraform/plugins", isDir: true, want: true},
		{name: "regular dir", path: "terraform", isDir: true, want: false},
		{name: "tfstate", path: "terraform.tfstate", isDir: false, want: true},
		{name: "tfstate backup", path: "prod.tfstate.backup", isDir: false, want: true},
		{name: "terraform file", path: "main.tf", isDir: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGeneratedArtifactPath(tc.path, tc.isDir); got != tc.want {
				t.Fatalf("isGeneratedArtifactPath(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.want)
			}
		})
	}
}
