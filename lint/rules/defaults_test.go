package rules_test

import (
	"testing"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

func TestDefaultRules(t *testing.T) {
	r := rules.DefaultRules()
	if len(r) == 0 {
		t.Fatal("DefaultRules returned empty slice")
	}
	// Verify each entry implements lint.Rule.
	for i, rule := range r {
		if rule.ID() == "" {
			t.Errorf("DefaultRules[%d]: empty rule ID", i)
		}
		if rule.Description() == "" {
			t.Errorf("DefaultRules[%d] (%s): empty description", i, rule.ID())
		}
	}
}

func TestNewDefaultLinter(t *testing.T) {
	linter := rules.NewDefaultLinter()
	if linter == nil {
		t.Fatal("NewDefaultLinter returned nil")
	}
	violations := linter.Lint([]byte("# Hello\n\nsome text\n"))
	// A simple valid document should produce no violations (or at least not panic).
	_ = violations
}

func TestNewDefaultLinterLints(t *testing.T) {
	linter := rules.NewDefaultLinter()
	// MD047: file should end with a single newline – this document has no trailing newline.
	src := []byte("# Hello")
	violations := linter.Lint(src)
	for _, v := range violations {
		if v.Rule == "MD047" {
			return // expected violation found
		}
	}
	t.Error("expected MD047 violation for file without trailing newline, got none")
}

func TestDefaultRulesMatchNewLinter(t *testing.T) {
	// NewDefaultLinter() and lint.NewLinter(rules.DefaultRules()...) should behave identically.
	src := []byte("# Hello\n")
	linter1 := rules.NewDefaultLinter()
	linter2 := lint.NewLinter(rules.DefaultRules()...)
	v1 := linter1.Lint(src)
	v2 := linter2.Lint(src)
	if len(v1) != len(v2) {
		t.Errorf("violation count mismatch: NewDefaultLinter=%d, NewLinter(DefaultRules())=%d", len(v1), len(v2))
	}
}
