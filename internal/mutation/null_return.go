package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// NullReturn changes `return expr` to `return nil/None/null`.
type NullReturn struct{}

func (o *NullReturn) Name() string { return "null_return" }

func (o *NullReturn) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	retTypes := []string{"return", "return_statement"}
	targets := collectNodesByTypes(root, retTypes)

	// Filter to returns that have an expression (not bare returns)
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		if n.NamedChildCount() > 0 {
			// Has a return value
			firstChild := n.NamedChild(0)
			if firstChild != nil && firstChild.Kind() != "nil" && firstChild.Kind() != "none" && firstChild.Kind() != "null" {
				candidates = append(candidates, n)
			}
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t

	// Build the replacement return statement
	var replacement string
	switch lang.Name {
	case "ruby":
		replacement = "return nil"
	case "python":
		replacement = "return None"
	case "go":
		replacement = "return"
	default: // javascript, typescript
		replacement = "return null"
	}

	old := nodeText(node, source)
	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Changed return to `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
