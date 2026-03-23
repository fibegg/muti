package language

import (
	"testing"
)

func TestGet_KnownLanguages(t *testing.T) {
	for _, name := range []string{"ruby", "go", "python", "javascript", "typescript"} {
		cfg, err := Get(name)
		if err != nil {
			t.Errorf("Get(%q) error: %v", name, err)
		}
		if cfg == nil {
			t.Errorf("Get(%q) returned nil", name)
			continue
		}
		if cfg.TSLanguage == nil {
			t.Errorf("Get(%q).TSLanguage is nil", name)
		}
	}
}

func TestGet_Aliases(t *testing.T) {
	tests := map[string]string{
		"js": "javascript",
		"ts": "typescript",
		"py": "python",
		"rb": "ruby",
	}
	for alias, expected := range tests {
		cfg, err := Get(alias)
		if err != nil {
			t.Errorf("Get(%q) error: %v", alias, err)
		}
		if cfg.Name != expected {
			t.Errorf("Get(%q).Name = %q, want %q", alias, cfg.Name, expected)
		}
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("fortran")
	if err == nil {
		t.Error("expected error for unknown language")
	}
}

func TestDetectFromFile(t *testing.T) {
	tests := map[string]string{
		"app.go":        "go",
		"test.rb":       "ruby",
		"script.py":     "python",
		"index.js":      "javascript",
		"component.tsx": "typescript",
		"util.mjs":      "javascript",
	}
	for file, expected := range tests {
		cfg, ok := DetectFromFile(file)
		if !ok {
			t.Errorf("DetectFromFile(%q) failed", file)
			continue
		}
		if cfg.Name != expected {
			t.Errorf("DetectFromFile(%q).Name = %q, want %q", file, cfg.Name, expected)
		}
	}
}

func TestDetectFromFile_Unknown(t *testing.T) {
	_, ok := DetectFromFile("data.csv")
	if ok {
		t.Error("expected false for unknown extension")
	}
}

func TestListLanguages(t *testing.T) {
	langs := ListLanguages()
	if len(langs) != 5 {
		t.Errorf("expected 5 languages, got %d: %v", len(langs), langs)
	}
}
