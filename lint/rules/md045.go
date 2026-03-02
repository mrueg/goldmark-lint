package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD045 checks that images have alternate text (alt text).
type MD045 struct{}

func (r MD045) ID() string          { return "MD045" }
func (r MD045) Aliases() []string   { return []string{"no-alt-text"} }
func (r MD045) Description() string { return "Images should have alternate text (alt text)" }

// md045ImgTagRE matches the opening of an HTML <img> tag (case-insensitive).
var md045ImgTagRE = regexp.MustCompile(`(?i)^<img\b`)

// md045AltAttrRE matches the presence of an alt attribute in an HTML tag.
var md045AltAttrRE = regexp.MustCompile(`(?i)\balt\s*=`)

// md045AriaHiddenTrueRE matches aria-hidden="true" in an HTML tag.
var md045AriaHiddenTrueRE = regexp.MustCompile(`(?i)\baria-hidden\s*=\s*["']?true["']?`)

// md045BlockImgTagRE matches an <img> start tag that may span multiple lines.
// It captures the img start tag (open until the closing >).
var md045BlockImgTagRE = regexp.MustCompile(`(?is)<img\b[^>]*>`)

func (r MD045) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Image:
			// Check if the image has non-empty alt text (any non-nil child node).
			if node.FirstChild() == nil {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    inlineNodeLine(node, doc.Source),
					Column:  1,
					Message: "Images should have alternate text (alt text)",
				})
			}

		case *ast.RawHTML:
			// Check inline HTML <img> tags that lack an alt attribute.
			if node.Segments == nil || node.Segments.Len() == 0 {
				return ast.WalkContinue, nil
			}
			var sb strings.Builder
			for i := 0; i < node.Segments.Len(); i++ {
				seg := node.Segments.At(i)
				sb.Write(doc.Source[seg.Start:seg.Stop])
			}
			tagText := sb.String()
			if !md045ImgTagRE.MatchString(tagText) {
				return ast.WalkContinue, nil
			}
			if !md045AltAttrRE.MatchString(tagText) && !md045AriaHiddenTrueRE.MatchString(tagText) {
				seg := node.Segments.At(0)
				lineNum := countLine(doc.Source, seg.Start)
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: "Images should have alternate text (alt text)",
				})
			}

		case *ast.HTMLBlock:
			// Check block-level HTML containing <img> tags without alt text.
			// Join all lines so that multi-line <img> tags are handled correctly.
			if node.Lines() == nil || node.Lines().Len() == 0 {
				return ast.WalkContinue, nil
			}
			firstSeg := node.Lines().At(0)
			lastSeg := node.Lines().At(node.Lines().Len() - 1)
			blockText := string(doc.Source[firstSeg.Start:lastSeg.Stop])
			// Find each <img> tag in the block and check for alt/aria-hidden.
			for _, match := range md045BlockImgTagRE.FindAllStringIndex(blockText, -1) {
				tag := blockText[match[0]:match[1]]
				if !md045AltAttrRE.MatchString(tag) && !md045AriaHiddenTrueRE.MatchString(tag) {
					// Report the line where the <img> starts.
					lineNum := countLine(doc.Source, firstSeg.Start+match[0])
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    lineNum,
						Column:  1,
						Message: "Images should have alternate text (alt text)",
					})
				}
			}
		}

		return ast.WalkContinue, nil
	})

	return violations
}
