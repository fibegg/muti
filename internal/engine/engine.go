package engine

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
	"github.com/fibegg/muti/internal/mutation"
)

// Engine performs mutations on source files.
type Engine struct {
	Dirs       []string
	Extensions []string
	ForceLang  string
	Operators  []mutation.Operator
	Verbose    bool
}

// NewEngine creates a mutation engine.
func NewEngine(dirs []string, extensions []string, forceLang string, operators []mutation.Operator, verbose bool) *Engine {
	return &Engine{
		Dirs:       dirs,
		Extensions: extensions,
		ForceLang:  forceLang,
		Operators:  operators,
		Verbose:    verbose,
	}
}

// Mutate applies count random mutations and returns the results.
func (e *Engine) Mutate(count int) ([]*mutation.MutationResult, error) {
	files, err := e.discoverFiles()
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no source files found in directories: %v", e.Dirs)
	}
	if len(e.Operators) == 0 {
		return nil, fmt.Errorf("no operators available")
	}

	var results []*mutation.MutationResult

	// Try to apply exactly `count` mutations (or fewer if we fail repeatedly)
	maxRetries := count * 10
	retries := 0

	for len(results) < count && retries < maxRetries {
		retries++

		file := files[rand.Intn(len(files))]
		langCfg, ok := language.DetectFromFile(file)
		if !ok && e.ForceLang == "" {
			continue // Should be filtered by discoverFiles, but just in case
		}
		if e.ForceLang != "" {
			langCfg, _ = language.Get(e.ForceLang)
		}
		if langCfg == nil {
			continue
		}

		source, err := os.ReadFile(file)
		if err != nil {
			if e.Verbose {
				fmt.Fprintf(os.Stderr, "  [DEBUG] skip %s: read error %v\n", file, err)
			}
			continue
		}
		if len(source) == 0 {
			continue
		}

		parser := tree_sitter.NewParser()
		if err := parser.SetLanguage(langCfg.TSLanguage); err != nil {
			parser.Close()
			continue
		}

		tree := parser.Parse(source, nil)
		if tree == nil {
			parser.Close()
			continue
		}

		// Pick random operator
		op := e.Operators[rand.Intn(len(e.Operators))]

		// Apply mutation
		result, mutated, err := op.Apply(source, tree, langCfg)
		tree.Close()
		parser.Close()

		if err != nil {
			if e.Verbose {
				fmt.Fprintf(os.Stderr, "  [DEBUG] %s @ %s error: %v\n", op.Name(), file, err)
			}
			continue
		}
		if result == nil {
			continue // No matching AST nodes for this operator in this file
		}

		// Inject file path and diff
		result.File = file
		result.Diff = generateDiff(file, source, mutated)

		// Save the mutated file
		if err := os.WriteFile(file, mutated, 0o644); err != nil {
			return nil, fmt.Errorf("failed to write mutated file %s: %w", file, err)
		}

		results = append(results, result)
	}

	return results, nil
}

// discoverFiles finds all supported source files in target directories.
func (e *Engine) discoverFiles() ([]string, error) {
	var files []string

	for _, dir := range e.Dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil //nolint:nilerr // intentional: skip unreadable paths, continue walking
			}
			if info.IsDir() {
				name := info.Name()
				if name == ".git" || name == "node_modules" || name == "vendor" || name == "bin" {
					return filepath.SkipDir
				}
				return nil
			}

			// If specific extensions requested, filter by them
			if len(e.Extensions) > 0 {
				ext := strings.TrimPrefix(filepath.Ext(path), ".")
				found := false
				for _, e := range e.Extensions {
					if ext == e {
						found = true
						break
					}
				}
				if found {
					files = append(files, path)
				}
				return nil
			}

			// Otherwise auto-detect based on supported languages
			if _, ok := language.DetectFromFile(path); ok {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking %s: %w", dir, err)
		}
	}

	return files, nil
}

func generateDiff(filePath string, original, mutated []byte) string {
	origLines := strings.Split(string(original), "\n")
	newLines := strings.Split(string(mutated), "\n")

	// Find changed line ranges
	type change struct {
		origStart, origEnd int // 0-indexed, exclusive end
		newStart, newEnd   int
	}

	var changes []change
	oi, ni := 0, 0
	for oi < len(origLines) && ni < len(newLines) {
		if origLines[oi] == newLines[ni] {
			oi++
			ni++
			continue
		}
		// Start of a difference
		cs := change{origStart: oi, newStart: ni}
		// Find end of difference
		for oi < len(origLines) && ni < len(newLines) && origLines[oi] != newLines[ni] {
			oi++
			ni++
		}
		cs.origEnd = oi
		cs.newEnd = ni
		changes = append(changes, cs)
	}
	// Handle trailing lines
	if oi < len(origLines) || ni < len(newLines) {
		changes = append(changes, change{
			origStart: oi, origEnd: len(origLines),
			newStart: ni, newEnd: len(newLines),
		})
	}

	if len(changes) == 0 {
		return ""
	}

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	diff.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

	const context = 3

	for _, c := range changes {
		// Context window
		ctxStart := c.origStart - context
		if ctxStart < 0 {
			ctxStart = 0
		}
		ctxOrigEnd := c.origEnd + context
		if ctxOrigEnd > len(origLines) {
			ctxOrigEnd = len(origLines)
		}
		ctxNewStart := c.newStart - context
		if ctxNewStart < 0 {
			ctxNewStart = 0
		}
		ctxNewEnd := c.newEnd + context
		if ctxNewEnd > len(newLines) {
			ctxNewEnd = len(newLines)
		}

		origLen := (ctxOrigEnd - ctxStart)
		newLen := (ctxNewEnd - ctxNewStart)
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", ctxStart+1, origLen, ctxNewStart+1, newLen))

		// Context before
		for i := ctxStart; i < c.origStart; i++ {
			diff.WriteString(" " + origLines[i] + "\n")
		}
		// Removed lines
		for i := c.origStart; i < c.origEnd; i++ {
			diff.WriteString("-" + origLines[i] + "\n")
		}
		// Added lines
		for i := c.newStart; i < c.newEnd; i++ {
			diff.WriteString("+" + newLines[i] + "\n")
		}
		// Context after
		for i := c.origEnd; i < ctxOrigEnd; i++ {
			diff.WriteString(" " + origLines[i] + "\n")
		}
	}

	return diff.String()
}
