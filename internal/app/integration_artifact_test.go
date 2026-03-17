package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexSkipsTerraformArtifacts(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	workdir := filepath.Join(t.TempDir(), "workrepo-artifacts")
	if err := os.MkdirAll(filepath.Join(workdir, ".terraform", "providers"), 0o755); err != nil {
		t.Fatalf("MkdirAll .terraform: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, ".terraform", "providers", "state.tf"), []byte("resource \"aws_s3_bucket\" \"x\" {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile state.tf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "terraform.tfstate"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile terraform.tfstate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "main.tf"), []byte("resource \"aws_s3_bucket\" \"app\" {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile main.tf: %v", err)
	}

	res, err := svc.Index(ctx, workdir, IndexOptions{})
	if err != nil {
		t.Fatalf("Index error: %v", err)
	}
	if res.FilesIndexed != 1 {
		t.Fatalf("expected only main.tf to be indexed, got %+v", res)
	}

	artifactSkips := 0
	for _, d := range res.FilesSkipped {
		if d.Code == "SKIPPED_ARTIFACT" {
			artifactSkips++
		}
	}
	if artifactSkips < 2 {
		t.Fatalf("expected at least two SKIPPED_ARTIFACT diagnostics, got %d (%+v)", artifactSkips, res.FilesSkipped)
	}
}
