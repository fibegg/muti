package mutation

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// FlipConditional swaps if/else branches.
type FlipConditional struct{}

func (o *FlipConditional) Name() string { return "flip_conditional" }

func (o *FlipConditional) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	ifTypes := []string{"if", "if_statement", "if_expression"}
	targets := collectNodesByTypes(root, ifTypes)

	// Filter to only those with an else branch
	var withElse []*tree_sitter.Node
	for _, n := range targets {
		consequence := n.ChildByFieldName("consequence")
		alternative := n.ChildByFieldName("alternative")
		if consequence != nil && alternative != nil {
			withElse = append(withElse, n)
		}
	}

	t := pickRandom(withElse)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	consequence := node.ChildByFieldName("consequence")
	alternative := node.ChildByFieldName("alternative")

	consText := nodeText(consequence, source)
	altText := nodeText(alternative, source)

	// Replace back-to-front using ORIGINAL source positions.
	// Alternative comes after consequence in the source, so replace it first
	// to keep consequence positions valid.
	mutated := replaceRange(source, alternative.StartByte(), alternative.EndByte(), consText)
	mutated = replaceRange(mutated, consequence.StartByte(), consequence.EndByte(), altText)

	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: fmt.Sprintf("Flipped conditional branches at line %d", node.StartPosition().Row+1),
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    consText,
		Mutated:     altText,
	}, mutated, nil
}
