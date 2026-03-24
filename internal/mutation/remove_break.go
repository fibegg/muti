package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveBreak removes break, continue, and next statements from loops.
type RemoveBreak struct{}

func (o *RemoveBreak) Name() string { return "remove_break" }

func (o *RemoveBreak) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	breakTypes := []string{
		"break_statement",    // JS/TS, Python, Go
		"continue_statement", // JS/TS, Python, Go
		"break",              // Ruby
		"next",               // Ruby
	}

	targets := collectNodesByTypes(root, breakTypes)

	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), "")
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Removed `" + old + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     "(removed)",
	}, mutated, nil
}
