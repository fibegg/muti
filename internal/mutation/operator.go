package mutation

import (
	"fmt"
	"math/rand"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// MutationResult holds metadata about an applied mutation.
type MutationResult struct {
	File        string `json:"file"`
	Operator    string `json:"operator"`
	Description string `json:"description"`
	Line        uint   `json:"line"`
	Column      uint   `json:"column"`
	Original    string `json:"original"`
	Mutated     string `json:"mutated"`
	Diff        string `json:"diff,omitempty"`
	ProbeResult string `json:"probe_result,omitempty"`
	ProbeExit   int    `json:"probe_exit_code,omitempty"`
}

// Operator defines a mutation operator.
type Operator interface {
	// Name returns the operator's identifier.
	Name() string
	// Apply attempts to apply a mutation to the given source.
	// Returns mutated source and result, or nil if no targets found.
	Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error)
}

// All returns all registered operators.
func All() []Operator {
	return []Operator{
		&SwapBoolean{},
		&NegateEquality{},
		&SwapLogical{},
		&FlipConditional{},
		&InjectEarlyReturn{},
		&RemoveStatement{},
		&SwapInteger{},
		&EmptyString{},
		&NullReturn{},
		&SwapComparison{},
		&RemoveGuardClause{},
		&RemoveErrorHandler{},
		&ReplaceArgWithNull{},
		&SwapArithmetic{},
		&SwapHashKey{},
	}
}

// AllNames returns names of all operators.
func AllNames() []string {
	ops := All()
	names := make([]string, len(ops))
	for i, op := range ops {
		names[i] = op.Name()
	}
	return names
}

// ByName returns an operator by name, or nil.
func ByName(name string) Operator {
	for _, op := range All() {
		if op.Name() == name {
			return op
		}
	}
	return nil
}

// FilterOperators returns operators minus any in skip, or only the specified one.
func FilterOperators(only string, skip []string) ([]Operator, error) {
	if only != "" {
		op := ByName(only)
		if op == nil {
			return nil, fmt.Errorf("unknown operator: %s", only)
		}
		return []Operator{op}, nil
	}

	skipSet := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipSet[s] = true
	}

	all := All()
	result := make([]Operator, 0, len(all))
	for _, op := range all {
		if !skipSet[op.Name()] {
			result = append(result, op)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("all operators have been skipped")
	}
	return result, nil
}

// collectNodesByTypes collects all nodes matching any of the given types.
func collectNodesByTypes(node *tree_sitter.Node, types []string) []*tree_sitter.Node {
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}
	var result []*tree_sitter.Node
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if typeSet[n.Kind()] {
			result = append(result, n)
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(uint(i))
			if child != nil {
				walk(child)
			}
		}
	}
	walk(node)
	return result
}

// nodeText extracts the source text for a node.
func nodeText(node *tree_sitter.Node, source []byte) string {
	start := node.StartByte()
	end := node.EndByte()
	if int(end) > len(source) {
		end = uint(len(source))
	}
	return string(source[start:end])
}

// replaceRange replaces bytes from start to end with replacement.
func replaceRange(source []byte, start, end uint, replacement string) []byte {
	result := make([]byte, 0, len(source)-int(end-start)+len(replacement))
	result = append(result, source[:start]...)
	result = append(result, []byte(replacement)...)
	result = append(result, source[end:]...)
	return result
}

// pickRandom picks a random element or returns nil if empty.
func pickRandom[T any](items []T) *T {
	if len(items) == 0 {
		return nil
	}
	return &items[rand.Intn(len(items))]
}

// validateSyntax re-parses and checks for ERROR nodes.
func validateSyntax(mutated []byte, lang *language.LangConfig) bool {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang.TSLanguage); err != nil {
		return false
	}
	tree := parser.Parse(mutated, nil)
	if tree == nil {
		return false
	}
	defer tree.Close()
	return !tree.RootNode().HasError()
}
