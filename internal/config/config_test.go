package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Dirs) != 1 || cfg.Dirs[0] != "." {
		t.Errorf("Dirs = %v, want [\".\"]", cfg.Dirs)
	}
	if cfg.Rounds != 10 {
		t.Errorf("Rounds = %d, want 10", cfg.Rounds)
	}
	if cfg.Tool != "echo" {
		t.Errorf("Tool = %q, want \"echo\"", cfg.Tool)
	}
	if cfg.TestMode {
		t.Error("TestMode should default to false")
	}
	if cfg.Verbose {
		t.Error("Verbose should default to false")
	}
	if cfg.Mutations != 0 {
		t.Errorf("Mutations = %d, want 0 (random)", cfg.Mutations)
	}
}
