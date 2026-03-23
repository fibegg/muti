package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fibegg/muti/internal/mutation"
)

func TestDiscoverFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "test.rb"), []byte("puts 'hi'"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b,c"), 0o644)

	_ = os.MkdirAll(filepath.Join(dir, "vendor"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "vendor", "lib.go"), []byte("package lib"), 0o644)

	_ = os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(""), 0o644)

	eng := NewEngine([]string{dir}, nil, "", nil, false)
	files, err := eng.discoverFiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if strings.Contains(f, "vendor") {
			t.Errorf("vendor file should be skipped: %s", f)
		}
		if strings.Contains(f, ".csv") {
			t.Errorf("csv file should be skipped: %s", f)
		}
	}
}

func TestDiscoverFiles_WithExtensions(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "test.rb"), []byte("puts 'hi'"), 0o644)

	eng := NewEngine([]string{dir}, []string{"rb"}, "", nil, false)
	files, err := eng.discoverFiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
	if len(files) > 0 && !strings.HasSuffix(files[0], ".rb") {
		t.Errorf("expected .rb file, got %s", files[0])
	}
}

func TestDiscoverFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	eng := NewEngine([]string{dir}, nil, "", nil, false)
	files, err := eng.discoverFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestGenerateDiff(t *testing.T) {
	orig := []byte("line1\nline2\nline3\n")
	mutd := []byte("line1\nchanged\nline3\n")
	diff := generateDiff("test.go", orig, mutd)
	if !strings.Contains(diff, "-line2") || !strings.Contains(diff, "+changed") {
		t.Errorf("diff missing expected content:\n%s", diff)
	}
	if !strings.Contains(diff, "--- a/test.go") {
		t.Errorf("diff missing header:\n%s", diff)
	}
}

func TestMutate_NoFiles(t *testing.T) {
	dir := t.TempDir()
	ops := []mutation.Operator{&mutation.SwapBoolean{}}
	eng := NewEngine([]string{dir}, nil, "", ops, false)
	_, err := eng.Mutate(1)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}
