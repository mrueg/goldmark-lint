package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

const helpText = `goldmark-lint
https://github.com/mrueg/goldmark-lint

Syntax: goldmark-lint glob0 [glob1] [...] [globN] [--fix] [--help]

Glob expressions:
- * matches any number of characters, but not /
- ? matches a single character, but not /
- ** matches any number of characters, including /
- {} allows for a comma-separated list of "or" expressions
- ! at the beginning of a pattern negates the match

Optional parameters:
- --fix   updates files to resolve fixable issues (not yet implemented)
- --help  writes this message to stderr and exits without doing anything else

Exit codes:
- 0: Linting was successful and there were no errors
- 1: Linting was successful and there were errors
- 2: Linting was not successful due to a problem or failure
`

func main() {
	fix := flag.Bool("fix", false, "updates files to resolve fixable issues (not yet implemented)")
	help := flag.Bool("help", false, "writes help message to stderr and exits")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, helpText)
	}
	flag.Parse()

	if *help {
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(0)
	}

	_ = fix // no rules support auto-fix yet

	if flag.NArg() < 1 {
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(2)
	}

	linter := lint.NewLinter(
		rules.MD001{},
		rules.MD009{},
		rules.MD010{},
		rules.MD012{},
		rules.MD013{},
		rules.MD022{},
		rules.MD025{},
		rules.MD041{},
		rules.MD047{},
	)

	aliasMap := make(map[string]string)
	for _, r := range linter.Rules {
		aliasMap[r.ID()] = r.Alias()
	}

	exitCode := 0
	for _, pattern := range flag.Args() {
		files, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid glob pattern %q: %v\n", pattern, err)
			exitCode = 2
			continue
		}
		if len(files) == 0 {
			// treat as literal file path
			files = []string{pattern}
		}
		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				exitCode = 2
				continue
			}
			violations := linter.Lint(source)
			for _, v := range violations {
				fmt.Fprintf(os.Stderr, "%s:%d:%d %s/%s %s\n", file, v.Line, v.Column, v.Rule, aliasMap[v.Rule], v.Message)
				if exitCode < 1 {
					exitCode = 1
				}
			}
		}
	}
	os.Exit(exitCode)
}
