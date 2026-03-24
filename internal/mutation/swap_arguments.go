package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapArguments swaps the first two arguments of a function call.
type SwapArguments struct{}

func (o *SwapArguments) Name() string { return "swap_arguments" }

func (o *SwapArguments) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Find function call nodes
	callTypes := []string{
		"call",            // Ruby
		"call_expression", // JS/TS/Go
		"argument_list",   // Go, Python
		"method_call",     // Ruby
	}

	var candidates []argPair
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		kind := n.Kind()
		// Look for argument lists inside calls
		if kind == "arguments" || kind == "argument_list" || kind == "method_parameters" {
			pair := extractArgPair(n, source)
			if pair != nil {
				candidates = append(candidates, *pair)
			}
		}
		// For call_expression/call, look for their argument children
		if contains(callTypes, kind) {
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(uint(i))
				if child != nil {
					ck := child.Kind()
					if ck == "arguments" || ck == "argument_list" {
						pair := extractArgPair(child, source)
						if pair != nil {
							candidates = append(candidates, *pair)
						}
					}
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(uint(i))
			if child != nil {
				walk(child)
			}
		}
	}
	walk(root)

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	pair := *t

	// Swap: replace later one first to preserve offsets
	var mutated []byte
	if pair.firstStart > pair.secondStart {
		mutated = replaceRange(source, pair.firstStart, pair.firstEnd, pair.secondText)
		mutated = replaceRange(mutated, pair.secondStart, pair.secondEnd, pair.firstText)
	} else {
		mutated = replaceRange(source, pair.secondStart, pair.secondEnd, pair.firstText)
		mutated = replaceRange(mutated, pair.firstStart, pair.firstEnd, pair.secondText)
	}

	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped arguments `" + pair.firstText + "` ↔ `" + pair.secondText + "`",
		Line:        pair.line,
		Column:      pair.column,
		Original:    pair.firstText + ", " + pair.secondText,
		Mutated:     pair.secondText + ", " + pair.firstText,
	}, mutated, nil
}

type argPair struct {
	firstText   string
	secondText  string
	firstStart  uint
	firstEnd    uint
	secondStart uint
	secondEnd   uint
	line        uint
	column      uint
}

func extractArgPair(node *tree_sitter.Node, source []byte) *argPair {
	// Collect non-punctuation children (skip commas, parens)
	var args []*tree_sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "," || kind == "(" || kind == ")" {
			continue
		}
		args = append(args, child)
	}

	if len(args) < 2 {
		return nil
	}

	first := args[0]
	second := args[1]

	return &argPair{
		firstText:   nodeText(first, source),
		secondText:  nodeText(second, source),
		firstStart:  first.StartByte(),
		firstEnd:    first.EndByte(),
		secondStart: second.StartByte(),
		secondEnd:   second.EndByte(),
		line:        node.StartPosition().Row + 1,
		column:      node.StartPosition().Column,
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
