package mutation

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveGuardClause removes early-return guard clauses from function bodies.
type RemoveGuardClause struct{}

func (o *RemoveGuardClause) Name() string { return "remove_guard_clause" }

func (o *RemoveGuardClause) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	funcTypes := []string{
		"method", "function_definition", "function_declaration",
		"method_definition",
	}
	funcs := collectNodesByTypes(root, funcTypes)

	type guard struct {
		funcNode *tree_sitter.Node
		ifNode   *tree_sitter.Node
	}
	var candidates []guard

	for _, fn := range funcs {
		body := fn.ChildByFieldName("body")
		if body == nil {
			for i := 0; i < int(fn.ChildCount()); i++ {
				child := fn.Child(uint(i))
				if child != nil {
					kind := child.Kind()
					if kind == "block" || kind == "statement_block" || kind == "body" || kind == "suite" {
						body = child
						break
					}
				}
			}
		}
		if body == nil || body.NamedChildCount() < 2 {
			continue
		}

		// Look for if statements that contain only a return/raise/throw
		for i := 0; i < int(body.NamedChildCount()); i++ {
			stmt := body.NamedChild(uint(i))
			if stmt == nil {
				continue
			}
			kind := stmt.Kind()
			if kind != "if" && kind != "if_statement" && kind != "if_expression" {
				continue
			}

			// Check if the consequence is a bare return/raise/throw
			consequence := stmt.ChildByFieldName("consequence")
			if consequence == nil {
				continue
			}
			alternative := stmt.ChildByFieldName("alternative")
			if alternative != nil {
				// Not a guard clause if it has an else
				continue
			}

			if isGuardBody(consequence) {
				candidates = append(candidates, guard{funcNode: fn, ifNode: stmt})
			}
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	g := *t

	// Remove the guard clause
	start := g.ifNode.StartByte()
	end := g.ifNode.EndByte()
	if int(end) < len(source) && source[end] == '\n' {
		end++
	}

	old := nodeText(g.ifNode, source)
	mutated := replaceRange(source, start, end, "")
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	funcName := "anonymous"
	nameNode := g.funcNode.ChildByFieldName("name")
	if nameNode != nil {
		funcName = nodeText(nameNode, source)
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: fmt.Sprintf("Removed guard clause from `%s`", funcName),
		Line:        g.ifNode.StartPosition().Row + 1,
		Column:      g.ifNode.StartPosition().Column,
		Original:    old,
		Mutated:     "",
	}, mutated, nil
}

// isGuardBody checks if a node contains only return/raise/throw statements.
func isGuardBody(node *tree_sitter.Node) bool {
	kind := node.Kind()
	guardTypes := map[string]bool{
		"return": true, "return_statement": true,
		"raise": true, "throw_statement": true,
	}
	if guardTypes[kind] {
		return true
	}
	// Check children for single-statement bodies
	if node.NamedChildCount() == 1 {
		child := node.NamedChild(0)
		if child != nil {
			return guardTypes[child.Kind()]
		}
	}
	return false
}
