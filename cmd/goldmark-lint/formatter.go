package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

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
func formatDefault(violations []fileViolation, w io.Writer) {
	for _, fv := range violations {
		for _, v := range fv.Violations {
			fmt.Fprintf(w, "%s:%d:%d %s %s\n", fv.File, v.Line, v.Column, v.Rule, v.Message)
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
	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(suites)
	fmt.Fprintln(w)
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
	fmt.Fprintf(w, "TAP version 13\n1..%d\n", len(entries))
	for i, e := range entries {
		fmt.Fprintf(w, "not ok %d - %s:%d:%d %s %s\n", i+1, e.file, e.v.Line, e.v.Column, e.v.Rule, e.v.Message)
	}
}

// outputFormatterSpec holds a format name and optional outfile for a single formatter run.
type outputFormatterSpec struct {
	format  string // "default", "json", "junit", or "tap"
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
	case "markdownlint-cli2-formatter-default":
		return "default"
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
