package mutation

import (
	"fmt"
	"strconv"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// ChangeArrayIndex changes array/subscript indices (e.g. arr[0] → arr[1]).
type ChangeArrayIndex struct{}

func (o *ChangeArrayIndex) Name() string { return "change_array_index" }

func (o *ChangeArrayIndex) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Subscript/index access node types
	subTypes := []string{
		"subscript",            // Python
		"index_expression",     // Go
		"element_reference",    // Ruby
		"subscript_expression", // JS/TS
		"member_expression",    // skip — this is dot access
	}
	// Only use actual index types
	actualTypes := []string{
		"subscript",
		"index_expression",
		"element_reference",
		"subscript_expression",
	}

	indexNodes := collectNodesByTypes(root, actualTypes)

	// Find integer index children within subscript expressions
	var candidates []*tree_sitter.Node
	for _, n := range indexNodes {
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(uint(i))
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "integer" || ck == "int_literal" || ck == "integer_literal" || ck == "number" {
				candidates = append(candidates, child)
			}
		}
	}

	// Also walk looking for bracket indices in languages where
	// the grammar nests differently
	_ = subTypes // suppress unused warning

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	val, err := strconv.ParseInt(old, 0, 64)
	if err != nil {
		return nil, nil, err
	}

	// Change index: 0→1, anything else→0
	var newVal int64
	if val == 0 {
		newVal = 1
	} else {
		newVal = 0
	}
	replacement := fmt.Sprintf("%d", newVal)

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Changed index `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
