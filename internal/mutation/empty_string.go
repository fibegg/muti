package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// EmptyString replaces non-empty string literals with "".
type EmptyString struct{}

func (o *EmptyString) Name() string { return "empty_string" }

func (o *EmptyString) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	strTypes := []string{"string", "string_literal", "interpreted_string_literal", "raw_string_literal", "string_content"}
	targets := collectNodesByTypes(root, strTypes)

	// Filter to non-empty strings
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		text := nodeText(n, source)
		// Must be longer than just quotes
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

	// Determine the quote style used
	replacement := `""`
	if len(old) > 0 && old[0] == '\'' {
		replacement = `''`
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
		Description: "Replaced string " + preview + " with " + replacement,
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
