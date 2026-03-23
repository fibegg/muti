package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapBoolean swaps true↔false.
type SwapBoolean struct{}

func (o *SwapBoolean) Name() string { return "swap_boolean" }

func (o *SwapBoolean) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	targets := collectNodesByTypes(root, []string{"true", "false"})
	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)
	var replacement string
	if node.Kind() == "true" {
		replacement = "false"
	} else {
		replacement = "true"
	}

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
