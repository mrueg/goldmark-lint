# goldmark-lint

A Markdown linter written in Go using the [goldmark](https://github.com/yuin/goldmark) parser.
It implements all rules from [markdownlint](https://github.com/DavidAnson/markdownlint) /
[markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2) and supports auto-fixing for
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
  - [`--list-rules`](#--list-rules)
  - [`--summary`](#--summary)
  - [`--watch`](#--watch)
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

```
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
  --format           read stdin, apply fixes, write stdout
  --list-rules       print a table of all rules with their aliases, enabled/disabled state, and options
  --no-cache         disable reading/writing the .markdownlint-cli2-cache file
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

<!-- markdownlint-configure-file { "MD001": false } -->
File-level rule configuration via JSON.
```

Omit the rule ID to disable/enable all rules. Rule aliases (e.g.
`heading-increment` for MD001) are also accepted.

### Supported rule options

| Rule  | Option                 | Default                              | Description                                          |
|-------|------------------------|--------------------------------------|------------------------------------------------------|
| MD003 | `style`                | `consistent`                         | Heading style (`atx`, `setext`, `consistent`)        |
| MD004 | `style`                | `consistent`                         | Unordered list marker style (`asterisk`, `dash`, `plus`, `consistent`) |
| MD007 | `indent`               | `2`                                  | Spaces per indentation level                         |
| MD009 | `br_spaces`            | `2`                                  | Trailing spaces allowed for line breaks              |
| MD012 | `maximum`              | `1`                                  | Max consecutive blank lines                          |
| MD013 | `line_length`          | `80`                                 | Maximum line length                                  |
| MD022 | `lines_above`          | `1`                                  | Blank lines required above headings                  |
| MD022 | `lines_below`          | `1`                                  | Blank lines required below headings                  |
| MD024 | `siblings_only`        | `false`                              | Only check sibling headings                          |
| MD025 | `level`                | `1`                                  | Top-level heading level                              |
| MD026 | `punctuation`          | `.,;:!。，；：！`                    | Punctuation characters to check in headings          |
| MD029 | `style`                | `one_or_ordered`                     | Ordered list numbering style                         |
| MD030 | `ul_single`            | `1`                                  | Spaces after unordered list marker (single-line item) |
| MD030 | `ol_single`            | `1`                                  | Spaces after ordered list marker (single-line item)  |
| MD033 | `allowed_elements`     | `[]`                                 | HTML elements that are allowed                       |
| MD035 | `style`                | `consistent`                         | Horizontal rule style (e.g. `---`, `***`, `consistent`) |
| MD036 | `punctuation`          | `.,;:!?。，；：！？`                 | Punctuation that exempts a line from the check       |
| MD041 | `level`                | `1`                                  | Required first-line heading level                    |
| MD043 | `headings`             | `[]`                                 | Required heading structure list                      |
| MD043 | `match_case`           | `false`                              | Require exact case match for headings                |
| MD044 | `names`                | `[]`                                 | Proper names to enforce correct capitalisation       |
| MD044 | `code_blocks`          | `true`                               | Check inside code blocks                             |
| MD044 | `html_elements`        | `true`                               | Check inside HTML elements                           |
| MD046 | `style`                | `consistent`                         | Code block style (`fenced`, `indented`, `consistent`) |
| MD048 | `style`                | `consistent`                         | Code fence style (`backtick`, `tilde`, `consistent`) |
| MD049 | `style`                | `consistent`                         | Emphasis style (`asterisk`, `underscore`, `consistent`) |
| MD050 | `style`                | `consistent`                         | Strong style (`asterisk`, `underscore`, `consistent`) |
| MD051 | `ignore_case`          | `false`                              | Ignore case when comparing link fragments            |
| MD051 | `ignored_pattern`      | `""`                                 | Regex pattern for fragments to ignore                |
| MD052 | `shortcut_syntax`      | `false`                              | Also check shortcut reference syntax `[label]`       |
| MD052 | `ignored_labels`       | `["x"]`                              | Reference labels to ignore                           |
| MD053 | `ignored_definitions`  | `["//"]`                             | Reference definitions to ignore                      |
| MD054 | `autolink`             | `true`                               | Allow autolinks `<url>`                              |
| MD054 | `collapsed`            | `true`                               | Allow collapsed reference links `[label][]`          |
| MD054 | `full`                 | `true`                               | Allow full reference links `[text][label]`           |
| MD054 | `inline`               | `true`                               | Allow inline links `[text](url)`                     |
| MD054 | `shortcut`             | `true`                               | Allow shortcut reference links `[label]`             |
| MD054 | `url_inline`           | `true`                               | Allow inline links whose text equals their URL       |
| MD055 | `style`                | `consistent`                         | Table pipe style (`leading_and_trailing`, `leading_only`, `trailing_only`, `no_leading_or_trailing`, `consistent`) |
| MD059 | `prohibited_texts`     | `["click here","here","link","more"]`| Generic link text phrases to prohibit               |
| MD060 | `style`                | `any`                                | Table column style (`aligned`, `compact`, `tight`, `any`) |
| MD060 | `aligned_delimiter`    | `false`                              | Require delimiter row to align with header           |

## Features

- Parses Markdown with the goldmark library for accurate, spec-compliant analysis.
- Reports violations with file, line, and column information.
- Auto-fix support (`--fix`) for a subset of rules.
- stdin support: lint with `goldmark-lint -` or format with `goldmark-lint --format`.
- Watch mode (`--watch`): re-lint files on every change, running until interrupted.
- Configuration file discovery: searches from the current directory up to the filesystem root.
- Supports `.markdownlint-cli2.yaml` and `.markdownlint.yaml` config formats.
- Config inheritance via `extends` for composable configuration.
- Per-glob rule overrides via `overrides` for fine-grained control.
- Inline disable comments (`markdownlint-disable`, `markdownlint-disable-next-line`, etc.).
- Multiple output formats: default text, JSON, JUnit XML, TAP, SARIF, and GitHub Actions annotations.
- Result caching via `.markdownlint-cli2-cache` to speed up repeated runs.
- Gitignore integration via the `gitignore` config key.
- `--list-rules` flag to inspect all rules with their enabled state and current options.
- `--summary` flag to print a per-rule violation count after linting.

## Comparison with markdownlint-cli2

goldmark-lint adds several features beyond what markdownlint-cli2 provides, but
markdownlint-cli2 also has capabilities that goldmark-lint does not:

| Feature | goldmark-lint | markdownlint-cli2 |
|---------|:---:|:---:|
| `--fail-on-warning` flag (exit code 1 for warnings) | ✅ | ❌ |
| SARIF output format | ✅ | ❌ |
| GitHub Actions annotation output format | ✅ | ❌ |
| `--list-rules` flag (inspect rules, options, and enabled state) | ✅ | ❌ |
| `--summary` flag (per-rule violation count breakdown) | ✅ | ❌ |
| Single self-contained binary (no Node.js required) | ✅ | ❌ |
| Embeddable Go library | ✅ | ❌ |
| Custom rule plugins | ❌ | ✅ |
| Shared configurations via npm packages | ❌ | ✅ |

### `--fail-on-warning`

By default, violations marked as `"warning"` severity in the config do not cause
a non-zero exit code. The `--fail-on-warning` flag changes this so that any
violation — regardless of severity — causes goldmark-lint to exit with code 1.
This is useful for stricter CI gates:

```sh
goldmark-lint --fail-on-warning '**/*.md'
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

```
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

## Rules

The table below lists all [markdownlint rules](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md).
Rules that are implemented in goldmark-lint are marked ✅. Rules marked 🔧 also support auto-fixing.

| Rule | Description | Status |
|------|-------------|--------|
| [MD001](https://github.com/DavidAnson/markdownlint/blob/main/doc/md001.md) | Heading levels should only increment by one level at a time | ✅ |
| [MD003](https://github.com/DavidAnson/markdownlint/blob/main/doc/md003.md) | Heading style | ✅ |
| [MD004](https://github.com/DavidAnson/markdownlint/blob/main/doc/md004.md) | Unordered list style | ✅ |
| [MD005](https://github.com/DavidAnson/markdownlint/blob/main/doc/md005.md) | Inconsistent indentation for list items at the same level | ✅ |
| [MD007](https://github.com/DavidAnson/markdownlint/blob/main/doc/md007.md) | Unordered list indentation | ✅ |
| [MD009](https://github.com/DavidAnson/markdownlint/blob/main/doc/md009.md) | Trailing spaces | ✅ 🔧 |
| [MD010](https://github.com/DavidAnson/markdownlint/blob/main/doc/md010.md) | Hard tabs | ✅ 🔧 |
| [MD011](https://github.com/DavidAnson/markdownlint/blob/main/doc/md011.md) | Reversed link syntax | ✅ 🔧 |
| [MD012](https://github.com/DavidAnson/markdownlint/blob/main/doc/md012.md) | Multiple consecutive blank lines | ✅ 🔧 |
| [MD013](https://github.com/DavidAnson/markdownlint/blob/main/doc/md013.md) | Line length | ✅ |
| [MD014](https://github.com/DavidAnson/markdownlint/blob/main/doc/md014.md) | Dollar signs used before commands without showing output | ✅ 🔧 |
| [MD018](https://github.com/DavidAnson/markdownlint/blob/main/doc/md018.md) | No space after hash on ATX style heading | ✅ 🔧 |
| [MD019](https://github.com/DavidAnson/markdownlint/blob/main/doc/md019.md) | Multiple spaces after hash on ATX style heading | ✅ 🔧 |
| [MD020](https://github.com/DavidAnson/markdownlint/blob/main/doc/md020.md) | No space inside hashes on closed ATX style heading | ✅ 🔧 |
| [MD021](https://github.com/DavidAnson/markdownlint/blob/main/doc/md021.md) | Multiple spaces inside hashes on closed ATX style heading | ✅ 🔧 |
| [MD022](https://github.com/DavidAnson/markdownlint/blob/main/doc/md022.md) | Headings should be surrounded by blank lines | ✅ |
| [MD023](https://github.com/DavidAnson/markdownlint/blob/main/doc/md023.md) | Headings must start at the beginning of the line | ✅ 🔧 |
| [MD024](https://github.com/DavidAnson/markdownlint/blob/main/doc/md024.md) | Multiple headings with the same content | ✅ |
| [MD025](https://github.com/DavidAnson/markdownlint/blob/main/doc/md025.md) | Multiple top-level headings in the same document | ✅ |
| [MD026](https://github.com/DavidAnson/markdownlint/blob/main/doc/md026.md) | Trailing punctuation in heading | ✅ 🔧 |
| [MD027](https://github.com/DavidAnson/markdownlint/blob/main/doc/md027.md) | Multiple spaces after blockquote symbol | ✅ 🔧 |
| [MD028](https://github.com/DavidAnson/markdownlint/blob/main/doc/md028.md) | Blank line inside blockquote | ✅ |
| [MD029](https://github.com/DavidAnson/markdownlint/blob/main/doc/md029.md) | Ordered list item prefix | ✅ 🔧 |
| [MD030](https://github.com/DavidAnson/markdownlint/blob/main/doc/md030.md) | Spaces after list markers | ✅ 🔧 |
| [MD031](https://github.com/DavidAnson/markdownlint/blob/main/doc/md031.md) | Fenced code blocks should be surrounded by blank lines | ✅ 🔧 |
| [MD032](https://github.com/DavidAnson/markdownlint/blob/main/doc/md032.md) | Lists should be surrounded by blank lines | ✅ 🔧 |
| [MD033](https://github.com/DavidAnson/markdownlint/blob/main/doc/md033.md) | Inline HTML | ✅ |
| [MD034](https://github.com/DavidAnson/markdownlint/blob/main/doc/md034.md) | Bare URL used | ✅ |
| [MD035](https://github.com/DavidAnson/markdownlint/blob/main/doc/md035.md) | Horizontal rule style | ✅ |
| [MD036](https://github.com/DavidAnson/markdownlint/blob/main/doc/md036.md) | Emphasis used instead of a heading | ✅ |
| [MD037](https://github.com/DavidAnson/markdownlint/blob/main/doc/md037.md) | Spaces inside emphasis markers | ✅ 🔧 |
| [MD038](https://github.com/DavidAnson/markdownlint/blob/main/doc/md038.md) | Spaces inside code span elements | ✅ 🔧 |
| [MD039](https://github.com/DavidAnson/markdownlint/blob/main/doc/md039.md) | Spaces inside link text | ✅ 🔧 |
| [MD040](https://github.com/DavidAnson/markdownlint/blob/main/doc/md040.md) | Fenced code blocks should have a language specified | ✅ |
| [MD041](https://github.com/DavidAnson/markdownlint/blob/main/doc/md041.md) | First line in a file should be a top-level heading | ✅ |
| [MD042](https://github.com/DavidAnson/markdownlint/blob/main/doc/md042.md) | No empty links | ✅ |
| [MD043](https://github.com/DavidAnson/markdownlint/blob/main/doc/md043.md) | Required heading structure | ✅ |
| [MD044](https://github.com/DavidAnson/markdownlint/blob/main/doc/md044.md) | Proper names should have the correct capitalization | ✅ 🔧 |
| [MD045](https://github.com/DavidAnson/markdownlint/blob/main/doc/md045.md) | Images should have alternate text (alt text) | ✅ |
| [MD046](https://github.com/DavidAnson/markdownlint/blob/main/doc/md046.md) | Code block style | ✅ |
| [MD047](https://github.com/DavidAnson/markdownlint/blob/main/doc/md047.md) | Files should end with a single newline character | ✅ 🔧 |
| [MD048](https://github.com/DavidAnson/markdownlint/blob/main/doc/md048.md) | Code fence style | ✅ 🔧 |
| [MD049](https://github.com/DavidAnson/markdownlint/blob/main/doc/md049.md) | Emphasis style | ✅ 🔧 |
| [MD050](https://github.com/DavidAnson/markdownlint/blob/main/doc/md050.md) | Strong style | ✅ 🔧 |
| [MD051](https://github.com/DavidAnson/markdownlint/blob/main/doc/md051.md) | Link fragments should be valid | ✅ |
| [MD052](https://github.com/DavidAnson/markdownlint/blob/main/doc/md052.md) | Reference links and images should use a label that is defined | ✅ |
| [MD053](https://github.com/DavidAnson/markdownlint/blob/main/doc/md053.md) | Link and image reference definitions should be needed | ✅ 🔧 |
| [MD054](https://github.com/DavidAnson/markdownlint/blob/main/doc/md054.md) | Link and image style | ✅ |
| [MD055](https://github.com/DavidAnson/markdownlint/blob/main/doc/md055.md) | Table pipe style | ✅ |
| [MD056](https://github.com/DavidAnson/markdownlint/blob/main/doc/md056.md) | Table column count | ✅ |
| [MD058](https://github.com/DavidAnson/markdownlint/blob/main/doc/md058.md) | Tables should be surrounded by blank lines | ✅ 🔧 |
| [MD059](https://github.com/DavidAnson/markdownlint/blob/main/doc/md059.md) | Link text should be descriptive | ✅ |
| [MD060](https://github.com/DavidAnson/markdownlint/blob/main/doc/md060.md) | Table column style | ✅ |

## License

[MIT](LICENSE)