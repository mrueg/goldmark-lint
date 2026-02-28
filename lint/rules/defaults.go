package rules

import "github.com/mrueg/goldmark-lint/lint"

// DefaultRules returns a slice of all lint rules with their default settings.
// This is the standard set of rules used by the goldmark-lint CLI tool.
// It is intended for use by library consumers who want to create a [lint.Linter]
// with all rules enabled:
//
//	linter := lint.NewLinter(rules.DefaultRules()...)
func DefaultRules() []lint.Rule {
	return []lint.Rule{
		MD001{},
		MD003{},
		MD004{},
		MD005{},
		MD007{},
		MD009{},
		MD010{},
		MD011{},
		MD012{},
		MD013{},
		MD014{},
		MD018{},
		MD019{},
		MD020{},
		MD021{},
		MD022{},
		MD023{},
		MD024{},
		MD025{},
		MD026{},
		MD027{},
		MD028{},
		MD029{},
		MD030{},
		MD031{},
		MD032{},
		MD033{},
		MD034{},
		MD035{},
		MD036{},
		MD037{},
		MD038{},
		MD039{},
		MD040{},
		MD041{},
		MD042{},
		MD043{},
		MD044{},
		MD045{},
		MD046{},
		MD047{},
		MD048{},
		MD049{},
		MD050{},
		MD051{},
		MD052{},
		MD053{},
		MD054{},
		MD055{},
		MD056{},
		MD058{},
		MD059{},
		MD060{},
	}
}

// NewDefaultLinter creates a [lint.Linter] with all rules enabled at their
// default settings. It is equivalent to:
//
//	lint.NewLinter(rules.DefaultRules()...)
func NewDefaultLinter() *lint.Linter {
	return lint.NewLinter(DefaultRules()...)
}
