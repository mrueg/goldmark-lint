package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

Config file:
- Reads .markdownlint-cli2.yaml (or .yml, .jsonc, .json) from the current
  directory or any parent directory (same discovery as markdownlint-cli2).
- Supports "config" (rule enable/disable and options) and "ignores" keys.

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

	// Auto-discover config file starting from the current working directory.
	var cfg *ConfigFile
	if cwd, err := os.Getwd(); err == nil {
		if cfgPath := findConfigFile(cwd); cfgPath != "" {
			loaded, err := loadConfig(cfgPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", cfgPath, err)
				os.Exit(2)
			}
			cfg = loaded
		}
	}

	var ruleCfg map[string]interface{}
	var ignores []string
	if cfg != nil {
		ruleCfg = cfg.Config
		ignores = cfg.Ignores
	}

	linter := newLinterFromConfig(ruleCfg)

	exitCode := 0
	for _, pattern := range flag.Args() {
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, file := range files {
			if isIgnored(file, ignores) {
				continue
			}
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
