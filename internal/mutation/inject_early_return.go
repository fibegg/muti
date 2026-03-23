package mutation

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// InjectEarlyReturn injects a return null/nil/None at the start of a function.
type InjectEarlyReturn struct{}

func (o *InjectEarlyReturn) Name() string { return "inject_early_return" }

func (o *InjectEarlyReturn) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()
	funcTypes := []string{
		"method", "function_definition", "function_declaration",
		"method_definition", "arrow_function", "lambda",
	}
	targets := collectNodesByTypes(root, funcTypes)

	t := pickRandom(targets)
	if t == nil {
		return nil, nil, nil
	}
	node := *t

	// Find the body of the function
	body := node.ChildByFieldName("body")
	if body == nil {
		// Some languages have different field names
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(uint(i))
			if child != nil {
				kind := child.Kind()
				if kind == "block" || kind == "statement_block" || kind == "body" || kind == "suite" {
					body = child
					break
				}
			}
		}
	}
	if body == nil {
		return nil, nil, nil
	}

	// Determine the return statement to inject
	var returnStmt string
	switch lang.Name {
	case "ruby":
		returnStmt = "return nil\n"
	case "python":
		returnStmt = "return None\n"
	case "go":
		returnStmt = "return\n"
	default:
		returnStmt = "return null;\n"
	}

	// Find the indentation of the first statement in the body
	indent := ""
	if body.ChildCount() > 0 {
		firstChild := body.Child(0)
		if firstChild != nil {
			lineStart := firstChild.StartByte()
			// Walk backward to find the start of the line
			for lineStart > 0 && source[lineStart-1] != '\n' {
				lineStart--
			}
			indent = string(source[lineStart:firstChild.StartByte()])
		}
	} else {
		indent = "  "
	}

	// Inject after the opening of the body
	insertPoint := body.StartByte()
	// Skip past the opening brace/colon/newline
	bodyText := nodeText(body, source)
	if strings.HasPrefix(bodyText, "{") || strings.HasPrefix(bodyText, ":") {
		insertPoint++
		returnStmt = "\n" + indent + returnStmt
	} else {
		returnStmt = indent + returnStmt
	}

	mutated := replaceRange(source, insertPoint, insertPoint, returnStmt)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	funcName := "anonymous"
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		funcName = nodeText(nameNode, source)
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: fmt.Sprintf("Injected early return at start of `%s`", funcName),
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    "",
		Mutated:     strings.TrimSpace(returnStmt),
	}, mutated, nil
}
