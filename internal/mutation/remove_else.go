package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveElse removes the else/elif branch from if statements.
type RemoveElse struct{}

func (o *RemoveElse) Name() string { return "remove_else" }

func (o *RemoveElse) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Find else clauses across languages
	elseTypes := []string{
		"else_clause",    // Python, Ruby
		"else",           // Ruby, Go
		"else_statement", // some grammars
	}

	targets := collectNodesByTypes(root, elseTypes)

	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	// Remove the else clause entirely
	mutated := replaceRange(source, node.StartByte(), node.EndByte(), "")
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	preview := old
	if len(preview) > 40 {
		preview = preview[:37] + "..."
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Removed else branch",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    preview,
		Mutated:     "(removed)",
	}, mutated, nil
}
