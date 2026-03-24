package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// HardcodeTrue replaces conditional expressions with true/True,
// testing that conditions actually matter and aren't always-true.
type HardcodeTrue struct{}

func (o *HardcodeTrue) Name() string { return "hardcode_true" }

func (o *HardcodeTrue) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Find if-statement nodes and target their condition expression
	ifTypes := []string{
		"if_statement",  // Python, JS/TS, Go
		"if",            // Ruby
		"if_expression", // some grammars
	}

	targets := collectNodesByTypes(root, ifTypes)

	var candidates []*tree_sitter.Node
	for _, n := range targets {
		cond := extractCondition(n, source)
		if cond != nil {
			condText := nodeText(cond, source)
			// Don't replace if already true/True
			if condText != "true" && condText != "True" {
				candidates = append(candidates, cond)
			}
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	node := *t
	old := nodeText(node, source)

	// Use language-appropriate true literal
	replacement := "true"
	if lang.Name == "python" {
		replacement = "True"
	}

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	preview := old
	if len(preview) > 35 {
		preview = preview[:32] + "..."
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Hardcoded condition `" + preview + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}

func extractCondition(node *tree_sitter.Node, source []byte) *tree_sitter.Node {
	kind := node.Kind()

	switch kind {
	case "if_statement":
		// Python/JS/TS: condition is typically the second child (after "if" keyword)
		// Go: condition is after "if" keyword
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			ck := child.Kind()
			// Skip keywords and blocks
			if ck == "if" || ck == "else" || ck == "block" || ck == "statement_block" ||
				ck == "else_clause" || ck == "elif_clause" || ck == "consequence" ||
				ck == "{" || ck == "}" || ck == ":" || ck == "then" {
				continue
			}
			// Skip parenthesized_expression wrapper — get the inner node
			if ck == "parenthesized_expression" && child.ChildCount() >= 3 {
				inner := child.Child(1)
				if inner != nil {
					return inner
				}
			}
			// The first non-keyword child is the condition
			return child
		}

	case "if", "if_expression":
		// Ruby: condition is typically the first expression child
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "if" || ck == "then" || ck == "else" || ck == "end" ||
				ck == "elsif" || ck == "{" || ck == "}" {
				continue
			}
			return child
		}
	}

	return nil
}
