package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/mrueg/goldmark-lint/lint"
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
- --fix            updates files to resolve fixable issues
- --no-cache       disable reading/writing the .markdownlint-cli2-cache file
- --output-format  output format: default, json, junit, tap (default: default)
- --help           writes this message to the console and exits without doing anything else
- --version        prints the version and exits

Config file:
- Reads .markdownlint-cli2.yaml (or .yml, .jsonc, .json) from the current
  directory or any parent directory (same discovery as markdownlint-cli2).
- Supports "config" (rule enable/disable and options), "ignores",
  "overrides" (per-glob rule config overrides), "extends" (inherit
  configuration from another config file), and "outputFormatters" keys.

Exit codes:
- 0: Linting was successful and there were no errors
- 1: Linting was successful and there were errors
- 2: Linting was not successful due to a problem or failure
`

func main() {
	fix := flag.Bool("fix", false, "updates files to resolve fixable issues")
	help := flag.Bool("help", false, "writes help message and exits")
	ver := flag.Bool("version", false, "prints the version and exits")
	noCache := flag.Bool("no-cache", false, "disable reading/writing the cache file")
	outputFormat := flag.String("output-format", "", "output format: default, json, junit, tap")
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

	// Validate --output-format flag if specified.
	if *outputFormat != "" {
		switch *outputFormat {
		case "default", "json", "junit", "tap":
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown output format %q; supported formats: default, json, junit, tap\n", *outputFormat)
			os.Exit(2)
		}
	}

	// Auto-discover config file starting from the current working directory.
	var cfg *ConfigFile
	cwd, _ := os.Getwd()
	if cwd != "" {
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

	// Determine the formatter specs to use.
	// CLI flag takes priority; then config outputFormatters; then default.
	var formatterSpecs []outputFormatterSpec
	if *outputFormat != "" {
		formatterSpecs = []outputFormatterSpec{{format: *outputFormat}}
	} else if cfg != nil && len(cfg.OutputFormatters) > 0 {
		formatterSpecs = parseOutputFormatters(cfg.OutputFormatters)
	}
	if len(formatterSpecs) == 0 {
		formatterSpecs = []outputFormatterSpec{{format: "default"}}
	}

	// Default linter (used when no overrides are defined or override doesn't match).
	linter := newLinterFromConfig(ruleCfg)

	// Load cache (skip when --no-cache or --fix is used).
	useCache := !*noCache && !*fix
	cache := make(lintCache)
	if useCache && cwd != "" {
		cache = loadCache(cwd)
	}

	exitCode := 0

	// allViolations collects violations from all sources for the final formatter run.
	var allViolations []fileViolation

	// Handle stdin ("-") sequentially – stdin cannot be parallelised.
	for _, pattern := range flag.Args() {
		if pattern != "-" {
			continue
		}
		source, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			exitCode = 2
			continue
		}
		violations := linter.Lint(source)
		allViolations = append(allViolations, fileViolation{File: "stdin", Violations: violations})
	}

	// Collect all non-stdin files in order so that output remains deterministic.
	var allFiles []string
	for _, pattern := range flag.Args() {
		if pattern == "-" {
			continue
		}
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, file := range files {
			if !isIgnored(file, ignores) {
				allFiles = append(allFiles, file)
			}
		}
	}

	// fileResult carries the outcome of processing a single file.
	type fileResult struct {
		violations []lint.Violation
		err        error
		errCode    int
	}

	results := make([]fileResult, len(allFiles))
	newEntries := make(lintCache) // updated cache entries collected from goroutines
	var mu sync.Mutex             // protects newEntries

	var wg sync.WaitGroup
	for i, file := range allFiles {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()

			source, err := os.ReadFile(file)
			if err != nil {
				results[i] = fileResult{err: err, errCode: 2}
				return
			}

			hash := hashContent(source)

			// Cache hit: file unchanged, replay cached violations.
			if useCache {
				if entry, ok := cache[file]; ok && entry.Hash == hash {
					results[i] = fileResult{violations: entry.Violations}
					return
				}
			}

			// Determine the effective linter for this file.
			fileLinter := linter
			if len(overrides) > 0 {
				fileCfg := effectiveConfigForFile(ruleCfg, overrides, file)
				fileLinter = newLinterFromConfig(fileCfg)
			}

			// Apply fixes if requested.
			if *fix {
				fixed := fileLinter.Fix(source)
				if err := os.WriteFile(file, fixed, 0644); err != nil {
					results[i] = fileResult{err: err, errCode: 2}
					return
				}
				source = fixed
				hash = hashContent(source)
			}

			violations := fileLinter.Lint(source)
			results[i] = fileResult{violations: violations}

			// Store the new cache entry.
			if useCache {
				mu.Lock()
				newEntries[file] = cacheEntry{Hash: hash, Violations: violations}
				mu.Unlock()
			}
		}(i, file)
	}
	wg.Wait()

	// Collect file results in original order for deterministic output.
	for i, file := range allFiles {
		r := results[i]
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, r.err)
			if r.errCode > exitCode {
				exitCode = r.errCode
			}
			continue
		}
		allViolations = append(allViolations, fileViolation{File: file, Violations: r.violations})
	}

	// Apply per-violation severity so formatters can use it.
	for i := range allViolations {
		for j := range allViolations[i].Violations {
			allViolations[i].Violations[j].Severity = getRuleSeverity(allViolations[i].Violations[j].Rule, ruleCfg)
		}
	}

	// Calculate violation-based exit code (only upgrade, never downgrade from 2).
	if exitCode < 1 {
		for _, fv := range allViolations {
			for _, v := range fv.Violations {
				if v.Severity != "warning" {
					exitCode = 1
					break
				}
			}
			if exitCode == 1 {
				break
			}
		}
	}

	// Run each configured formatter.
	for _, spec := range formatterSpecs {
		var w io.Writer
		var closeFile func()
		if spec.outfile != "" {
			f, err := os.Create(spec.outfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", spec.outfile, err)
				if exitCode < 2 {
					exitCode = 2
				}
				continue
			}
			w = f
			closeFile = func() {
				if err := f.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not close output file %s: %v\n", spec.outfile, err)
				}
			}
		} else if spec.format == "default" {
			w = os.Stderr
			closeFile = func() {}
		} else {
			w = os.Stdout
			closeFile = func() {}
		}

		switch spec.format {
		case "json":
			formatJSON(allViolations, w)
		case "junit":
			formatJUnit(allViolations, w)
		case "tap":
			formatTAP(allViolations, w)
		default:
			formatDefault(allViolations, w)
		}
		closeFile()
	}

	// Persist updated cache entries.
	if useCache && cwd != "" && len(newEntries) > 0 {
		for k, v := range newEntries {
			cache[k] = v
		}
		if err := saveCache(cwd, cache); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save cache: %v\n", err)
		}
	}

	os.Exit(exitCode)
}
