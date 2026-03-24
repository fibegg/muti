package mutation

import (
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/fibegg/muti/internal/language"
)

func parseGo(t *testing.T, source []byte) *tree_sitter.Tree {
	t.Helper()
	lang, _ := language.Get("go")
	parser := tree_sitter.NewParser()
	t.Cleanup(func() { parser.Close() })
	if err := parser.SetLanguage(lang.TSLanguage); err != nil {
		t.Fatalf("set language: %v", err)
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		t.Fatal("parse returned nil")
	}
	t.Cleanup(func() { tree.Close() })
	return tree
}

func goLang(t *testing.T) *language.LangConfig {
	t.Helper()
	cfg, _ := language.Get("go")
	return cfg
}

func TestReplaceRange_Middle(t *testing.T) {
	src := []byte("hello world")
	got := string(replaceRange(src, 5, 6, "_beautiful_"))
	want := "hello_beautiful_world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReplaceRange_Start(t *testing.T) {
	src := []byte("hello world")
	got := string(replaceRange(src, 0, 5, "HI"))
	want := "HI world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReplaceRange_End(t *testing.T) {
	src := []byte("hello world")
	got := string(replaceRange(src, 6, 11, "EARTH"))
	want := "hello EARTH"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPickRandom_Empty(t *testing.T) {
	var items []int
	if got := pickRandom(items); got != nil {
		t.Errorf("expected nil, got %v", *got)
	}
}

func TestPickRandom_Single(t *testing.T) {
	items := []int{42}
	got := pickRandom(items)
	if got == nil || *got != 42 {
		t.Errorf("expected 42, got %v", got)
	}
}

func TestValidateSyntax_Valid(t *testing.T) {
	src := []byte("package main\nfunc main() {}\n")
	lang := goLang(t)
	if !validateSyntax(src, lang) {
		t.Error("expected valid syntax")
	}
}

func TestValidateSyntax_Invalid(t *testing.T) {
	src := []byte("package main\nfunc {{{{\n")
	lang := goLang(t)
	if validateSyntax(src, lang) {
		t.Error("expected invalid syntax")
	}
}

func TestFilterOperators_Only(t *testing.T) {
	ops, err := FilterOperators("swap_boolean", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Name() != "swap_boolean" {
		t.Errorf("expected [swap_boolean], got %v", ops)
	}
}

func TestFilterOperators_UnknownOnly(t *testing.T) {
	_, err := FilterOperators("nonexistent_op", nil)
	if err == nil {
		t.Error("expected error for unknown operator")
	}
}

func TestFilterOperators_SkipAll(t *testing.T) {
	_, err := FilterOperators("", AllNames())
	if err == nil {
		t.Error("expected error when all operators skipped")
	}
}

func TestFilterOperators_SkipSome(t *testing.T) {
	ops, err := FilterOperators("", []string{"swap_boolean", "negate_equality"})
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if op.Name() == "swap_boolean" || op.Name() == "negate_equality" {
			t.Errorf("operator %s should have been skipped", op.Name())
		}
	}
}

func TestAllNames_Count(t *testing.T) {
	names := AllNames()
	if len(names) != 15 {
		t.Errorf("expected 15 operators, got %d", len(names))
	}
}

func TestSwapBoolean(t *testing.T) {
	src := []byte("package main\nvar x = true\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	op := &SwapBoolean{}
	result, mutated, err := op.Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation result, got nil")
	}
	if result.Operator != "swap_boolean" {
		t.Errorf("operator = %q", result.Operator)
	}
	if result.Original != "true" || result.Mutated != "false" {
		t.Errorf("expected true->false, got %q->%q", result.Original, result.Mutated)
	}
	if string(mutated) == string(src) {
		t.Error("mutated source should differ from original")
	}
}

func TestSwapBoolean_NoTargets(t *testing.T) {
	src := []byte("package main\nvar x = 42\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, _ := (&SwapBoolean{}).Apply(src, tree, lang)
	if result != nil {
		t.Error("expected nil result on source with no booleans")
	}
}

func TestSwapInteger(t *testing.T) {
	src := []byte("package main\nvar x = 0\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, err := (&SwapInteger{}).Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation")
	}
	if result.Mutated != "1" {
		t.Errorf("expected 0->1, got %q->%q", result.Original, result.Mutated)
	}
}

func TestEmptyString(t *testing.T) {
	src := []byte("package main\nvar x = \"hello world\"\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, err := (&EmptyString{}).Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation")
	}
	if result.Mutated != `""` {
		t.Errorf("expected empty string, got %q", result.Mutated)
	}
}

func TestEmptyString_AlreadyEmpty(t *testing.T) {
	src := []byte("package main\nvar x = \"\"\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, _ := (&EmptyString{}).Apply(src, tree, lang)
	if result != nil {
		t.Error("expected nil result on already-empty string")
	}
}

func TestSwapArithmetic(t *testing.T) {
	src := []byte("package main\nvar x = 1 + 2\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, err := (&SwapArithmetic{}).Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation")
	}
	if result.Original != "+" || result.Mutated != "-" {
		t.Errorf("expected +->-, got %q->%q", result.Original, result.Mutated)
	}
}

func TestNullReturn_Go(t *testing.T) {
	src := []byte("package main\nfunc foo() int { return 42 }\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, err := (&NullReturn{}).Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation")
	}
	if result.Mutated != "return" {
		t.Errorf("expected 'return', got %q", result.Mutated)
	}
}

func TestSwapHashKey_Go(t *testing.T) {
	src := []byte("package main\nvar x = map[string]int{\"mykey\": 42}\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, mutated, err := (&SwapHashKey{}).Apply(src, tree, lang)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected mutation result, got nil")
	}
	if result.Operator != "swap_hash_key" {
		t.Errorf("operator = %q", result.Operator)
	}
	if result.Original == result.Mutated {
		t.Error("original and mutated should differ")
	}
	if string(mutated) == string(src) {
		t.Error("mutated source should differ from original")
	}
}

func TestSwapHashKey_NoTargets(t *testing.T) {
	src := []byte("package main\nvar x = 42\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	result, _, _ := (&SwapHashKey{}).Apply(src, tree, lang)
	if result != nil {
		t.Error("expected nil result on source with no hash keys")
	}
}

func TestAllOperators_NoTargets(t *testing.T) {
	src := []byte("package main\n")
	tree := parseGo(t, src)
	lang := goLang(t)

	for _, op := range All() {
		result, _, err := op.Apply(src, tree, lang)
		if err != nil {
			t.Errorf("%s returned error on empty source: %v", op.Name(), err)
		}
		if result != nil {
			t.Logf("%s found a target in minimal source (acceptable)", op.Name())
		}
	}
}
