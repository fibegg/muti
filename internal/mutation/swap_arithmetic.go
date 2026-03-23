package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapArithmetic swaps +↔- and *↔/ operators.
type SwapArithmetic struct{}

func (o *SwapArithmetic) Name() string { return "swap_arithmetic" }

func (o *SwapArithmetic) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	swaps := map[string]string{
		"+": "-", "-": "+",
		"*": "/", "/": "*",
	}

	var candidates []*tree_sitter.Node
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		kind := n.Kind()
		if kind == "binary" || kind == "binary_expression" || kind == "binary_operator" {
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(uint(i))
				if child != nil {
					text := nodeText(child, source)
					if _, ok := swaps[text]; ok {
						candidates = append(candidates, child)
					}
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(uint(i))
			if child != nil {
				walk(child)
			}
		}
	}
	walk(root)

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)
	replacement := swaps[old]

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped arithmetic `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
