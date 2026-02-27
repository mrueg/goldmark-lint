package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: goldmark-lint <file> [file...]")
		os.Exit(1)
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

	exitCode := 0
	for _, pattern := range os.Args[1:] {
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
				exitCode = 1
				continue
			}
			violations := linter.Lint(source)
			for _, v := range violations {
				fmt.Printf("%s:%d:%d %s %s\n", file, v.Line, v.Column, v.Rule, v.Message)
				exitCode = 1
			}
		}
	}
	os.Exit(exitCode)
}
