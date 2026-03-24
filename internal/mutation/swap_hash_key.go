package mutation

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

// SwapHashKey mutates hash/object/dict keys by renaming them.
// This tests whether code is sensitive to specific key names.
// Works across Ruby hashes, JS/TS objects, Python dicts, and Go map literals.
type SwapHashKey struct{}

func (o *SwapHashKey) Name() string { return "swap_hash_key" }

func (o *SwapHashKey) Apply(source []byte, tree *tree_sitter.Tree, lang *language.LangConfig) (*MutationResult, []byte, error) {
	root := tree.RootNode()

	// Tree-sitter node types that represent key-value pairs across languages:
	//   Ruby:       pair (key: value)
	//   Python:     pair (key: value)
	//   JavaScript: pair (key: value)
	//   TypeScript: pair (key: value)
	//   Go:         keyed_element (key: value)
	pairTypes := map[string]bool{
		"pair":          true,
		"keyed_element": true,
	}

	var candidates []*tree_sitter.Node
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if pairTypes[n.Kind()] {
			// The first child is the key
			key := n.Child(0)
			if key != nil {
				keyText := nodeText(key, source)
				// Only mutate keys that are meaningful (not empty/too short)
				if len(keyText) > 0 {
					candidates = append(candidates, key)
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
	node := *t
	old := nodeText(node, source)

	// Build a replacement key that preserves the syntactic form.
	replacement := mutateKey(old, node.Kind())

	mutated := replaceRange(source, node.StartByte(), node.EndByte(), replacement)
	if !validateSyntax(mutated, lang) {
		return nil, nil, nil
	}

	return &MutationResult{
		Operator:    o.Name(),
		Description: "Renamed key `" + old + "` → `" + replacement + "`",
		Line:        node.StartPosition().Row + 1,
		Column:      node.StartPosition().Column,
		Original:    old,
		Mutated:     replacement,
	}, mutated, nil
}

// mutateKey produces a renamed version of a key, preserving its syntactic form.
func mutateKey(key string, nodeKind string) string {
	switch {
	// Ruby symbol keys:  :foo  or  foo:  (hash_key_symbol / simple_symbol)
	case nodeKind == "simple_symbol" && len(key) > 1 && key[0] == ':':
		return ":_muted_key"
	case nodeKind == "hash_key_symbol":
		return "_muted_key"

	// Quoted string keys: "foo", 'foo'
	case len(key) >= 2 && (key[0] == '"' || key[0] == '\''):
		quote := key[0]
		return string(quote) + "_muted_key" + string(quote)

	// Bare identifier keys (JS/TS shorthand, Go)
	case nodeKind == "property_identifier" ||
		nodeKind == "identifier" ||
		nodeKind == "shorthand_property_identifier":
		return "_muted_key"

	// Go composite literal keys — could be identifiers or expressions
	case nodeKind == "literal_element":
		if len(key) >= 2 && key[0] == '"' {
			return `"_muted_key"`
		}
		return "_muted_key"

	// Fallback: if it looks like a string, wrap in same quotes; else bare rename
	default:
		if len(key) >= 2 && key[0] == '"' {
			return `"_muted_key"`
		}
		if len(key) >= 2 && key[0] == '\'' {
			return `'_muted_key'`
		}
		return "_muted_key"
	}
}
