package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapIfElse swaps the if-body and else-body to test branch coverage.
type SwapIfElse struct{}

func (o *SwapIfElse) Name() string { return "swap_if_else" }

func (o *SwapIfElse) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	ifTypes := []string{"if_statement", "if", "if_expression"}
	targets := collectNodesByTypes(root, ifTypes)

	// Filter to those that have both consequence and alternative
	var candidates []ifPair
	for _, n := range targets {
		pair := extractIfElsePair(n, source)
		if pair != nil {
			candidates = append(candidates, *pair)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	pair := *t

	// Swap: replace the if-body with the else-body and vice versa
	// We must replace the later one first to preserve byte offsets
	var mutated []byte
	if pair.thenStart > pair.elseStart {
		mutated = replaceRange(source, pair.thenStart, pair.thenEnd, pair.elseText)
		mutated = replaceRange(mutated, pair.elseStart, pair.elseEnd, pair.thenText)
	} else {
		mutated = replaceRange(source, pair.elseStart, pair.elseEnd, pair.thenText)
		mutated = replaceRange(mutated, pair.thenStart, pair.thenEnd, pair.elseText)
	}

	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Swapped if/else branches",
		Line:        pair.line,
		Column:      pair.column,
		Original:    "if-branch",
		Mutated:     "else-branch",
	}, mutated, nil
}

type ifPair struct {
	thenText  string
	elseText  string
	thenStart uint
	thenEnd   uint
	elseStart uint
	elseEnd   uint
	line      uint
	column    uint
}

func extractIfElsePair(node *tree_sitter.Node, source []byte) *ifPair {
	kind := node.Kind()
	var thenNode, elseNode *tree_sitter.Node

	switch kind {
	case "if_statement", "if_expression":
		// Languages like Python, Ruby: look for named children
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "block" || ck == "then" || ck == "statement_block" || ck == "consequence" {
				if thenNode == nil {
					thenNode = child
				}
			}
			if ck == "else_clause" || ck == "else" || ck == "alternative" {
				elseNode = child
			}
		}
	case "if":
		// Ruby-style: named children consequence/alternative
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "then" {
				thenNode = child
			}
			if ck == "else" {
				elseNode = child
			}
		}
	}

	if thenNode == nil || elseNode == nil {
		return nil
	}

	return &ifPair{
		thenText:  nodeText(thenNode, source),
		elseText:  nodeText(elseNode, source),
		thenStart: thenNode.StartByte(),
		thenEnd:   thenNode.EndByte(),
		elseStart: elseNode.StartByte(),
		elseEnd:   elseNode.EndByte(),
		line:      node.StartPosition().Row + 1,
		column:    node.StartPosition().Column,
	}
}
