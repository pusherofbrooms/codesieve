package app

import "testing"

func TestSliceLines(t *testing.T) {
	content := "a\nb\nc\nd"
	chunk, start, end, err := SliceLines(content, 2, 3)
	if err != nil {
		t.Fatalf("SliceLines error: %v", err)
	}
	if start != 2 || end != 3 {
		t.Fatalf("range = %d-%d", start, end)
	}
	if chunk != "b\nc" {
		t.Fatalf("chunk = %q", chunk)
	}
}

func TestSliceLinesInvalidRange(t *testing.T) {
	_, _, _, err := SliceLines("a\nb", 3, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
