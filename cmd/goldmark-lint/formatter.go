package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/mrueg/goldmark-lint/lint"
)

// fileViolation pairs a violation with the file it came from.
type fileViolation struct {
	File      string
	Violation lint.Violation
}

// writeText writes violations in the default text format to w.
func writeText(w io.Writer, violations []fileViolation) {
	for _, fv := range violations {
		fmt.Fprintf(w, "%s:%d:%d %s %s\n",
			fv.File, fv.Violation.Line, fv.Violation.Column,
			fv.Violation.Rule, fv.Violation.Message)
	}
}

// jsonViolation is the JSON representation of a violation.
type jsonViolation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// writeJSON writes violations as a JSON array to w.
func writeJSON(w io.Writer, violations []fileViolation) error {
	out := make([]jsonViolation, len(violations))
	for i, fv := range violations {
		out[i] = jsonViolation{
			File:     fv.File,
			Line:     fv.Violation.Line,
			Column:   fv.Violation.Column,
			Rule:     fv.Violation.Rule,
			Message:  fv.Violation.Message,
			Severity: fv.Violation.Severity,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// JUnit XML types

type xmlTestSuites struct {
	XMLName    xml.Name       `xml:"testsuites"`
	TestSuites []xmlTestSuite `xml:"testsuite"`
}

type xmlTestSuite struct {
	XMLName  xml.Name      `xml:"testsuite"`
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Errors   int           `xml:"errors,attr"`
	Cases    []xmlTestCase `xml:"testcase"`
}

type xmlTestCase struct {
	XMLName   xml.Name     `xml:"testcase"`
	Name      string       `xml:"name,attr"`
	Classname string       `xml:"classname,attr"`
	Failure   *xmlFailure  `xml:"failure,omitempty"`
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// writeJUnit writes violations as JUnit XML to w.
func writeJUnit(w io.Writer, violations []fileViolation) error {
	cases := make([]xmlTestCase, len(violations))
	failures := 0
	for i, fv := range violations {
		msg := fmt.Sprintf("%s: %s", fv.Violation.Rule, fv.Violation.Message)
		text := fmt.Sprintf("%s:%d:%d %s %s", fv.File, fv.Violation.Line, fv.Violation.Column, fv.Violation.Rule, fv.Violation.Message)
		cases[i] = xmlTestCase{
			Name:      fv.File,
			Classname: fv.Violation.Rule,
			Failure: &xmlFailure{
				Message: msg,
				Text:    text,
			},
		}
		failures++
	}
	suite := xmlTestSuites{
		TestSuites: []xmlTestSuite{
			{
				Name:     "goldmark-lint",
				Tests:    len(violations),
				Failures: failures,
				Errors:   0,
				Cases:    cases,
			},
		},
	}
	if _, err := fmt.Fprintln(w, xml.Header[:len(xml.Header)-1]); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suite); err != nil {
		return err
	}
	return enc.Flush()
}

// SARIF 2.1.0 types (minimal subset)

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
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string           `json:"id"`
	ShortDescription sarifMessage     `json:"shortDescription"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

// writeSARIF writes violations as SARIF 2.1.0 JSON to w.
func writeSARIF(w io.Writer, violations []fileViolation, ver string) error {
	// Collect unique rules
	rulesSeen := map[string]bool{}
	var sarifRules []sarifRule
	for _, fv := range violations {
		if !rulesSeen[fv.Violation.Rule] {
			rulesSeen[fv.Violation.Rule] = true
			sarifRules = append(sarifRules, sarifRule{
				ID:               fv.Violation.Rule,
				ShortDescription: sarifMessage{Text: fv.Violation.Rule},
			})
		}
	}

	results := make([]sarifResult, len(violations))
	for i, fv := range violations {
		level := "error"
		if fv.Violation.Severity == "warning" {
			level = "warning"
		}
		results[i] = sarifResult{
			RuleID:  fv.Violation.Rule,
			Level:   level,
			Message: sarifMessage{Text: fv.Violation.Message},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: fv.File},
						Region: sarifRegion{
							StartLine:   fv.Violation.Line,
							StartColumn: fv.Violation.Column,
						},
					},
				},
			},
		}
	}

	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    "goldmark-lint",
						Version: ver,
						Rules:   sarifRules,
					},
				},
				Results: results,
			},
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
