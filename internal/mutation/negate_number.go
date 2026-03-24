package mutation

import (
	"fmt"
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// NegateNumber negates numeric literals (x → -x) to test sign sensitivity.
type NegateNumber struct{}

func (o *NegateNumber) Name() string { return "negate_number" }

func (o *NegateNumber) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	numTypes := []string{
		"integer",                    // Ruby, Python
		"int_literal",                // Go
		"integer_literal",            // Go
		"float",                      // Ruby, Python
		"float_literal",              // Go
		"number",                     // JS/TS
		"interpreted_string_literal", // skip — not a number
	}
	// Only use number-like types
	actualTypes := []string{}
	for _, t := range numTypes {
		if !strings.Contains(t, "string") {
			actualTypes = append(actualTypes, t)
		}
	}

	targets := collectNodesByTypes(root, actualTypes)

	// Filter to non-zero values (negating 0 is pointless)
	var candidates []*tree_sitter.Node
	for _, n := range targets {
		text := nodeText(n, source)
		val, err := strconv.ParseFloat(text, 64)
		if err == nil && val != 0 {
			candidates = append(candidates, n)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	// Check if already negative by looking at parent (unary_expression with -)
	replacement := fmt.Sprintf("-%s", old)

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Negated `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
