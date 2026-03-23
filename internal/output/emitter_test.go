package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fibegg/muti/internal/mutation"
)

func TestEmitEcho(t *testing.T) {
	emitter := NewEmitter("echo", "", false)
	result := &mutation.MutationResult{
		File:     "test.go",
		Operator: "swap_boolean",
		Line:     1,
	}

	if err := emitter.Emit(result); err != nil {
		t.Fatal(err)
	}
}

func TestSaveToFile(t *testing.T) {
	dir := t.TempDir()
	emitter := NewEmitter("", dir, false)
	result := &mutation.MutationResult{
		File:     "test.go",
		Operator: "swap_boolean",
		Line:     10,
		Column:   5,
	}

	if err := emitter.Emit(result); err != nil {
		t.Fatal(err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file, got %d", len(entries))
	}
}

func TestSaveToFile_UniqueNames(t *testing.T) {
	dir := t.TempDir()
	emitter := NewEmitter("", dir, false)
	result := &mutation.MutationResult{
		File:     "test.go",
		Operator: "swap_boolean",
		Line:     10,
		Column:   5,
	}

	_ = emitter.Emit(result)
	_ = emitter.Emit(result)

	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected 2 unique files, got %d: %v", len(entries), names)
	}
}

func TestEmitToolCommand(t *testing.T) {
	emitter := NewEmitter("cat", "", false)
	result := &mutation.MutationResult{
		File:     "test.go",
		Operator: "swap_boolean",
	}

	if err := emitter.Emit(result); err != nil {
		t.Fatal(err)
	}
}

func TestSaveToFile_SubDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	emitter := NewEmitter("", dir, false)
	result := &mutation.MutationResult{
		Operator: "test",
		Line:     1,
	}

	if err := emitter.Emit(result); err != nil {
		t.Fatalf("should create nested dirs: %v", err)
	}
}
