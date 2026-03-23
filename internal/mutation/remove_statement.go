package mutation

import (
	"fmt"
	"math/rand"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveStatement removes a random statement from a function body.
type RemoveStatement struct{}

func (o *RemoveStatement) Name() string { return "remove_statement" }

func (o *RemoveStatement) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	funcTypes := []string{
		"method", "function_definition", "function_declaration",
		"method_definition", "arrow_function",
	}
	funcs := collectNodesByTypes(root, funcTypes)

	// Filter to functions with multi-statement bodies
	type bodyInfo struct {
		funcNode *tree_sitter.Node
		body     *tree_sitter.Node
	}
	var candidates []bodyInfo
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
		if body != nil && body.NamedChildCount() > 1 {
			candidates = append(candidates, bodyInfo{funcNode: fn, body: body})
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	info := *t

	// Pick a random statement from the body
	stmtCount := int(info.body.NamedChildCount())
	idx := rand.Intn(stmtCount)
	stmt := info.body.NamedChild(uint(idx))
	if stmt == nil {
		return nil, nil, nil
	}

	// Remove the statement (and trailing whitespace/newline)
	start := stmt.StartByte()
	end := stmt.EndByte()
	// Include the trailing newline if present
	if int(end) < len(source) && source[end] == '\n' {
		end++
	}

	mutated := replaceRange(source, start, end, "")
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	funcName := "anonymous"
	nameNode := info.funcNode.ChildByFieldName("name")
	if nameNode != nil {
		funcName = nodeText(nameNode, source)
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: fmt.Sprintf("Removed statement #%d from `%s`", idx+1, funcName),
		Line:        stmt.StartPosition().Row + 1,
		Column:      stmt.StartPosition().Column,
		Original:    nodeText(stmt, source),
		Mutated:     "",
	}, mutated, nil
}
