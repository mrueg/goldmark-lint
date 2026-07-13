package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/urfave/cli/v3"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	app := &cli.Command{
		Name:                  "goldmark-lint",
		Usage:                 "Lint Markdown files",
		Version:               version,
		EnableShellCompletion: true,
		ArgsUsage:             "glob0 [glob1] [...] [globN]",
		Description: `Lint Markdown files matching the given glob patterns.

Use '-' as an argument to read from stdin.
Use --format to read stdin, apply fixes, and write to stdout.

Glob expressions:
  * matches any number of characters, but not /
  ? matches a single character, but not /
  ** matches any number of characters, including /

Config file:
  Reads .markdownlint-cli2.yaml (or .yml, .jsonc, .json) from the current
  directory or any parent directory (same discovery as markdownlint-cli2).
  Also reads .markdownlint.yaml (or .yml, .jsonc, .json), which uses the
  simpler rule-only format (compatible with vscode-markdownlint).
  .markdownlint-cli2.* files take priority when both are present.

Exit codes:
  0  Linting was successful and there were no errors
  1  Linting was successful and there were errors
  2  Linting was not successful due to a problem or failure`,
		// Suppress printing of empty error messages produced by cli.Exit("", code).
		ExitErrHandler: func(_ context.Context, _ *cli.Command, err error) {
			if err == nil {
				return
			}
			if msg := err.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			if exitCoder, ok := err.(cli.ExitCoder); ok {
				os.Exit(exitCoder.ExitCode())
			}
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to config file (overrides auto-discovery)",
				Sources: cli.EnvVars("GOLDMARK_LINT_CONFIG"),
			},
			&cli.BoolFlag{
				Name:    "fail-on-warning",
				Usage:   "exit with code 1 even when all violations are warnings",
				Sources: cli.EnvVars("GOLDMARK_LINT_FAIL_ON_WARNING"),
			},
			&cli.BoolFlag{
				Name:    "format",
				Usage:   "read stdin, apply fixes, write stdout",
				Sources: cli.EnvVars("GOLDMARK_LINT_FORMAT"),
			},
			&cli.BoolFlag{
				Name:    "list-rules",
				Usage:   "print a table of all rules with their aliases, enabled/disabled state, and options",
				Sources: cli.EnvVars("GOLDMARK_LINT_LIST_RULES"),
			},
			&cli.BoolFlag{
				Name:    "no-cache",
				Usage:   "disable reading/writing the cache file",
				Sources: cli.EnvVars("GOLDMARK_LINT_NO_CACHE"),
			},
			&cli.BoolFlag{
				Name:    "no-globs",
				Usage:   "ignore the globs config key at runtime",
				Sources: cli.EnvVars("GOLDMARK_LINT_NO_GLOBS"),
			},
			&cli.StringFlag{
				Name:    "output-format",
				Usage:   "output format: default, json, junit, tap, sarif, github",
				Sources: cli.EnvVars("GOLDMARK_LINT_OUTPUT_FORMAT"),
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "suppress all output except violations",
				Sources: cli.EnvVars("GOLDMARK_LINT_QUIET"),
			},
			&cli.BoolFlag{
				Name:    "summary",
				Usage:   "print a count-per-rule breakdown after linting",
				Sources: cli.EnvVars("GOLDMARK_LINT_SUMMARY"),
			},
			&cli.BoolFlag{
				Name:    "watch",
				Usage:   "re-lint files whenever they change (runs until Ctrl+C)",
				Sources: cli.EnvVars("GOLDMARK_LINT_WATCH"),
			},
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:    "fix",
							Usage:   "updates files to resolve fixable issues",
							Sources: cli.EnvVars("GOLDMARK_LINT_FIX"),
						},
					},
					{
						&cli.BoolFlag{
							Name:    "fix-dry-run",
							Usage:   "show a diff of changes --fix would make, without modifying files",
							Sources: cli.EnvVars("GOLDMARK_LINT_FIX_DRY_RUN"),
						},
					},
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, cmd)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

