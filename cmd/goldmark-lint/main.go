package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// version is set at build time via -ldflags.
var version = "dev"

const helpText = `goldmark-lint
https://github.com/mrueg/goldmark-lint

Syntax: goldmark-lint glob0 [glob1] [...] [globN] [--fix] [--help] [--version]
        goldmark-lint - (read from stdin)

Glob expressions:
- * matches any number of characters, but not /
- ? matches a single character, but not /
- ** matches any number of characters, including /

Optional parameters:
- --fix      updates files to resolve fixable issues
- --help     writes this message to the console and exits without doing anything else
- --version  prints the version and exits

Config file:
- Reads .markdownlint-cli2.yaml (or .yml, .jsonc, .json) from the current
  directory or any parent directory (same discovery as markdownlint-cli2).
- Supports "config" (rule enable/disable and options), "ignores", and
  "overrides" (per-glob rule config overrides) keys.

Exit codes:
- 0: Linting was successful and there were no errors
- 1: Linting was successful and there were errors
- 2: Linting was not successful due to a problem or failure
`

func main() {
	fix := flag.Bool("fix", false, "updates files to resolve fixable issues")
	help := flag.Bool("help", false, "writes help message and exits")
	ver := flag.Bool("version", false, "prints the version and exits")
	flag.Parse()

	if *help {
		fmt.Print(helpText)
		os.Exit(0)
	}

	if *ver {
		fmt.Println(version)
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
	var overrides []GlobOverride
	if cfg != nil {
		ruleCfg = cfg.Config
		ignores = cfg.Ignores
		overrides = cfg.Overrides
	}

	// Default linter (used when no overrides are defined or override doesn't match).
	linter := newLinterFromConfig(ruleCfg)

	exitCode := 0
	for _, pattern := range flag.Args() {
		// Special case: "-" means read from stdin
		if pattern == "-" {
			source, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				exitCode = 2
				continue
			}
			violations := linter.Lint(source)
			for _, v := range violations {
				v.Severity = getRuleSeverity(v.Rule, ruleCfg)
				fmt.Fprintf(os.Stderr, "stdin:%d:%d %s %s\n", v.Line, v.Column, v.Rule, v.Message)
				if v.Severity != "warning" && exitCode < 1 {
					exitCode = 1
				}
			}
			continue
		}

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
			fileLinter := linter
			if len(overrides) > 0 {
				fileCfg := effectiveConfigForFile(ruleCfg, overrides, file)
				fileLinter = newLinterFromConfig(fileCfg)
			}
			if *fix {
				fixed := fileLinter.Fix(source)
				if err := os.WriteFile(file, fixed, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
					exitCode = 2
					continue
				}
				source = fixed
			}
			violations := fileLinter.Lint(source)
			for _, v := range violations {
				v.Severity = getRuleSeverity(v.Rule, ruleCfg)
				fmt.Fprintf(os.Stderr, "%s:%d:%d %s %s\n", file, v.Line, v.Column, v.Rule, v.Message)
				if v.Severity != "warning" && exitCode < 1 {
					exitCode = 1
				}
			}
		}
	}
	os.Exit(exitCode)
}
