package ruletest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

// PromtoolAvailable reports whether a promtool binary can be located. It is
// intended for tests that need to skip when promtool is not installed.
func PromtoolAvailable() bool {
	_, err := resolvePromtool()

	return err == nil
}

// Evaluate runs scenarios against the rules registered in reg and returns the
// outcome.
//
// The returned error is non-nil only when the harness itself could not run:
// invalid scenarios, an empty registry, a missing promtool, or rules promtool
// cannot load. Failed assertions are reported in Results, not as an error.
func Evaluate(reg *operatorrules.Registry, scenarios ...Scenario) (Results, error) {
	rf, err := buildRuleFile(reg)
	if err != nil {
		return Results{}, err
	}

	if err := validateScenarios(scenarios, rf.alertNames()); err != nil {
		return Results{}, err
	}

	bin, err := resolvePromtool()
	if err != nil {
		return Results{}, err
	}

	dir, cleanup, err := artifactDir()
	if err != nil {
		return Results{}, err
	}
	defer cleanup()

	var results Results

	for i, s := range scenarios {
		result, err := evaluateScenario(bin, dir, i, rf, s)
		if err != nil {
			return Results{}, err
		}

		results.Scenarios = append(results.Scenarios, result)
	}

	return results, nil
}

// evaluateScenario runs a single scenario in its own directory and promtool
// invocation. One scenario per invocation keeps promtool's file-level
// evaluation_interval matched to the scenario that asked for it, and makes the
// expected report shape exact: one testcase, named after the scenario.
func evaluateScenario(bin, dir string, index int, rf *ruleFile, s Scenario) (ScenarioResult, error) {
	sub := filepath.Join(dir, fmt.Sprintf("scenario-%d", index))
	if err := os.MkdirAll(sub, 0o700); err != nil {
		return ScenarioResult{}, fmt.Errorf("creating scenario dir: %w", err)
	}

	testPath, err := writeFiles(sub, rf, s)
	if err != nil {
		return ScenarioResult{}, err
	}

	results, err := runPromtool(bin, sub, testPath)
	if err != nil {
		return ScenarioResult{}, err
	}

	// Guard against a report that does not describe the run we asked for. An
	// unrecognised schema parses into zero testcases, which would otherwise
	// read as "nothing failed".
	if len(results.Scenarios) != 1 {
		return ScenarioResult{}, fmt.Errorf(
			"promtool reported %d testcases for scenario %q, expected exactly 1; "+
				"this usually means its output format is not the one this package expects",
			len(results.Scenarios), s.Name)
	}

	if got := results.Scenarios[0].Name; got != s.Name {
		return ScenarioResult{}, fmt.Errorf("promtool reported testcase %q but scenario %q was submitted", got, s.Name)
	}

	return results.Scenarios[0], nil
}

// artifactDir returns a directory for the generated YAML. When
// RULETEST_KEEP_ARTIFACTS is set the directory is kept and its path reported so
// the generated files can be inspected and replayed by hand.
func artifactDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "ruletest-")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}

	if os.Getenv(envKeepArtifacts) != "" {
		fmt.Fprintf(os.Stderr, "ruletest: keeping generated files in %s\n", dir)

		return dir, func() {}, nil
	}

	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