// run contains the main linting logic, invoked by the cli action.
func run(_ context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")
	failOnWarning := cmd.Bool("fail-on-warning")
	fix := cmd.Bool("fix")
	fixDryRun := cmd.Bool("fix-dry-run")
	format := cmd.Bool("format")
	listRules := cmd.Bool("list-rules")
	noCache := cmd.Bool("no-cache")
	noGlobs := cmd.Bool("no-globs")
	outputFormat := cmd.String("output-format")
	quiet := cmd.Bool("quiet")
	summary := cmd.Bool("summary")
	watch := cmd.Bool("watch")

	// Validate --output-format flag if specified.
	if outputFormat != "" {
		switch outputFormat {
		case "default", "json", "junit", "tap", "sarif", "github":
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown output format %q; supported formats: default, json, junit, tap, sarif, github\n", outputFormat)
			os.Exit(2)
		}
	}

	// Auto-discover config file starting from the current working directory,
	// or use the explicitly specified --config path.
	var cfg *ConfigFile
	cwd, _ := os.Getwd()
	if configPath != "" {
		loaded, err := loadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", configPath, err)
			os.Exit(2)
		}
		cfg = loaded
	} else if cwd != "" {
		if cfgPath := findConfigFile(cwd); cfgPath != "" {
			loaded, err := loadConfig(cfgPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", cfgPath, err)
				os.Exit(2)
			}
			cfg = loaded
		}
	}

	// Determine the effective input globs: CLI args take priority, then config globs.
	// When --no-globs is set, config globs are ignored.
	inputGlobs := cmd.Args().Slice()
	if len(inputGlobs) == 0 && !noGlobs && cfg != nil && len(cfg.Globs) > 0 {
		inputGlobs = cfg.Globs
	}
	if listRules {
		var ruleCfgForList map[string]interface{}
		if cfg != nil {
			ruleCfgForList = cfg.Config
		}
		printRulesTable(os.Stdout, ruleCfgForList)
		os.Exit(0)
	}
	if len(inputGlobs) == 0 && !format {
		_ = cli.ShowAppHelp(cmd)
		os.Exit(2)
	}

	var ruleCfg map[string]interface{}
	var ignores []string
	var ignorePatterns []string
	var overrides []GlobOverride
	var noInlineConfig bool
	// effectiveFix is true when --fix is passed on CLI or fix:true is in config.
	effectiveFix := fix
	if cfg != nil {
		ruleCfg = cfg.Config
		ignores = cfg.Ignores
		ignorePatterns = cfg.IgnorePatterns
		overrides = cfg.Overrides
		noInlineConfig = cfg.NoInlineConfig
		if cfg.Fix {
			effectiveFix = true
		}
		// gitignore: read .gitignore files and add patterns to ignores.
		if gitignoreIsEnabled(cfg.Gitignore) && cwd != "" {
			pattern := gitignoreGlobPattern(cfg.Gitignore)
			if pattern == "" {
				// bool true: walk from cwd up to the git repository root.
				ignores = append(ignores, collectGitignorePatterns(cwd)...)
			} else {
				// string: use the glob pattern to find gitignore files.
				for _, f := range findFilesMatchingGlob(cwd, pattern) {
					ignores = append(ignores, parseGitignore(f)...)
				}
			}
		}
	}

	// Determine the formatter specs to use.
	// CLI flag takes priority; then config outputFormatters; then default.
	var formatterSpecs []outputFormatterSpec
	if outputFormat != "" {
		formatterSpecs = []outputFormatterSpec{{format: outputFormat}}
	} else if cfg != nil && len(cfg.OutputFormatters) > 0 {
		formatterSpecs = parseOutputFormatters(cfg.OutputFormatters)
	}
	if len(formatterSpecs) == 0 {
		formatterSpecs = []outputFormatterSpec{{format: "default"}}
	}

	// Default linter (used when no overrides are defined or override doesn't match).
	linter := newLinterFromConfig(ruleCfg)
	linter.NoInlineConfig = noInlineConfig
	// frontMatter: set custom front matter regexp on linter if configured.
	if cfg != nil && cfg.FrontMatter != "" {
		re, err := regexp.Compile(cfg.FrontMatter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid frontMatter regex %q: %v\n", cfg.FrontMatter, err)
			os.Exit(2)
		}
		linter.FrontMatterRegexp = re
	}

	// ignorePatterns: compile and validate each regex pattern at startup.
	ignorePatternRegexps := make([]*regexp.Regexp, 0, len(ignorePatterns))
	for _, p := range ignorePatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid ignorePattern regex %q: %v\n", p, err)
			os.Exit(2)
		}
		ignorePatternRegexps = append(ignorePatternRegexps, re)
	}

	// Load cache (skip when --no-cache, fix, fix-dry-run, or watch is used).
	useCache := !noCache && !effectiveFix && !fixDryRun && !watch
	cache := make(lintCache)
	if useCache && cwd != "" {
		cache = loadCache(cwd)
	}

	// --format: read stdin, apply fixes, write stdout, then exit.
	if format {
		source, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(2)
		}
		fixed := linter.Fix(source)
		if _, err := os.Stdout.Write(fixed); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing stdout: %v\n", err)
			os.Exit(2)
		}
		os.Exit(0)
	}

	exitCode := 0

	// allViolations collects violations from all sources for the final formatter run.
	var allViolations []fileViolation

	// Handle stdin ("-") sequentially – stdin cannot be parallelised.
	// Stdin can only be requested via CLI args (not config globs).
	for _, pattern := range cmd.Args().Slice() {
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
	for _, pattern := range inputGlobs {
		if pattern == "-" {
			continue
		}
		files, err := doublestar.FilepathGlob(pattern)
		if err != nil || len(files) == 0 {
			files = []string{pattern}
		}
		for _, file := range files {
			if !isIgnored(file, ignores) && !isIgnoredByPattern(file, ignorePatternRegexps) {
				allFiles = append(allFiles, file)
			}
		}
	}

	// fileResult carries the outcome of processing a single file.
	type fileResult struct {
		violations []lint.Violation
		err        error
		errCode    int
		original   []byte // non-nil when --fix-dry-run: the content before fixing
		fixed      []byte // non-nil when --fix-dry-run: the content after fixing
	}

	results := make([]fileResult, len(allFiles))
	newEntries := make(lintCache) // updated cache entries collected from goroutines
	var mu sync.Mutex             // protects newEntries

	// Bound the number of concurrent goroutines to avoid resource exhaustion
	// on large repositories. Using GOMAXPROCS as the concurrency limit ensures
	// we keep all CPUs busy without spawning more goroutines than useful.
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i, file := range allFiles {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()
			sem <- struct{}{} // acquire a slot; limits concurrent work
			defer func() { <-sem }() // release slot on exit

			source, err := os.ReadFile(file)
			if err != nil {
				results[i] = fileResult{err: err, errCode: 2}
				return
			}

			hash := hashContent(source)

			// Compute the effective rule config for this file once; it is used
			// both for the config-hash cache key and for building the linter.
			effectiveCfg := ruleCfg
			if len(overrides) > 0 {
				effectiveCfg = effectiveConfigForFile(ruleCfg, overrides, file)
			}
			configHash := hashConfig(effectiveCfg, version)

			// Cache hit: file unchanged AND effective config unchanged.
			if useCache {
				if entry, ok := cache[file]; ok && entry.Hash == hash && entry.ConfigHash == configHash {
					results[i] = fileResult{violations: entry.Violations}
					return
				}
			}

			// Determine the effective linter for this file.
			fileLinter := linter
			if len(overrides) > 0 {
				fileLinter = newLinterFromConfig(effectiveCfg)
				fileLinter.NoInlineConfig = noInlineConfig
				fileLinter.FrontMatterRegexp = linter.FrontMatterRegexp
			}

			// Apply fixes if requested.
			var origContent, fixedContent []byte
			if effectiveFix {
				fixedContent = fileLinter.Fix(source)
				if err := os.WriteFile(file, fixedContent, 0644); err != nil {
					results[i] = fileResult{err: err, errCode: 2}
					return
				}
				source = fixedContent
				hash = hashContent(source)
			} else if fixDryRun {
				fixedContent = fileLinter.Fix(source)
				origContent = source
				source = fixedContent
				hash = hashContent(source)
			}

			violations := fileLinter.Lint(source)
			results[i] = fileResult{violations: violations, original: origContent, fixed: fixedContent}

			// Store the new cache entry.
			if useCache {
				mu.Lock()
				newEntries[file] = cacheEntry{Hash: hash, ConfigHash: configHash, Violations: violations}
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

	// --fix-dry-run: output a unified diff for every file that would be changed.
	if fixDryRun && !quiet {
		color := isColorEnabled(os.Stdout)
		for i, file := range allFiles {
			r := results[i]
			if r.original != nil {
				formatFileDiff(file, r.original, r.fixed, os.Stdout, color)
			}
		}
	}

	// Apply per-violation severity so formatters can use it.
	for i := range allViolations {
		fileCfg := ruleCfg
		if len(overrides) > 0 && allViolations[i].File != "stdin" && allViolations[i].File != "-" {
			fileCfg = effectiveConfigForFile(ruleCfg, overrides, allViolations[i].File)
		}
		for j := range allViolations[i].Violations {
			allViolations[i].Violations[j].Severity = getRuleSeverity(allViolations[i].Violations[j].Rule, fileCfg)
		}
	}

	// Calculate violation-based exit code (only upgrade, never downgrade from 2).
	if exitCode < 1 {
		for _, fv := range allViolations {
			for _, v := range fv.Violations {
				if v.Severity != "warning" || failOnWarning {
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
		case "sarif":
			formatSARIF(allViolations, w)
		case "github":
			formatGitHubActions(allViolations, w)
		default:
			formatDefault(allViolations, w)
		}

		// Print per-rule summary if requested, using the same format and writer.
		if summary && !quiet {
			switch spec.format {
			case "json":
				formatSummaryJSON(allViolations, w)
			case "junit":
				formatSummaryJUnit(allViolations, w)
			case "tap":
				formatSummaryTAP(allViolations, w)
			case "sarif":
				formatSummaryJSON(allViolations, w)
			case "github":
				formatSummaryGitHub(allViolations, w)
			default:
				formatSummary(allViolations, os.Stderr)
			}
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

	// --watch: after the initial lint run, poll files for changes and re-lint.
	// Violations found during watch cycles are printed to stderr but do not
	// affect the exit code – the process exits 0 on interrupt (Ctrl+C) since
	// watch mode is an interactive session, not a one-shot check.
	if watch {
		var watchStatusOut io.Writer = os.Stderr
		if quiet {
			watchStatusOut = io.Discard
		}
		runWatch(allFiles, watchStatusOut, func(changed []string) {
			var watchViolations []fileViolation
			for _, file := range changed {
				source, err := os.ReadFile(file)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
					continue
				}
				if effectiveFix {
					fixed := linter.Fix(source)
					if err := os.WriteFile(file, fixed, 0644); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", file, err)
						continue
					}
					source = fixed
				}
				fileLinter := linter
				fileCfg := ruleCfg
				if len(overrides) > 0 {
					fileCfg = effectiveConfigForFile(ruleCfg, overrides, file)
					fileLinter = newLinterFromConfig(fileCfg)
					fileLinter.NoInlineConfig = noInlineConfig
					fileLinter.FrontMatterRegexp = linter.FrontMatterRegexp
				}
				violations := fileLinter.Lint(source)
				for j := range violations {
					violations[j].Severity = getRuleSeverity(violations[j].Rule, fileCfg)
				}
				watchViolations = append(watchViolations, fileViolation{File: file, Violations: violations})
			}
			formatDefault(watchViolations, os.Stderr)
		})
		return nil
	}

	if exitCode != 0 {
		return cli.Exit("", exitCode)
	}
	return nil
}

// printRulesTable writes a human-readable table of all known rules to w.
// Each row shows the rule ID, aliases, enabled/disabled state, and current
// option values (as a JSON object, omitting empty values).
func printRulesTable(w io.Writer, cfg map[string]interface{}) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "RULE\tALIASES\tENABLED\tOPTIONS"); err != nil {
		return
	}
	if _, err := fmt.Fprintln(tw, "----\t-------\t-------\t-------"); err != nil {
		return
	}
	for _, info := range buildAllRulesInfo(cfg) {
		aliases := ""
		if ar, ok := info.rule.(lint.AliasedRule); ok {
			aliases = strings.Join(ar.Aliases(), ", ")
		}
		enabled := "true"
		if !info.enabled {
			enabled = "false"
		}
		options := ruleOptionsJSON(info.rule)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", info.rule.ID(), aliases, enabled, options); err != nil {
			return
		}
	}
	_ = tw.Flush()
}

// ruleOptionsJSON marshals rule to JSON and returns the result, omitting the
// outer braces when there are no fields so that empty rules show "{}".
func ruleOptionsJSON(rule lint.Rule) string {
	data, err := json.Marshal(rule)
	if err != nil || string(data) == "null" {
		return "{}"
	}
	return string(data)
}
