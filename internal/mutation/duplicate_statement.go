package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// DuplicateStatement duplicates a statement to test idempotency assumptions.
type DuplicateStatement struct{}

func (o *DuplicateStatement) Name() string { return "duplicate_statement" }

func (o *DuplicateStatement) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Statement node types across languages
	stmtTypes := []string{
		"expression_statement",  // JS/TS/Python/Go
		"assignment",            // Ruby, Python
		"call",                  // Ruby
		"return_statement",      // JS/TS/Python
		"short_var_declaration", // Go
	}

	targets := collectNodesByTypes(root, stmtTypes)

	// Filter: avoid very short or trivial statements
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		text := nodeText(n, source)
		if len(text) > 3 { // Skip trivial stuff
			candidates = append(candidates, n)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	// Determine the line separator to use
	separator := "\n"

	// Insert a duplicate right after the statement
	duplicate := old + separator + old
	mutated := replaceRange(source, node.StartByte(), node.EndByte(), duplicate)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	preview := old
	if len(preview) > 35 {
		preview = preview[:32] + "..."
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Duplicated statement `" + preview + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     duplicate,
	}, mutated, nil
}
