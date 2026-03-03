package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD033 checks for inline HTML in Markdown documents.
type MD033 struct {
	// AllowedElements is a list of HTML element names that are permitted.
	AllowedElements []string `json:"allowed_elements"`
	// TableAllowedElements is a list of HTML element names that are permitted
	// inside GFM table cells (in addition to AllowedElements).
	TableAllowedElements []string `json:"table_allowed_elements"`
}

func (r MD033) ID() string          { return "MD033" }
func (r MD033) Aliases() []string   { return []string{"no-inline-html"} }
func (r MD033) Description() string { return "Inline HTML" }

// htmlOpenTagRE matches opening HTML tags (not closing tags like </div>).
// Used to scan HTML block content for individual opening tags.
var htmlOpenTagRE = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9-]*)(?:\s[^>]*)?/?>`)

// htmlOpenTagStartRE matches the START of an opening HTML tag whose attributes
// span multiple lines (i.e., the closing ">" is not on the same line).
// Example: `<img src="..." (with no ">" on this line)
var htmlOpenTagStartRE = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9-]*)\s[^>]*$`)

func (r MD033) isAllowed(tag string) bool {
	for _, allowed := range r.AllowedElements {
		if allowed == tag {
			return true
		}
	}
	return false
}

func (r MD033) isTableAllowed(tag string) bool {
	for _, allowed := range r.TableAllowedElements {
		if allowed == tag {
			return true
		}
	}
	return false
}

func (r MD033) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	// Build a line-based table mask for table_allowed_elements support.
	var tableMask []bool
	if len(r.TableAllowedElements) > 0 {
		codeMask := fencedCodeBlockMask(doc.Lines)
		tableMask = make([]bool, len(doc.Lines))
		for _, tbl := range findTables(doc.Lines, codeMask) {
			for i := tbl[0]; i <= tbl[1]; i++ {
				tableMask[i] = true
			}
		}
	}

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.HTMLBlock:
			// Skip HTML comment blocks (type 2: <!-- ... -->).
			// markdownlint does not flag HTML comments as inline HTML.
			if node.HTMLBlockType == ast.HTMLBlockType2 {
				return ast.WalkContinue, nil
			}
			// Scan each line of the HTML block for opening tags.
			// Markdownlint reports each opening tag individually on its source line
			// and skips closing tags. This matches the behaviour of reporting
			// each <dt>, <dd>, etc. separately while not flagging </details>.
			if node.Lines() != nil {
				for i := 0; i < node.Lines().Len(); i++ {
					seg := node.Lines().At(i)
					lineNum := countLine(doc.Source, seg.Start)
					lineContent := strings.TrimRight(string(seg.Value(doc.Source)), "\r\n")
					// Skip HTML comment lines.
					if strings.HasPrefix(strings.TrimSpace(lineContent), "<!--") {
						continue
					}
					for _, m := range htmlOpenTagRE.FindAllStringSubmatch(lineContent, -1) {
						tag := strings.ToLower(m[1])
						if r.isAllowed(tag) {
							continue
						}
						if len(r.TableAllowedElements) > 0 && tableMask != nil {
							lineIdx := lineNum - 1
							if lineIdx >= 0 && lineIdx < len(tableMask) && tableMask[lineIdx] && r.isTableAllowed(tag) {
								continue
							}
						}
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    lineNum,
							Column:  1,
							Message: fmt.Sprintf("Inline HTML [Element: %s]", tag),
						})
					}
					// Also detect opening tags whose attributes span multiple lines
					// (i.e., the closing ">" appears on a later line).  The per-line
					// regex above requires ">" on the same line so it misses these.
					// htmlOpenTagStartRE uses [^>]*$ which only matches when no ">"
					// appears between the tag name and end-of-line, so it naturally
					// skips complete tags and avoids double-reporting.
					for _, m := range htmlOpenTagStartRE.FindAllStringSubmatch(lineContent, -1) {
						tag := strings.ToLower(m[1])
						if r.isAllowed(tag) {
							continue
						}
						if len(r.TableAllowedElements) > 0 && tableMask != nil {
							lineIdx := lineNum - 1
							if lineIdx >= 0 && lineIdx < len(tableMask) && tableMask[lineIdx] && r.isTableAllowed(tag) {
								continue
							}
						}
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    lineNum,
							Column:  1,
							Message: fmt.Sprintf("Inline HTML [Element: %s]", tag),
						})
					}
				}
			}

		case *ast.RawHTML:
			lineNum := 1
			if node.Segments != nil && node.Segments.Len() > 0 {
				seg := node.Segments.At(0)
				lineNum = countLine(doc.Source, seg.Start)
			}
			// Skip closing tags (e.g. </b>) — only opening tags are reported,
			// matching markdownlint-cli2 behaviour.
			if isClosingRawHTML(node, doc.Source) {
				return ast.WalkContinue, nil
			}
			// Extract the tag name for allowed-element checking.
			tag := rawHTMLTagName(node, doc.Source)
			if r.isAllowed(tag) {
				return ast.WalkContinue, nil
			}
			// Check table_allowed_elements when inside a table line.
			if len(r.TableAllowedElements) > 0 && tableMask != nil {
				lineIdx := lineNum - 1
				if lineIdx >= 0 && lineIdx < len(tableMask) && tableMask[lineIdx] && r.isTableAllowed(tag) {
					return ast.WalkContinue, nil
				}
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: fmt.Sprintf("Inline HTML [Element: %s]", tag),
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}

// rawHTMLTagName extracts the tag name from a RawHTML node (e.g. "br" from "<br/>").
func rawHTMLTagName(n *ast.RawHTML, source []byte) string {
	if n.Segments == nil || n.Segments.Len() == 0 {
		return "unknown"
	}
	seg := n.Segments.At(0)
	raw := string(seg.Value(source))
	// Strip leading '<' and optional '/'.
	i := 0
	if i < len(raw) && raw[i] == '<' {
		i++
	}
	if i < len(raw) && raw[i] == '/' {
		i++
	}
	// Read the tag name (alphanumeric + hyphen).
	start := i
	for i < len(raw) && (raw[i] == '-' || (raw[i] >= 'a' && raw[i] <= 'z') || (raw[i] >= 'A' && raw[i] <= 'Z') || (raw[i] >= '0' && raw[i] <= '9')) {
		i++
	}
	if i == start {
		return "unknown"
	}
	return raw[start:i]
}

// isClosingRawHTML reports whether a RawHTML node represents a closing tag
// (i.e., starts with "</"), such as </b> or </div>.
func isClosingRawHTML(n *ast.RawHTML, source []byte) bool {
	if n.Segments == nil || n.Segments.Len() == 0 {
		return false
	}
	seg := n.Segments.At(0)
	raw := seg.Value(source)
	return len(raw) >= 2 && raw[0] == '<' && raw[1] == '/'
}
