package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
  "sort"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// ANSI color/style escape sequences.
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// isColorEnabled reports whether colored output should be used for w.
// Colors are enabled only when w is an interactive terminal and the NO_COLOR
// environment variable is not set (see https://no-color.org/).
func isColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// fileViolation groups violations for a single file.
type fileViolation struct {
	File       string
	Violations []lint.Violation
}

// ruleInfoURL returns the markdownlint documentation URL for a rule ID.
func ruleInfoURL(ruleID string) string {
	return "https://github.com/DavidAnson/markdownlint/blob/main/doc/" + strings.ToLower(ruleID) + ".md"
}

// formatDefault writes violations in the default text format to w.
// When w is an interactive terminal and NO_COLOR is not set, the output is
// colored: the file path is bold, the position is cyan, and the rule ID is
// red (errors) or yellow (warnings).
func formatDefault(violations []fileViolation, w io.Writer) {
	formatDefaultWithColor(violations, w, isColorEnabled(w))
}

// formatDefaultWithColor is the internal implementation of formatDefault.
// color controls whether ANSI escape sequences are emitted.
func formatDefaultWithColor(violations []fileViolation, w io.Writer, color bool) {
	for _, fv := range violations {
		for _, v := range fv.Violations {
			if color {
				ruleColor := colorRed
				if v.Severity == "warning" {
					ruleColor = colorYellow
				}
				_, _ = fmt.Fprintf(w, "%s%s%s:%s%d:%d%s %s%s%s %s\n",
					colorBold, fv.File, colorReset,
					colorCyan, v.Line, v.Column, colorReset,
					ruleColor, v.Rule, colorReset,
					v.Message)
			} else {
				_, _ = fmt.Fprintf(w, "%s:%d:%d %s %s\n", fv.File, v.Line, v.Column, v.Rule, v.Message)
			}
		}
	}
}

// jsonViolation is the JSON output structure for a single violation.
type jsonViolation struct {
	FileName        string   `json:"fileName"`
	LineNumber      int      `json:"lineNumber"`
	ColumnNumber    int      `json:"columnNumber"`
	RuleNames       []string `json:"ruleNames"`
	RuleDescription string   `json:"ruleDescription"`
	RuleInformation string   `json:"ruleInformation"`
	ErrorDetail     *string  `json:"errorDetail"`
	ErrorContext    *string  `json:"errorContext"`
	ErrorRange      *[2]int  `json:"errorRange"`
}

