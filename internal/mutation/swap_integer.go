package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapInteger swaps 0↔1 integer literals.
type SwapInteger struct{}

func (o *SwapInteger) Name() string { return "swap_integer" }

func (o *SwapInteger) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	intTypes := []string{"integer", "integer_literal", "int_literal", "number"}
	targets := collectNodesByTypes(root, intTypes)

	// Filter to 0 or 1
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		text := nodeText(n, source)
		if text == "0" || text == "1" {
			candidates = append(candidates, n)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)
	var replacement string
	if old == "0" {
		replacement = "1"
	} else {
		replacement = "0"
	}

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped integer `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
