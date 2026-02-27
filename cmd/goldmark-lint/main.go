package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

Optional parameters:
- --fix   updates files to resolve fixable issues
- --help  writes this message to the console and exits without doing anything else

Exit codes:
- 0: Linting was successful and there were no errors
- 1: Linting was successful and there were errors
- 2: Linting was not successful due to a problem or failure
`

func main() {
	fix := flag.Bool("fix", false, "updates files to resolve fixable issues")
	help := flag.Bool("help", false, "writes help message and exits")
	flag.Parse()

	if *help {
		fmt.Print(helpText)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Fprint(os.Stderr, helpText)
		os.Exit(2)
	}

	linter := lint.NewLinter(
		rules.MD001{},
		rules.MD003{},
		rules.MD004{},
		rules.MD007{},
		rules.MD009{},
		rules.MD010{},
		rules.MD012{},
		rules.MD013{},
		rules.MD022{},
		rules.MD024{},
		rules.MD025{},
		rules.MD029{},
		rules.MD033{},
		rules.MD034{},
		rules.MD041{},
		rules.MD047{},
	)

	exitCode := 0
	for _, pattern := range flag.Args() {
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
				exitCode = 2
				continue
			}
			if *fix {
				fixed := linter.Fix(source)
				if err := os.WriteFile(file, fixed, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
					exitCode = 2
					continue
				}
				source = fixed
			}
			violations := linter.Lint(source)
			for _, v := range violations {
				fmt.Fprintf(os.Stderr, "%s:%d:%d %s %s\n", file, v.Line, v.Column, v.Rule, v.Message)
				if exitCode < 1 {
					exitCode = 1
				}
			}
		}
	}
	os.Exit(exitCode)
}
