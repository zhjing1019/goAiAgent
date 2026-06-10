package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	base := BuildSystemPrompt(false)
	if base == "" {
		t.Fatal("empty prompt")
	}
	rag := BuildSystemPrompt(true)
	if !strings.Contains(rag, "search_knowledge") {
		t.Fatalf("rag prompt missing search_knowledge")
	}
}

func TestListDocFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# hi"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := listDocFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "a.md" {
		t.Fatalf("unexpected files: %v", files)
	}
}
