package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// RemoveMethodCall replaces obj.method(args) with just obj,
// testing whether method side-effects and return values matter.
type RemoveMethodCall struct{}

func (o *RemoveMethodCall) Name() string { return "remove_method_call" }

func (o *RemoveMethodCall) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Chained method call / member access node types
	callTypes := []string{
		"call",            // Ruby method call
		"method_call",     // Ruby
		"call_expression", // JS/TS/Go
	}

	targets := collectNodesByTypes(root, callTypes)

	// Filter: only method calls on a receiver (a.b(...) style)
	var candidates []methodCallInfo
	for _, n := range targets {
		info := extractMethodReceiver(n, source)
		if info != nil {
			candidates = append(candidates, *info)
		}
	}

	t := pickRandom(candidates)
	if t == nil {
		return nil, nil, nil
	}
	info := *t

	// Replace the entire call expression with just the receiver
	mutated := replaceRange(source, info.callStart, info.callEnd, info.receiverText)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Removed method call, kept receiver `" + info.receiverText + "`",
		Line:        info.line,
		Column:      info.column,
		Original:    info.fullText,
		Mutated:     info.receiverText,
	}, mutated, nil
}

type methodCallInfo struct {
	receiverText string
	fullText     string
	callStart    uint
	callEnd      uint
	line         uint
	column       uint
}

func extractMethodReceiver(node *tree_sitter.Node, source []byte) *methodCallInfo {
	// Look for patterns like: receiver.method(args)
	// The first child should be a member_expression/selector containing the receiver
	if node.ChildCount() < 2 {
		return nil
	}

	first := node.Child(0)
	if first == nil {
		return nil
	}

	kind := first.Kind()
	// For JS/TS call_expression: first child is member_expression
	if kind == "member_expression" || kind == "selector_expression" {
		// The receiver is the first child of the member_expression
		receiver := first.Child(0)
		if receiver == nil {
			return nil
		}
		receiverText := nodeText(receiver, source)
		if len(receiverText) == 0 {
			return nil
		}
		return &methodCallInfo{
			receiverText: receiverText,
			fullText:     nodeText(node, source),
			callStart:    node.StartByte(),
			callEnd:      node.EndByte(),
			line:         node.StartPosition().Row + 1,
			column:       node.StartPosition().Column,
		}
	}

	// For Ruby: call node where first child is the receiver, then ".", then method name
	if node.Kind() == "call" || node.Kind() == "method_call" {
		receiverText := nodeText(first, source)
		if len(receiverText) == 0 || first.Kind() == "identifier" {
			// Simple function call, not a method call — skip
			return nil
		}
		// Check that there's a "." separator
		if node.ChildCount() >= 3 {
			dot := node.Child(1)
			if dot != nil && nodeText(dot, source) == "." {
				return &methodCallInfo{
					receiverText: receiverText,
					fullText:     nodeText(node, source),
					callStart:    node.StartByte(),
					callEnd:      node.EndByte(),
					line:         node.StartPosition().Row + 1,
					column:       node.StartPosition().Column,
				}
			}
		}
	}

	return nil
}
