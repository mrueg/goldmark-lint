package lint_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

// naiveLineAt is the straightforward O(n) definition of "which line is this
// byte offset on". Document.LineAt is a binary-search implementation of the
// same function; these tests assert the two agree.
func naiveLineAt(source []byte, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(source); i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// lineAtProbe is a Rule that captures the Document handed to Check so tests can
// exercise LineAt against a Document built by the linter itself.
type lineAtProbe struct{ doc *lint.Document }

func (p *lineAtProbe) ID() string          { return "PROBE" }
func (p *lineAtProbe) Description() string { return "captures the document" }
func (p *lineAtProbe) Check(doc *lint.Document) []lint.Violation {
	p.doc = doc
	return nil
}

func TestDocumentLineAt_MatchesNaiveScan(t *testing.T) {
	sources := []string{
		"",
		"\n",
		"one line no newline",
		"first\nsecond\nthird\n",
		"first\nsecond\nthird",
		"\n\n\nleading blanks\n",
		"crlf\r\nlines\r\nhere\r\n",
		"unicode ✅ ünïcödé\nsecond ✅ line\n",
		strings.Repeat("a line of text\n", 200),
	}

	for i, src := range sources {
		t.Run(fmt.Sprintf("source%d", i), func(t *testing.T) {
			probe := &lineAtProbe{}
			lint.NewLinter(probe).Lint([]byte(src))
			doc := probe.doc
			if doc == nil {
				t.Fatal("probe rule was never invoked")
			}
			// Probe every offset, plus a few out-of-range ones.
			for pos := -3; pos <= len(doc.Source)+3; pos++ {
				want := naiveLineAt(doc.Source, pos)
				if pos < 0 {
					want = 1
				}
				if got := doc.LineAt(pos); got != want {
					t.Fatalf("LineAt(%d) = %d, want %d (source %q)", pos, got, want, src)
				}
			}
		})
	}
}

func TestDocumentLineAt_LazyIndexForLiteralDocument(t *testing.T) {
	// Rules such as MD001 and MD034 build a Document as a plain struct literal
	// inside their Fix methods, so LineAt must work without the linter having
	// populated the index.
	source := []byte("# One\n\n### Three\n")
	doc := &lint.Document{
		Source: source,
		Lines:  strings.Split(string(source), "\n"),
	}
	for pos := 0; pos <= len(source); pos++ {
		if got, want := doc.LineAt(pos), naiveLineAt(source, pos); got != want {
			t.Fatalf("LineAt(%d) = %d, want %d", pos, got, want)
		}
	}
}

// syntheticDoc builds an n-line Markdown document with a realistic mix of
// headings, fenced code blocks and prose.
func syntheticDoc(n int) []byte {
	var b strings.Builder
	for i := 0; i < n; i++ {
		switch {
		case i%50 == 0:
			fmt.Fprintf(&b, "## Heading %d\n\n", i)
		case i%37 == 0:
			b.WriteString("```go\nx := 1\n```\n")
		default:
			fmt.Fprintf(&b, "Some text line number %d with words.\n", i)
		}
	}
	return []byte(b.String())
}

// BenchmarkLintBySize makes the cost of linting a single document visible as a
// function of its size. Before the line-offset index, resolving AST byte
// offsets rescanned the source from byte 0 on every lookup, so these numbers
// grew quadratically (2.5k lines: 1.1s, 20k lines: 38s). They should now grow
// roughly linearly; a sharp super-linear jump means offset lookups regressed.
func BenchmarkLintBySize(b *testing.B) {
	for _, lines := range []int{1000, 2000, 4000, 8000} {
		src := syntheticDoc(lines)
		linter := rules.NewDefaultLinter()
		b.Run(fmt.Sprintf("%dlines", lines), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				linter.Lint(src)
			}
		})
	}
}
