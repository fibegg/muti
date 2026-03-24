package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// EmptyCollection replaces non-empty array/hash/dict literals with empty ones.
type EmptyCollection struct{}

func (o *EmptyCollection) Name() string { return "empty_collection" }

func (o *EmptyCollection) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Node types for collection literals across languages
	collectionTypes := []string{
		"array",                   // Ruby, JS, Python
		"hash",                    // Ruby
		"dictionary",              // Python
		"object",                  // JS/TS
		"composite_literal",       // Go (partial — we handle below)
		"list",                    // Python
		"array_expression",        // TS
	}

	targets := collectNodesByTypes(root, collectionTypes)

	// Filter to non-empty collections (must have > 0 meaningful children)
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		text := nodeText(n, source)
		kind := n.Kind()
		if kind == "composite_literal" {
			// Go composite literals: only target if they have a literal_value child with content
			continue // Skip Go composite literals — they include the type and are complex to empty safely
		}
		// Must have some content (not just brackets)
		if len(text) > 2 {
			candidates = append(candidates, n)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	// Determine the empty replacement based on type
	var replacement string
	switch node.Kind() {
	case "hash", "dictionary", "object":
		replacement = "{}"
	default:
		replacement = "[]"
	}

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	preview := old
	if len(preview) > 35 {
		preview = preview[:32] + "..."
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Emptied collection " + preview + " → " + replacement,
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
