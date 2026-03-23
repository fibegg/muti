package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveErrorHandler removes rescue/except/catch blocks.
type RemoveErrorHandler struct{}

func (o *RemoveErrorHandler) Name() string { return "remove_error_handler" }

func (o *RemoveErrorHandler) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Different languages have different error handling constructs
	handlerTypes := []string{
		"rescue",              // Ruby
		"except_clause",       // Python
		"catch_clause",        // JS/TS
		"rescue_block",        // Ruby alternative
		"except_group_clause", // Python 3.11+
	}

	targets := collectNodesByTypes(root, handlerTypes)

	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	start := node.StartByte()
	end := node.EndByte()
	if int(end) < len(source) && source[end] == '\n' {
		end++
	}

	mutated := replaceRange(source, start, end, "")
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Removed error handler block",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     "",
	}, mutated, nil
}
