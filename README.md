# goldmark-lint

A Markdown linter written in Go using the [goldmark](https://github.com/yuin/goldmark) parser.
It implements a subset of the rules from [markdownlint](https://github.com/DavidAnson/markdownlint) /
[markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2) and supports auto-fixing for
select rules.

## Installation

```sh
go install github.com/mrueg/goldmark-lint/cmd/goldmark-lint@latest
```

## Usage

```
goldmark-lint glob0 [glob1] [...] [globN] [--fix] [--help]

Glob expressions:
  *  matches any number of characters, but not /
  ?  matches a single character, but not /
  ** matches any number of characters, including /

Optional parameters:
  --fix   updates files to resolve fixable issues
  --help  writes this message to the console and exits without doing anything else

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
```

## Features

- Parses Markdown with the goldmark library for accurate, spec-compliant analysis.
- Reports violations with file, line, and column information.
- Auto-fix support (`--fix`) for a subset of rules.

## Rules

The table below lists all [markdownlint rules](https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md).
Rules that are implemented in goldmark-lint are marked ✅. Rules marked 🔧 also support auto-fixing.
Rules that are not yet implemented are marked ❌.

| Rule  | Description                                                              | Status |
|-------|--------------------------------------------------------------------------|--------|
| MD001 | Heading levels should only increment by one level at a time              | ✅     |
| MD003 | Heading style                                                            | ❌     |
| MD004 | Unordered list style                                                     | ❌     |
| MD005 | Inconsistent indentation for list items at the same level                | ❌     |
| MD007 | Unordered list indentation                                               | ❌     |
| MD009 | Trailing spaces                                                          | ✅ 🔧  |
| MD010 | Hard tabs                                                                | ✅ 🔧  |
| MD011 | Reversed link syntax                                                     | ❌     |
| MD012 | Multiple consecutive blank lines                                         | ✅ 🔧  |
| MD013 | Line length                                                              | ✅     |
| MD014 | Dollar signs used before commands without showing output                 | ❌     |
| MD018 | No space after hash on ATX style heading                                 | ❌     |
| MD019 | Multiple spaces after hash on ATX style heading                          | ❌     |
| MD020 | No space inside hashes on closed ATX style heading                       | ❌     |
| MD021 | Multiple spaces inside hashes on closed ATX style heading                | ❌     |
| MD022 | Headings should be surrounded by blank lines                             | ✅     |
| MD023 | Headings must start at the beginning of the line                         | ❌     |
| MD024 | Multiple headings with the same content                                  | ❌     |
| MD025 | Multiple top-level headings in the same document                         | ✅     |
| MD026 | Trailing punctuation in heading                                          | ❌     |
| MD027 | Multiple spaces after blockquote symbol                                  | ❌     |
| MD028 | Blank line inside blockquote                                             | ❌     |
| MD029 | Ordered list item prefix                                                 | ❌     |
| MD030 | Spaces after list markers                                                | ❌     |
| MD031 | Fenced code blocks should be surrounded by blank lines                   | ❌     |
| MD032 | Lists should be surrounded by blank lines                                | ❌     |
| MD033 | Inline HTML                                                              | ❌     |
| MD034 | Bare URL used                                                            | ❌     |
| MD035 | Horizontal rule style                                                    | ❌     |
| MD036 | Emphasis used instead of a heading                                       | ❌     |
| MD037 | Spaces inside emphasis markers                                           | ❌     |
| MD038 | Spaces inside code span elements                                         | ❌     |
| MD039 | Spaces inside link text                                                  | ❌     |
| MD040 | Fenced code blocks should have a language specified                      | ❌     |
| MD041 | First line in a file should be a top-level heading                       | ✅     |
| MD042 | No empty links                                                           | ❌     |
| MD043 | Required heading structure                                               | ❌     |
| MD044 | Proper names should have the correct capitalization                      | ❌     |
| MD045 | Images should have alternate text (alt text)                             | ❌     |
| MD046 | Code block style                                                         | ❌     |
| MD047 | Files should end with a single newline character                         | ✅ 🔧  |
| MD048 | Code fence style                                                         | ❌     |
| MD049 | Emphasis style                                                           | ❌     |
| MD050 | Strong style                                                             | ❌     |
| MD051 | Link fragments should be valid                                           | ❌     |
| MD052 | Reference links and images should use a label that is defined            | ❌     |
| MD053 | Link and image reference definitions should be needed                    | ❌     |
| MD054 | Link and image style                                                     | ❌     |
| MD055 | Table pipe style                                                         | ❌     |
| MD056 | Table column count                                                       | ❌     |
| MD058 | Tables should be surrounded by blank lines                               | ❌     |

## License

[MIT](LICENSE)