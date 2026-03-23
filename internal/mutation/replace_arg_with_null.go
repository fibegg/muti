package mutation

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// ReplaceArgWithNull replaces a random method call argument with nil/None/null.
type ReplaceArgWithNull struct{}

func (o *ReplaceArgWithNull) Name() string { return "replace_arg_with_null" }

func (o *ReplaceArgWithNull) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	callTypes := []string{
		"call", "method_call", "call_expression",
		"function_call", "invocation",
	}
	calls := collectNodesByTypes(root, callTypes)

	type argInfo struct {
		callNode *tree_sitter.Node
		argNode  *tree_sitter.Node
		argIdx   int
	}
	var candidates []argInfo

	for _, call := range calls {
		args := call.ChildByFieldName("arguments")
		if args == nil {
			// Try to find argument_list child
			for i := 0; i < int(call.ChildCount()); i++ {
				child := call.Child(uint(i))
				if child != nil {
					kind := child.Kind()
					if kind == "argument_list" || kind == "arguments" || kind == "actual_parameters" {
						args = child
						break
					}
				}
			}
		}
		if args == nil {
			continue
		}

		for i := 0; i < int(args.NamedChildCount()); i++ {
			arg := args.NamedChild(uint(i))
			if arg == nil {
				continue
			}
			// Skip if already null/nil/None
			kind := arg.Kind()
			if kind == "nil" || kind == "none" || kind == "null" {
				continue
			}
			candidates = append(candidates, argInfo{callNode: call, argNode: arg, argIdx: i})
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	info := *t

	old := nodeText(info.argNode, source)
	replacement := lang.NullLiteral

	mutated := replaceRange(source, info.argNode.StartByte(), info.argNode.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	methodName := "call"
	nameNode := info.callNode.ChildByFieldName("method")
	if nameNode == nil {
		nameNode = info.callNode.ChildByFieldName("function")
	}
	if nameNode != nil {
		methodName = nodeText(nameNode, source)
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: fmt.Sprintf("Replaced arg #%d of `%s` with `%s`", info.argIdx+1, methodName, replacement),
		Line:        info.argNode.StartPosition().Row + 1,
		Column:      info.argNode.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}
