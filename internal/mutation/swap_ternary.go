package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapTernary swaps the branches of ternary/conditional expressions (a ? b : c → a ? c : b).
type SwapTernary struct{}

func (o *SwapTernary) Name() string { return "swap_ternary" }

func (o *SwapTernary) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	ternaryTypes := []string{
		"conditional_expression", // JS/TS
		"conditional",            // Ruby
		"ternary",                // some grammars
	}

	targets := collectNodesByTypes(root, ternaryTypes)

	var candidates []ternaryPair
	for _, n := range targets {
		pair := extractTernaryPair(n, source)
		if pair != nil {
			candidates = append(candidates, *pair)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	pair := *t

	// Swap consequence and alternative (replace later one first)
	var mutated []byte
	if pair.consequenceStart > pair.alternativeStart {
		mutated = replaceRange(source, pair.consequenceStart, pair.consequenceEnd, pair.alternativeText)
		mutated = replaceRange(mutated, pair.alternativeStart, pair.alternativeEnd, pair.consequenceText)
	} else {
		mutated = replaceRange(source, pair.alternativeStart, pair.alternativeEnd, pair.consequenceText)
		mutated = replaceRange(mutated, pair.consequenceStart, pair.consequenceEnd, pair.alternativeText)
	}

	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped ternary branches `" + pair.consequenceText + "` ↔ `" + pair.alternativeText + "`",
		Line:        pair.line,
		Column:      pair.column,
		Original:    pair.consequenceText,
		Mutated:     pair.alternativeText,
	}, mutated, nil
}

type ternaryPair struct {
	consequenceText  string
	alternativeText  string
	consequenceStart uint
	consequenceEnd   uint
	alternativeStart uint
	alternativeEnd   uint
	line             uint
	column           uint
}

func extractTernaryPair(node *tree_sitter.Node, source []byte) *ternaryPair {
	// Ternary: condition ? consequence : alternative
	// Typically 5 children: condition, ?, consequence, :, alternative
	// Or via named fields

	// Collect non-punctuation children
	var parts []*tree_sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "?" || kind == ":" {
			continue
		}
		parts = append(parts, child)
	}

	// Need at least 3 parts: condition, consequence, alternative
	if len(parts) < 3 {
		return nil
	}

	consequence := parts[1]
	alternative := parts[2]

	// Don't swap if they're identical
	consText := nodeText(consequence, source)
	altText := nodeText(alternative, source)
	if consText == altText {
		return nil
	}

	return &ternaryPair{
		consequenceText:  consText,
		alternativeText:  altText,
		consequenceStart: consequence.StartByte(),
		consequenceEnd:   consequence.EndByte(),
		alternativeStart: alternative.StartByte(),
		alternativeEnd:   alternative.EndByte(),
		line:             node.StartPosition().Row + 1,
		column:           node.StartPosition().Column,
	}
}
