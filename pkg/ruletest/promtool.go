package ruletest

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	envPromtool      = "PROMTOOL"
	envSkipIfMissing = "RULETEST_SKIP_IF_MISSING"
	envKeepArtifacts = "RULETEST_KEEP_ARTIFACTS"

	junitFileName = "junit.xml"
)

// ErrPromtoolNotFound is returned when no promtool binary can be located.
var ErrPromtoolNotFound = errors.New("promtool not found")

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name  string          `xml:"name,attr"`
	Cases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name     string        `xml:"name,attr"`
	Failures []string      `xml:"failure"`
	Errors   []string      `xml:"error"`
	Skipped  *junitSkipped `xml:"skipped"`
}

type junitSkipped struct{}

// resolvePromtool locates the promtool binary, preferring the PROMTOOL
// environment variable over PATH.
func resolvePromtool() (string, error) {
	if p := os.Getenv(envPromtool); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("%s is set to %q but it cannot be used: %w", envPromtool, p, err)
		}

		return p, nil
	}

	p, err := exec.LookPath("promtool")
	if err != nil {
		return "", ErrPromtoolNotFound
	}

	return p, nil
}

// parseJUnit converts promtool's JUnit report into Results. An <error> element
// indicates promtool could not read the generated files at all, which is a bug
// in this package rather than a failed assertion, so it is returned as an error.
func parseJUnit(data []byte) (Results, error) {
	var parsed junitTestSuites
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return Results{}, fmt.Errorf("parsing promtool report: %w", err)
	}

	var results Results

	for _, suite := range parsed.Suites {
		for _, c := range suite.Cases {
			if len(c.Errors) > 0 {
				return Results{}, fmt.Errorf("promtool could not run the generated test file: %s",
					strings.TrimSpace(strings.Join(c.Errors, "; ")))
			}

			// We never ask promtool to skip anything, so a skipped testcase
			// means the report does not describe the run we requested.
			// Counting it as a pass would be a silent green.
			if c.Skipped != nil {
				return Results{}, fmt.Errorf("promtool reported testcase %q as skipped, which this package never requests", c.Name)
			}

			results.Scenarios = append(results.Scenarios, ScenarioResult{
				Name:     c.Name,
				Failures: trimAll(c.Failures),
			})
		}
	}

	return results, nil
}

func trimAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, strings.TrimSpace(s))
	}

	return out
}

// runPromtool invokes promtool against testFilePath and returns the parsed
// results. A non-zero exit status is expected when assertions fail and is not
// treated as an error on its own.
func runPromtool(bin, dir, testFilePath string) (Results, error) {
	junitPath := filepath.Join(dir, junitFileName)

	cmd := exec.Command(bin, "test", "rules", "--junit="+junitPath, filepath.Base(testFilePath))
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	data, readErr := os.ReadFile(junitPath)
	if readErr != nil {
		return Results{}, fmt.Errorf("promtool wrote no test report (%v): %s%s",
			runErr, stdout.String(), stderr.String())
	}

	results, err := parseJUnit(data)
	if err != nil {
		return Results{}, err
	}

	// promtool exits non-zero whenever any test fails, so a non-zero status
	// with failures in the report is the normal path. A non-zero status with
	// nothing to show for it means we did not understand the report: treat it
	// as a harness error rather than reporting a silent pass.
	if runErr != nil && !results.Failed() {
		return Results{}, fmt.Errorf("promtool exited with %v but its report lists no failures; "+
			"this usually means its output format is not the one this package expects: %s%s",
			runErr, stdout.String(), stderr.String())
	}

	return results, nil
}
