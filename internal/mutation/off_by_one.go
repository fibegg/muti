package mutation

import (
	"fmt"
	"strconv"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// OffByOne changes integer literals by +1 or -1 to catch boundary/fence-post bugs.
type OffByOne struct{}

func (o *OffByOne) Name() string { return "off_by_one" }

func (o *OffByOne) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	intTypes := []string{"integer", "int_literal", "integer_literal"}
	targets := collectNodesByTypes(root, intTypes)

	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	val, err := strconv.ParseInt(old, 0, 64)
	if err != nil {
		return nil, nil, err
	}

	// Add 1 to the value (the simplest off-by-one)
	newVal := val + 1
	replacement := fmt.Sprintf("%d", newVal)

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Off-by-one: `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
