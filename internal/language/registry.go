package language

import (
	"fmt"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// LangConfig describes a supported language.
type LangConfig struct {
	Name       string
	Extensions []string
	TSLanguage *tree_sitter.Language
	// NullLiteral is the null/nil representation for this language.
	NullLiteral string
}

var registry = map[string]*LangConfig{
	"ruby": {
		Name:        "ruby",
		Extensions:  []string{".rb"},
		TSLanguage:  tree_sitter.NewLanguage(tree_sitter_ruby.Language()),
		NullLiteral: "nil",
	},
	"go": {
		Name:        "go",
		Extensions:  []string{".go"},
		TSLanguage:  tree_sitter.NewLanguage(tree_sitter_go.Language()),
		NullLiteral: "nil",
	},
	"python": {
		Name:        "python",
		Extensions:  []string{".py"},
		TSLanguage:  tree_sitter.NewLanguage(tree_sitter_python.Language()),
		NullLiteral: "None",
	},
	"javascript": {
		Name:        "javascript",
		Extensions:  []string{".js", ".jsx", ".mjs", ".cjs"},
		TSLanguage:  tree_sitter.NewLanguage(tree_sitter_javascript.Language()),
		NullLiteral: "null",
	},
	"typescript": {
		Name:        "typescript",
		Extensions:  []string{".ts", ".tsx"},
		TSLanguage:  tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript()),
		NullLiteral: "null",
	},
}

// extToLang maps file extension to language name.
var extToLang map[string]string

func init() {
	extToLang = make(map[string]string)
	for name, cfg := range registry {
		for _, ext := range cfg.Extensions {
			extToLang[ext] = name
		}
	}
}

// Get returns the LangConfig by language name.
func Get(name string) (*LangConfig, error) {
	name = strings.ToLower(name)
	// Allow common aliases
	switch name {
	case "js":
		name = "javascript"
	case "ts":
		name = "typescript"
	case "py":
		name = "python"
	case "rb":
		name = "ruby"
	}
	cfg, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", name)
	}
	return cfg, nil
}

// DetectFromExtension returns the LangConfig for a given file extension.
func DetectFromExtension(ext string) (*LangConfig, bool) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	name, ok := extToLang[ext]
	if !ok {
		return nil, false
	}
	return registry[name], true
}

// DetectFromFile returns the LangConfig for a given file path.
func DetectFromFile(path string) (*LangConfig, bool) {
	ext := filepath.Ext(path)
	return DetectFromExtension(ext)
}

// ListLanguages returns all supported language names.
func ListLanguages() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ListAll returns all language configs.
func ListAll() map[string]*LangConfig {
	return registry
}