// formatJSON writes violations as a JSON array to w.
func formatJSON(violations []fileViolation, w io.Writer) {
	results := make([]jsonViolation, 0)
	for _, fv := range violations {
		for _, v := range fv.Violations {
			results = append(results, jsonViolation{
				FileName:        fv.File,
				LineNumber:      v.Line,
				ColumnNumber:    v.Column,
				RuleNames:       []string{v.Rule},
				RuleDescription: v.Message,
				RuleInformation: ruleInfoURL(v.Rule),
			})
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
}

// xmlTestSuites is the root element for JUnit XML output.
type xmlTestSuites struct {
	XMLName xml.Name   `xml:"testsuites"`
	Suites  []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	Name     string    `xml:"name,attr"`
	Tests    int       `xml:"tests,attr"`
	Failures int       `xml:"failures,attr"`
	Errors   int       `xml:"errors,attr"`
	Cases    []xmlCase `xml:"testcase"`
}

type xmlCase struct {
	Name      string       `xml:"name,attr"`
	ClassName string       `xml:"classname,attr"`
	Time      string       `xml:"time,attr"`
	Failures  []xmlFailure `xml:"failure,omitempty"`
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

// formatJUnit writes violations in JUnit XML format to w.
func formatJUnit(violations []fileViolation, w io.Writer) {
	var cases []xmlCase
	totalFiles := 0
	totalFailures := 0

	for _, fv := range violations {
		totalFiles++
		tc := xmlCase{
			Name:      fv.File,
			ClassName: "markdownlint",
			Time:      "0",
		}
		for _, v := range fv.Violations {
			totalFailures++
			msg := fmt.Sprintf("%d:%d %s %s", v.Line, v.Column, v.Rule, v.Message)
			tc.Failures = append(tc.Failures, xmlFailure{
				Message: msg,
				Type:    v.Rule,
				Text:    msg,
			})
		}
		cases = append(cases, tc)
	}

	suites := xmlTestSuites{
		Suites: []xmlSuite{
			{
				Name:     "markdownlint",
				Tests:    totalFiles,
				Failures: totalFailures,
				Errors:   0,
				Cases:    cases,
			},
		},
	}
	_, _ = fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(suites)
	_, _ = fmt.Fprintln(w)
}

// formatTAP writes violations in TAP (Test Anything Protocol) format to w.
func formatTAP(violations []fileViolation, w io.Writer) {
	type tapEntry struct {
		file string
		v    lint.Violation
	}
	var entries []tapEntry
	for _, fv := range violations {
		for _, v := range fv.Violations {
			entries = append(entries, tapEntry{fv.File, v})
		}
	}
	_, _ = fmt.Fprintf(w, "TAP version 13\n1..%d\n", len(entries))
	for i, e := range entries {
		_, _ = fmt.Fprintf(w, "not ok %d - %s:%d:%d %s %s\n", i+1, e.file, e.v.Line, e.v.Column, e.v.Rule, e.v.Message)
	}
}

// sarifLog is the top-level SARIF 2.1.0 log structure.
type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationUri string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string      `json:"id"`
	ShortDescription sarifText   `json:"shortDescription"`
	HelpUri          string      `json:"helpUri"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// sarifLevel maps a violation severity string to a SARIF level.
func sarifLevel(severity string) string {
	if severity == "warning" {
		return "warning"
	}
	return "error"
}

// formatSARIF writes violations in SARIF 2.1.0 format to w.
func formatSARIF(violations []fileViolation, w io.Writer) {
	// Collect unique rules in order of first appearance.
	seenRules := make(map[string]bool)
	var rules []sarifRule
	var results []sarifResult

	for _, fv := range violations {
		for _, v := range fv.Violations {
			if !seenRules[v.Rule] {
				seenRules[v.Rule] = true
				rules = append(rules, sarifRule{
					ID:               v.Rule,
					ShortDescription: sarifText{Text: v.Message},
					HelpUri:          ruleInfoURL(v.Rule),
				})
			}
			loc := sarifLocation{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI:       fv.File,
						URIBaseID: "%SRCROOT%",
					},
					Region: sarifRegion{
						StartLine:   v.Line,
						StartColumn: v.Column,
					},
				},
			}
			results = append(results, sarifResult{
				RuleID:    v.Rule,
				Level:     sarifLevel(v.Severity),
				Message:   sarifText{Text: v.Message},
				Locations: []sarifLocation{loc},
			})
		}
	}

	if results == nil {
		results = []sarifResult{}
	}
	if rules == nil {
		rules = []sarifRule{}
	}

	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "goldmark-lint",
						Version:        version,
						InformationUri: "https://github.com/mrueg/goldmark-lint",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(log)
}

// formatGitHubActions writes violations as GitHub Actions workflow commands to w.
// Errors use ::error and warnings use ::warning so that GitHub Actions displays
// them as native annotations in the PR diff view.
func formatGitHubActions(violations []fileViolation, w io.Writer) {
	for _, fv := range violations {
		for _, v := range fv.Violations {
			level := "error"
			if v.Severity == "warning" {
				level = "warning"
			}
			_, _ = fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d::%s %s\n",
				level, fv.File, v.Line, v.Column, v.Rule, v.Message)
		}
	}
}

// formatSummary writes a count-per-rule summary to w.
// Rules are sorted by count descending, then by rule ID ascending for ties.
func formatSummary(violations []fileViolation, w io.Writer) {
	counts := make(map[string]int)
	for _, fv := range violations {
		for _, v := range fv.Violations {
			counts[v.Rule]++
		}
	}
	if len(counts) == 0 {
		return
	}
	type ruleCount struct {
		rule  string
		count int
	}
	entries := make([]ruleCount, 0, len(counts))
	for rule, count := range counts {
		entries = append(entries, ruleCount{rule, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].rule < entries[j].rule
	})
	_, _ = fmt.Fprintln(w, "Summary:")
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "  %s: %d\n", e.rule, e.count)
	}
}

// outputFormatterSpec holds a format name and optional outfile for a single formatter run.
type outputFormatterSpec struct {
	format  string // "default", "json", "junit", "tap", or "sarif"
	outfile string
}

// formatterNameToFormat converts a markdownlint-cli2 formatter package name to
// the corresponding internal format string.
func formatterNameToFormat(name string) string {
	switch name {
	case "markdownlint-cli2-formatter-json":
		return "json"
	case "markdownlint-cli2-formatter-junit":
		return "junit"
	case "markdownlint-cli2-formatter-tap":
		return "tap"
	case "markdownlint-cli2-formatter-sarif":
		return "sarif"
	case "markdownlint-cli2-formatter-default":
		return "default"
	case "markdownlint-cli2-formatter-github":
		return "github"
	default:
		return name
	}
}

// parseOutputFormatters converts the raw outputFormatters config value (a slice of
// inner slices) into a slice of outputFormatterSpec values.
// Each inner slice has the formatter name as element 0 and an optional options
// map as element 1 (supporting the "outfile" key).
func parseOutputFormatters(raw []interface{}) []outputFormatterSpec {
	var specs []outputFormatterSpec
	for _, item := range raw {
		inner, ok := item.([]interface{})
		if !ok || len(inner) == 0 {
			continue
		}
		name, ok := inner[0].(string)
		if !ok {
			continue
		}
		spec := outputFormatterSpec{format: formatterNameToFormat(name)}
		if len(inner) > 1 {
			if opts, ok := inner[1].(map[string]interface{}); ok {
				if of, ok := opts["outfile"].(string); ok {
					spec.outfile = of
				}
			}
		}
		specs = append(specs, spec)
	}
	return specs
}
