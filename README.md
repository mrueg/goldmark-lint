# goldmark-lint

A Markdown linter written in Go using the
[goldmark](https://github.com/yuin/goldmark) parser.
It implements all rules from
[markdownlint](https://github.com/DavidAnson/markdownlint) /
[markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2) and
supports auto-fixing for
select rules.

## Table of Contents

- [Installation](#installation)
  - [Homebrew](#homebrew)
  - [Docker](#docker)
- [Library usage](#library-usage)
- [CLI usage](#cli-usage)
  - [Example](#example)
- [Configuration](#configuration)
  - [Config file format](#config-file-format)
  - [Simple config format (.markdownlint.yaml)](#simple-config-format-markdownlintyaml)
  - [Inline disable comments](#inline-disable-comments)
  - [Supported rule options](#supported-rule-options)
- [Features](#features)
- [Comparison with markdownlint-cli2](#comparison-with-markdownlint-cli2)
  - [`--fail-on-warning`](#--fail-on-warning)
  - [`--fix-dry-run`](#--fix-dry-run)
  - [`--list-rules`](#--list-rules)
  - [`--summary`](#--summary)
  - [`--watch`](#--watch)
- [Performance & Conformance](#performance--conformance)
  - [Benchmark](#benchmark)
  - [Conformance](#conformance)
- [Rules](#rules)
- [License](#license)

## Installation

```sh
go install github.com/mrueg/goldmark-lint/cmd/goldmark-lint@latest
```

### Homebrew

```sh
brew install mrueg/tap/goldmark-lint
```

### Docker

Container images are published to the GitHub Container Registry:

```sh
# Lint all Markdown files in the current directory tree
docker run --rm -v "$(pwd):/work" -w /work ghcr.io/mrueg/goldmark-lint '**/*.md'

# Read from stdin
echo "# Hello" | docker run --rm -i ghcr.io/mrueg/goldmark-lint -
```

## Library usage

goldmark-lint can also be used as a Go library. Import the `lint` and
`lint/rules` packages:

```go
import (
    "fmt"

    "github.com/mrueg/goldmark-lint/lint/rules"
)

func main() {
    linter := rules.NewDefaultLinter()
    violations := linter.Lint([]byte("# Hello\n\nsome text\n"))
    for _, v := range violations {
        fmt.Printf("line %d: [%s] %s\n", v.Line, v.Rule, v.Message)
    }
}
```

To enable only specific rules, or to customise rule options, construct the
linter directly with [lint.NewLinter]:

```go
import (
    "github.com/mrueg/goldmark-lint/lint"
    "github.com/mrueg/goldmark-lint/lint/rules"
)

linter := lint.NewLinter(
    rules.MD001{},
    rules.MD013{LineLength: 100},
)
violations := linter.Lint(source)
```

To auto-fix issues in a document, call `linter.Fix` with the source bytes.
It applies all rules that implement the `lint.FixableRule` interface and
returns the corrected content:

```go
import (
    "os"

    "github.com/mrueg/goldmark-lint/lint/rules"
)

source, _ := os.ReadFile("README.md")
linter := rules.NewDefaultLinter()
fixed := linter.Fix(source)
_ = os.WriteFile("README.md", fixed, 0644)
```

To implement a custom rule that also supports auto-fixing, implement the
`lint.FixableRule` interface by adding a `Fix(source []byte) []byte` method:

```go
import "github.com/mrueg/goldmark-lint/lint"

type MyRule struct{}

func (r MyRule) ID() string          { return "MY001" }
func (r MyRule) Description() string { return "My custom rule" }

func (r MyRule) Check(doc *lint.Document) []lint.Violation {
    // ... return violations
    return nil
}

// Fix rewrites source to resolve violations found by Check.
func (r MyRule) Fix(source []byte) []byte {
    // ... apply fixes and return corrected source
    return source
}
```

## CLI usage

```text
goldmark-lint glob0 [glob1] [...] [globN] [--fix] [--help] [--version]
goldmark-lint - (read from stdin)
goldmark-lint --format (read stdin, apply fixes, write stdout)

Glob expressions:
  *  matches any number of characters, but not /
  ?  matches a single character, but not /
  ** matches any number of characters, including /

Optional parameters:
  --config           path to config file (overrides auto-discovery)
  --fail-on-warning  exit with code 1 even when all violations are warnings
  --fix              updates files to resolve fixable issues
  --fix-dry-run      show a diff of changes --fix would make, without modifying files
  --format           read stdin, apply fixes, write stdout
  --list-rules       print a table of all rules with their aliases, enabled/disabled state, and options
  --no-cache         disable reading/writing the .goldmark-lint-cache file
  --no-globs         ignore the globs config key at runtime
  --output-format    output format: default, json, junit, tap, sarif, github (default: default)
  --summary          print a count-per-rule breakdown after linting
  --watch            re-lint files whenever they change (runs until Ctrl+C)
  --help             writes this message to the console and exits without doing anything else
  --version          prints the version and exits

Exit codes:
  0: Linting was successful and there were no errors
  1: Linting was successful and there were errors
  2: Linting was not successful due to a problem or failure
```

### Example

```sh
# Lint all Markdown files in the current directory tree
goldmark-lint '**/*.md'

# Lint and auto-fix fixable issues
goldmark-lint --fix '**/*.md'

# Preview what --fix would change (git diff style, no files modified)
goldmark-lint --fix-dry-run '**/*.md'

# Treat warnings as errors (useful for strict CI gates)
goldmark-lint --fail-on-warning '**/*.md'

# Read from stdin and report violations
goldmark-lint -

# Read from stdin, apply fixes, write to stdout (useful as an editor formatter)
goldmark-lint --format

# Use a custom config file
goldmark-lint --config path/to/.markdownlint-cli2.yaml '**/*.md'

# Re-lint files on every change (interactive watch mode)
goldmark-lint --watch '**/*.md'

# Print all rules with their enabled state and current options
goldmark-lint --list-rules

# Print a violation count per rule after linting
goldmark-lint --summary '**/*.md'

# Lint only Markdown files changed relative to the base branch (CI)
goldmark-lint $(git diff --name-only origin/main -- '*.md' '**/*.md')

# Lint only Markdown files with uncommitted changes (local)
goldmark-lint $(git diff --name-only -- '*.md' '**/*.md')
```

## Configuration

goldmark-lint reads configuration from a `.markdownlint-cli2.yaml` (or `.yml`,
`.jsonc`, `.json`) file, following the same discovery and format as
[markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2).

It also reads `.markdownlint.yaml` (or `.yml`, `.jsonc`, `.json`) files, which
use a simpler rule-only format (compatible with
[vscode-markdownlint](https://github.com/DavidAnson/vscode-markdownlint)).
`.markdownlint-cli2.*` files take priority when both are present.

The config file is searched starting from the current working directory and
walking up to the filesystem root. The first file found is used. The `--config`
flag overrides auto-discovery with an explicit path.

### Config file format

```yaml
# .markdownlint-cli2.yaml
config:
  default: true          # enable all rules (this is the default when omitted)
  MD013:
    line_length: 100     # override line length for MD013
  MD033:
    allowed_elements:    # allow specific HTML elements
      - br
  MD001: false           # disable MD001

ignores:
  - "vendor/**"          # ignore files matching these glob patterns
  - "node_modules/**"

# Inherit settings from another config file (merged with this file's settings)
extends: base-config.yaml

# Per-glob rule config overrides (applied in order; last match wins)
overrides:
  - files:
      - "docs/**"
    config:
      MD013:
        line_length: 120

# Default input globs when no CLI arguments are provided
globs:
  - "**/*.md"

# Enable --fix behaviour from the config file
fix: false

# Custom front matter pattern (Go regular expression)
frontMatter: "---[\\s\\S]*?---"

# Auto-ignore .gitignore entries (true = walk to git root; string = glob for gitignore files)
gitignore: true

# Disable inline markdownlint-disable comments
noInlineConfig: false

# Output formatters (same format as markdownlint-cli2)
outputFormatters:
  - - markdownlint-cli2-formatter-json
    - outfile: results.json
```

The `outputFormatters` key accepts a list of formatters. Each entry is a list
whose first element is the formatter name and whose optional second element is
an options object (supporting `outfile` to write output to a file instead of
stdout). Supported formatter names:

| Formatter name                           | Format           |
|------------------------------------------|------------------|
| `markdownlint-cli2-formatter-default`    | Default text     |
| `markdownlint-cli2-formatter-json`       | JSON array       |
| `markdownlint-cli2-formatter-junit`      | JUnit XML        |
| `markdownlint-cli2-formatter-tap`        | TAP              |
| `markdownlint-cli2-formatter-sarif`      | SARIF 2.1.0      |
| `markdownlint-cli2-formatter-github`     | GitHub Actions   |

The `--output-format` CLI flag overrides `outputFormatters` from the config
and accepts `default`, `json`, `junit`, `tap`, `sarif`, or `github`.

The `config` section mirrors the
[markdownlint configuration](https://github.com/DavidAnson/markdownlint#options)
format:

- Set a rule ID to `false` to disable it.
- Set a rule ID to `true` to enable it with default options.
- Set a rule ID to `"warning"` to enable it with warning severity (exit code 0).
- Set a rule ID to an object to enable it with specific options.
- Set `default: false` to disable all rules not explicitly listed.

### Simple config format (.markdownlint.yaml)

The `.markdownlint.yaml` (and `.yml`, `.json`, `.jsonc`) files use a flat
rule-only format where the entire file is a rule config map:

```yaml
# .markdownlint.yaml
default: true
MD013:
  line_length: 100
MD001: false
```

### Inline disable comments

goldmark-lint supports the same inline disable comment syntax as markdownlint:

```markdown
<!-- markdownlint-disable MD001 -->
Violations on this and following lines are suppressed for MD001.
<!-- markdownlint-enable MD001 -->

<!-- markdownlint-disable-next-line MD013 -->
This line's MD013 violation is suppressed.

This line's MD009 violation is suppressed. <!-- markdownlint-disable-line MD009 -->

<!-- markdownlint-disable-file MD001 -->
MD001 is suppressed for the entire file regardless of comment position.

<!-- markdownlint-enable-file MD001 -->
Re-enables a file-level disable for the remainder of the file.

<!-- markdownlint-capture -->
<!-- markdownlint-disable MD001 -->
Capture the current disable state, then disable MD001.
<!-- markdownlint-restore -->
Disable state is restored to what it was at the capture point.

<!-- markdownlint-configure-file { "MD001": false } -->
File-level rule configuration via JSON.
```

Omit the rule ID to disable/enable all rules. Rule aliases (e.g.
`heading-increment` for MD001) are also accepted.

### Supported rule options

| Rule  | Option                | Default                               | Description                                                                                                        |
| ----- | --------------------- | ------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| MD003 | `style`               | `consistent`                          | Heading style (`atx`, `setext`, `consistent`)                                                                      |
| MD004 | `style`               | `consistent`                          | Unordered list marker style (`asterisk`, `dash`, `plus`, `consistent`)                                             |
| MD007 | `indent`              | `2`                                   | Spaces per indentation level                                                                                       |
| MD009 | `br_spaces`           | `2`                                   | Trailing spaces allowed for line breaks                                                                            |
| MD012 | `maximum`             | `1`                                   | Max consecutive blank lines                                                                                        |
| MD013 | `line_length`         | `80`                                  | Maximum line length                                                                                                |
| MD022 | `lines_above`         | `1`                                   | Blank lines required above headings                                                                                |
| MD022 | `lines_below`         | `1`                                   | Blank lines required below headings                                                                                |
| MD024 | `siblings_only`       | `false`                               | Only check sibling headings                                                                                        |
| MD025 | `level`               | `1`                                   | Top-level heading level                                                                                            |
| MD026 | `punctuation`         | `.,;:!。，；：！`                     | Punctuation characters to check in headings                                                                        |
| MD029 | `style`               | `one_or_ordered`                      | Ordered list numbering style                                                                                       |
| MD030 | `ul_single`           | `1`                                   | Spaces after unordered list marker (single-line item)                                                              |
| MD030 | `ol_single`           | `1`                                   | Spaces after ordered list marker (single-line item)                                                                |
| MD033 | `allowed_elements`    | `[]`                                  | HTML elements that are allowed                                                                                     |
| MD035 | `style`               | `consistent`                          | Horizontal rule style (e.g. `---`, `***`, `consistent`)                                                            |
| MD036 | `punctuation`         | `.,;:!?。，；：！？`                  | Punctuation that exempts a line from the check                                                                     |
| MD041 | `level`               | `1`                                   | Required first-line heading level                                                                                  |
| MD043 | `headings`            | `[]`                                  | Required heading structure list                                                                                    |
| MD043 | `match_case`          | `false`                               | Require exact case match for headings                                                                              |
| MD044 | `names`               | `[]`                                  | Proper names to enforce correct capitalisation                                                                     |
| MD044 | `code_blocks`         | `true`                                | Check inside code blocks                                                                                           |
| MD044 | `html_elements`       | `true`                                | Check inside HTML elements                                                                                         |
| MD046 | `style`               | `consistent`                          | Code block style (`fenced`, `indented`, `consistent`)                                                              |
| MD048 | `style`               | `consistent`                          | Code fence style (`backtick`, `tilde`, `consistent`)                                                               |
| MD049 | `style`               | `consistent`                          | Emphasis style (`asterisk`, `underscore`, `consistent`)                                                            |
| MD050 | `style`               | `consistent`                          | Strong style (`asterisk`, `underscore`, `consistent`)                                                              |
| MD051 | `ignore_case`         | `false`                               | Ignore case when comparing link fragments                                                                          |
| MD051 | `ignored_pattern`     | `""`                                  | Regex pattern for fragments to ignore                                                                              |
| MD052 | `shortcut_syntax`     | `false`                               | Also check shortcut reference syntax `[label]`                                                                     |
| MD052 | `ignored_labels`      | `["x"]`                               | Reference labels to ignore                                                                                         |
| MD053 | `ignored_definitions` | `["//"]`                              | Reference definitions to ignore                                                                                    |
| MD054 | `autolink`            | `true`                                | Allow autolinks `<url>`                                                                                            |
| MD054 | `collapsed`           | `true`                                | Allow collapsed reference links `[label][]`                                                                        |
| MD054 | `full`                | `true`                                | Allow full reference links `[text][label]`                                                                         |
| MD054 | `inline`              | `true`                                | Allow inline links `[text](url)`                                                                                   |
| MD054 | `shortcut`            | `true`                                | Allow shortcut reference links `[label]`                                                                           |
| MD054 | `url_inline`          | `true`                                | Allow inline links whose text equals their URL                                                                     |
| MD055 | `style`               | `consistent`                          | Table pipe style (`leading_and_trailing`, `leading_only`, `trailing_only`, `no_leading_or_trailing`, `consistent`) |
| MD059 | `prohibited_texts`    | `["click here","here","link","more"]` | Generic link text phrases to prohibit                                                                              |
| MD060 | `style`               | `any`                                 | Table column style (`aligned`, `compact`, `tight`, `any`)                                                          |
| MD060 | `aligned_delimiter`   | `false`                               | Require delimiter row to align with header                                                                         |

## Features

- Parses Markdown with the goldmark library for accurate, spec-compliant analysis.
- Reports violations with file, line, and column information.
- Auto-fix support (`--fix`) for a subset of rules.
- Dry-run preview (`--fix-dry-run`): shows a git diff style unified diff of all
  changes `--fix` would make, without touching any files.
- stdin support: lint with `goldmark-lint -` or format with `goldmark-lint --format`.
- Watch mode (`--watch`): re-lint files on every change, running until interrupted.
- Configuration file discovery: searches from the current directory up to the
  filesystem root.
- Supports `.markdownlint-cli2.yaml` and `.markdownlint.yaml` config formats
  (YAML, JSON, and JSONC with comment stripping).
- Config inheritance via `extends` for composable configuration.
- Per-glob rule overrides via `overrides` for fine-grained control.
- Rules configurable by ID, alias (e.g. `no-hard-tabs`), or tag (e.g.
  `whitespace: false` disables all whitespace-tagged rules).
- Warning severity: set a rule to `"warning"` for informational violations that
  don't fail the build; combine with `--fail-on-warning` to override.
- `noInlineConfig` config key to globally disable inline `markdownlint-disable`
  comments.
- Inline disable comments (`markdownlint-disable`,
  `markdownlint-disable-next-line`, `markdownlint-disable-line`,
  `markdownlint-disable-file`, `markdownlint-enable-file`,
  `markdownlint-capture`/`restore`, `markdownlint-configure-file`).
- Multiple output formats via `--output-format`: default text, JSON, JUnit XML,
  TAP, SARIF, and GitHub Actions annotations.
- Colored terminal output: violations and diffs use ANSI colors when writing to
  a TTY (suppressed by `NO_COLOR`).
- Result caching via `.goldmark-lint-cache` to speed up repeated runs.
- Parallel file linting bounded by `GOMAXPROCS` for fast, deterministic output
  on large repositories.
- Gitignore integration via the `gitignore` config key.
- `--list-rules` flag to inspect all rules with their enabled state and current
  options.
- `--summary` flag to print a per-rule violation count after linting.

## Comparison with markdownlint-cli2

goldmark-lint adds several features beyond what markdownlint-cli2 provides, but
markdownlint-cli2 also has capabilities that goldmark-lint does not:

| Feature                                                         | goldmark-lint | markdownlint-cli2 |
| --------------------------------------------------------------- | :-----------: | :---------------: |
| `--fail-on-warning` flag (exit code 1 for warnings)             | ✅            | ❌                |
| `--fix-dry-run` flag (diff preview without modifying files)     | ✅            | ❌                |
| SARIF output format                                             | ✅            | ❌                |
| GitHub Actions annotation output format                         | ✅            | ❌                |
| `--list-rules` flag (inspect rules, options, and enabled state) | ✅            | ❌                |
| `--summary` flag (per-rule violation count breakdown)           | ✅            | ❌                |
| Single self-contained binary (no Node.js required)              | ✅            | ❌                |
| Embeddable Go library                                           | ✅            | ❌                |
| Custom rule plugins                                             | ❌            | ✅                |
| Shared configurations via npm packages                          | ❌            | ✅                |

### `--fail-on-warning`

By default, violations marked as `"warning"` severity in the config do not cause
a non-zero exit code. The `--fail-on-warning` flag changes this so that any
violation — regardless of severity — causes goldmark-lint to exit with code 1.
This is useful for stricter CI gates:

```sh
goldmark-lint --fail-on-warning '**/*.md'
```

### `--fix-dry-run`

Preview all changes that `--fix` would apply as a unified diff in git diff
style, without modifying any files on disk. The diff is written to stdout so it
can be inspected, piped, or saved. ANSI colors are used when stdout is a
terminal (`NO_COLOR` suppresses them).

The exit code follows the same logic as `--fix`: it reflects remaining
violations found after the would-be fixes are applied, so the result is
identical to what you would observe if you ran `--fix` and then re-linted.

`--fix-dry-run` and `--fix` are mutually exclusive; specifying both exits with
code 2.

```sh
# Show what --fix would change across all Markdown files
goldmark-lint --fix-dry-run '**/*.md'

# Save the proposed patch for review
goldmark-lint --fix-dry-run '**/*.md' > proposed-fixes.patch
```

Example output:

```diff
diff --git a/docs/guide.md b/docs/guide.md
--- a/docs/guide.md
+++ b/docs/guide.md
@@ -1,7 +1,7 @@
 # Guide
 
 Some intro text.
 
-Trailing spaces here.   
+Trailing spaces here.
 
 More content.
```

### `--list-rules`

Print a table of every known rule with its ID, aliases, enabled/disabled state,
and current option values (as JSON). Useful for inspecting which rules are active
and what options they use with the current config:

```sh
goldmark-lint --list-rules
goldmark-lint --config path/to/.markdownlint-cli2.yaml --list-rules
```

### `--summary`

Print a per-rule count of violations after linting finishes. Useful for
identifying which rules produce the most noise in a project:

```sh
goldmark-lint --summary '**/*.md'
```

Example output:

```text
Summary:
  MD013: 42
  MD009:  7
  MD047:  3
```

### `--watch`

Re-lint files whenever they change, running until interrupted (Ctrl+C). Useful
for keeping a terminal open while editing Markdown:

```sh
goldmark-lint --watch '**/*.md'
```

## Performance & Conformance

The numbers below were produced by running [`bench/bench.sh`](bench/bench.sh)
and [`bench/conform.sh`](bench/conform.sh) against two real-world corpora at
fixed commits.

### Benchmark

**Corpus:** same two repositories as the conformance run:

- [rust-lang/rfcs](https://github.com/rust-lang/rfcs) `c143e315` — 636 files
- [tldr-pages/tldr](https://github.com/tldr-pages/tldr) `05c563d1` — 33,769 files
- **Total: 34,405 Markdown files**

Measured with [hyperfine](https://github.com/sharkdp/hyperfine)
(10 runs, 3 warmup runs). goldmark-lint's content cache was enabled (default).

| Tool               | Mean       | Min        | Max        |
|--------------------|------------|------------|------------|
| goldmark-lint      | 7.941 s    | 7.898 s    | 8.054 s    |
| markdownlint-cli2  | 46.866 s   | 45.376 s   | 48.124 s   |

**goldmark-lint is ~5.9× faster than markdownlint-cli2** on this corpus.

Note that this corpus is made up of many small files — 34,405 files averaging
roughly 30 lines each — so it measures per-file overhead rather than how either
tool scales within a single large document. For that dimension, linting one
synthetic file of mixed headings, prose and fenced code:

| Lines in one file | goldmark-lint | markdownlint |
|-------------------|---------------|--------------|
| 5,000             | 0.29 s        | 1.76 s       |
| 10,000            | 0.42 s        | 2.33 s       |
| 20,000            | 0.87 s        | 3.91 s       |

To reproduce:

```sh
./bench/bench.sh
```

`bench/bench.sh` accepts `--runs N`, `--warmup N` and `--no-cache`. The CI
benchmark workflow passes `--no-cache` so that every run does the full parsing
work; the table above is from a default (cached) run.

### Conformance

**Corpus:** two real-world repositories at fixed commits:

- [rust-lang/rfcs](https://github.com/rust-lang/rfcs) `c143e315` — 636 files
- [tldr-pages/tldr](https://github.com/tldr-pages/tldr) `05c563d1` — 33,769 files
- **Total: 34,405 Markdown files**

Both tools were run with default settings. The table compares per-rule
violation counts; a delta of `0` means the tools agree on that rule's total.

| Rule      | goldmark-lint | markdownlint-cli2 | delta   |
| --------- | ------------: | ----------------: | ------: |
| MD001     |            16 |                16 |      +0 |
| MD003     |             3 |                 3 |      +0 |
| MD004     |         4,585 |             4,585 |      +0 |
| MD005     |            11 |                11 |      +0 |
| MD007     |         1,247 |             1,247 |      +0 |
| MD009     |           414 |               414 |      +0 |
| MD010     |           124 |               124 |      +0 |
| MD011     |             5 |                 5 |      +0 |
| MD012     |           858 |               858 |      +0 |
| MD013     |        33,217 |            33,217 |      +0 |
| MD014     |            13 |                13 |      +0 |
| MD019     |             2 |                 2 |      +0 |
| MD020     |             2 |                 2 |      +0 |
| MD022     |         3,166 |             3,166 |      +0 |
| MD024     |            91 |                91 |      +0 |
| MD026     |           169 |               169 |      +0 |
| MD027     |            15 |                15 |      +0 |
| MD028     |            66 |                66 |      +0 |
| MD029     |           110 |               110 |      +0 |
| MD030     |            63 |                63 |      +0 |
| MD031     |           908 |               908 |      +0 |
| MD032     |           539 |               562 |     -23 |
| MD033     |           214 |               214 |      +0 |
| MD034     |           355 |               351 |      +4 |
| MD035     |             3 |                 3 |      +0 |
| MD036     |            63 |                63 |      +0 |
| MD038     |            21 |                22 |      -1 |
| MD039     |             3 |                 3 |      +0 |
| MD040     |           536 |               536 |      +0 |
| MD041     |           621 |               621 |      +0 |
| MD045     |             2 |                 2 |      +0 |
| MD046     |           137 |               141 |      -4 |
| MD047     |             8 |                 8 |      +0 |
| MD049     |           346 |               346 |      +0 |
| MD050     |            24 |                24 |      +0 |
| MD051     |           233 |               233 |      +0 |
| MD052     |             9 |                10 |      -1 |
| MD053     |         3,220 |             3,220 |      +0 |
| MD055     |            71 |                71 |      +0 |
| MD056     |             6 |                 6 |      +0 |
| MD058     |            48 |                48 |      +0 |
| MD059     |            71 |                71 |      +0 |
| MD060     |         2,151 |             2,151 |      +0 |
| **TOTAL** |    **53,766** |        **53,791** | **-25** |

38 of the 43 rules produce identical counts. The remaining five (MD032, MD034,
MD038, MD046 and MD052) differ by 33 violations in total when counted without
regard to direction, and by -25 on balance — under 0.1% of the total either
way.

Counts alone can hide disagreement, because a false positive in one file and a
missed violation in another cancel out. `conform.sh` therefore also compares
the individual `(file, line, rule)` locations:

| Measure                              | Count  |
| ------------------------------------ | -----: |
| Locations both tools agree on        | 51,982 |
| Reported only by goldmark-lint       |      0 |
| Reported only by markdownlint-cli2   |      8 |

goldmark-lint reports no violation that markdownlint does not. The eight
locations it misses are MD046 (4) and MD032 (3), where goldmark and micromark
disagree about whether indented text following a flush-left link reference
definition is list content or an indented code block, and MD052 (1), a
reference link whose text spans two source lines.

To reproduce:

```sh
./bench/conform.sh
```

## Rules

The table below lists all [markdownlint rules](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md).
Rules that are implemented in goldmark-lint are marked ✅. Rules marked 🔧 also
support auto-fixing.
Rules that markdownlint itself has deprecated (MD002, MD006) or never assigned
(MD008, MD015–MD017, MD057) are not listed.

| Rule                                                                       | Description                                                   | Status |
| -------------------------------------------------------------------------- | ------------------------------------------------------------- | ------ |
| [MD001](https://github.com/DavidAnson/markdownlint/blob/main/doc/md001.md) | Heading levels should only increment by one level at a time   | ✅ 🔧  |
| [MD003](https://github.com/DavidAnson/markdownlint/blob/main/doc/md003.md) | Heading style                                                 | ✅ 🔧  |
| [MD004](https://github.com/DavidAnson/markdownlint/blob/main/doc/md004.md) | Unordered list style                                          | ✅ 🔧  |
| [MD005](https://github.com/DavidAnson/markdownlint/blob/main/doc/md005.md) | Inconsistent indentation for list items at the same level     | ✅ 🔧  |
| [MD007](https://github.com/DavidAnson/markdownlint/blob/main/doc/md007.md) | Unordered list indentation                                    | ✅ 🔧  |
| [MD009](https://github.com/DavidAnson/markdownlint/blob/main/doc/md009.md) | Trailing spaces                                               | ✅ 🔧  |
| [MD010](https://github.com/DavidAnson/markdownlint/blob/main/doc/md010.md) | Hard tabs                                                     | ✅ 🔧  |
| [MD011](https://github.com/DavidAnson/markdownlint/blob/main/doc/md011.md) | Reversed link syntax                                          | ✅ 🔧  |
| [MD012](https://github.com/DavidAnson/markdownlint/blob/main/doc/md012.md) | Multiple consecutive blank lines                              | ✅ 🔧  |
| [MD013](https://github.com/DavidAnson/markdownlint/blob/main/doc/md013.md) | Line length                                                   | ✅     |
| [MD014](https://github.com/DavidAnson/markdownlint/blob/main/doc/md014.md) | Dollar signs used before commands without showing output      | ✅ 🔧  |
| [MD018](https://github.com/DavidAnson/markdownlint/blob/main/doc/md018.md) | No space after hash on ATX style heading                      | ✅ 🔧  |
| [MD019](https://github.com/DavidAnson/markdownlint/blob/main/doc/md019.md) | Multiple spaces after hash on ATX style heading               | ✅ 🔧  |
| [MD020](https://github.com/DavidAnson/markdownlint/blob/main/doc/md020.md) | No space inside hashes on closed ATX style heading            | ✅ 🔧  |
| [MD021](https://github.com/DavidAnson/markdownlint/blob/main/doc/md021.md) | Multiple spaces inside hashes on closed ATX style heading     | ✅ 🔧  |
| [MD022](https://github.com/DavidAnson/markdownlint/blob/main/doc/md022.md) | Headings should be surrounded by blank lines                  | ✅ 🔧  |
| [MD023](https://github.com/DavidAnson/markdownlint/blob/main/doc/md023.md) | Headings must start at the beginning of the line              | ✅ 🔧  |
| [MD024](https://github.com/DavidAnson/markdownlint/blob/main/doc/md024.md) | Multiple headings with the same content                       | ✅     |
| [MD025](https://github.com/DavidAnson/markdownlint/blob/main/doc/md025.md) | Multiple top-level headings in the same document              | ✅     |
| [MD026](https://github.com/DavidAnson/markdownlint/blob/main/doc/md026.md) | Trailing punctuation in heading                               | ✅ 🔧  |
| [MD027](https://github.com/DavidAnson/markdownlint/blob/main/doc/md027.md) | Multiple spaces after blockquote symbol                       | ✅ 🔧  |
| [MD028](https://github.com/DavidAnson/markdownlint/blob/main/doc/md028.md) | Blank line inside blockquote                                  | ✅ 🔧  |
| [MD029](https://github.com/DavidAnson/markdownlint/blob/main/doc/md029.md) | Ordered list item prefix                                      | ✅ 🔧  |
| [MD030](https://github.com/DavidAnson/markdownlint/blob/main/doc/md030.md) | Spaces after list markers                                     | ✅ 🔧  |
| [MD031](https://github.com/DavidAnson/markdownlint/blob/main/doc/md031.md) | Fenced code blocks should be surrounded by blank lines        | ✅ 🔧  |
| [MD032](https://github.com/DavidAnson/markdownlint/blob/main/doc/md032.md) | Lists should be surrounded by blank lines                     | ✅ 🔧  |
| [MD033](https://github.com/DavidAnson/markdownlint/blob/main/doc/md033.md) | Inline HTML                                                   | ✅ 🔧  |
| [MD034](https://github.com/DavidAnson/markdownlint/blob/main/doc/md034.md) | Bare URL used                                                 | ✅ 🔧  |
| [MD035](https://github.com/DavidAnson/markdownlint/blob/main/doc/md035.md) | Horizontal rule style                                         | ✅ 🔧  |
| [MD036](https://github.com/DavidAnson/markdownlint/blob/main/doc/md036.md) | Emphasis used instead of a heading                            | ✅ 🔧  |
| [MD037](https://github.com/DavidAnson/markdownlint/blob/main/doc/md037.md) | Spaces inside emphasis markers                                | ✅ 🔧  |
| [MD038](https://github.com/DavidAnson/markdownlint/blob/main/doc/md038.md) | Spaces inside code span elements                              | ✅ 🔧  |
| [MD039](https://github.com/DavidAnson/markdownlint/blob/main/doc/md039.md) | Spaces inside link text                                       | ✅ 🔧  |
| [MD040](https://github.com/DavidAnson/markdownlint/blob/main/doc/md040.md) | Fenced code blocks should have a language specified           | ✅ 🔧  |
| [MD041](https://github.com/DavidAnson/markdownlint/blob/main/doc/md041.md) | First line in a file should be a top-level heading            | ✅     |
| [MD042](https://github.com/DavidAnson/markdownlint/blob/main/doc/md042.md) | No empty links                                                | ✅     |
| [MD043](https://github.com/DavidAnson/markdownlint/blob/main/doc/md043.md) | Required heading structure                                    | ✅     |
| [MD044](https://github.com/DavidAnson/markdownlint/blob/main/doc/md044.md) | Proper names should have the correct capitalization           | ✅ 🔧  |
| [MD045](https://github.com/DavidAnson/markdownlint/blob/main/doc/md045.md) | Images should have alternate text (alt text)                  | ✅     |
| [MD046](https://github.com/DavidAnson/markdownlint/blob/main/doc/md046.md) | Code block style                                              | ✅ 🔧  |
| [MD047](https://github.com/DavidAnson/markdownlint/blob/main/doc/md047.md) | Files should end with a single newline character              | ✅ 🔧  |
| [MD048](https://github.com/DavidAnson/markdownlint/blob/main/doc/md048.md) | Code fence style                                              | ✅ 🔧  |
| [MD049](https://github.com/DavidAnson/markdownlint/blob/main/doc/md049.md) | Emphasis style                                                | ✅ 🔧  |
| [MD050](https://github.com/DavidAnson/markdownlint/blob/main/doc/md050.md) | Strong style                                                  | ✅ 🔧  |
| [MD051](https://github.com/DavidAnson/markdownlint/blob/main/doc/md051.md) | Link fragments should be valid                                | ✅     |
| [MD052](https://github.com/DavidAnson/markdownlint/blob/main/doc/md052.md) | Reference links and images should use a label that is defined | ✅     |
| [MD053](https://github.com/DavidAnson/markdownlint/blob/main/doc/md053.md) | Link and image reference definitions should be needed         | ✅ 🔧  |
| [MD054](https://github.com/DavidAnson/markdownlint/blob/main/doc/md054.md) | Link and image style                                          | ✅     |
| [MD055](https://github.com/DavidAnson/markdownlint/blob/main/doc/md055.md) | Table pipe style                                              | ✅ 🔧  |
| [MD056](https://github.com/DavidAnson/markdownlint/blob/main/doc/md056.md) | Table column count                                            | ✅     |
| [MD058](https://github.com/DavidAnson/markdownlint/blob/main/doc/md058.md) | Tables should be surrounded by blank lines                    | ✅ 🔧  |
| [MD059](https://github.com/DavidAnson/markdownlint/blob/main/doc/md059.md) | Link text should be descriptive                               | ✅     |
| [MD060](https://github.com/DavidAnson/markdownlint/blob/main/doc/md060.md) | Table column style                                            | ✅ 🔧  |

## License

[MIT](LICENSE)
