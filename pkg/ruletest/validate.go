package ruletest

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// validateScenarios checks scenarios for mistakes that promtool would either
// accept silently or report in a way that hides the real cause.
func validateScenarios(scenarios []Scenario, knownAlerts []string) error {
	if len(scenarios) == 0 {
		return errors.New("no scenarios provided")
	}

	for i := range scenarios {
		if err := validateScenario(&scenarios[i], i, knownAlerts); err != nil {
			return err
		}
	}

	return nil
}

func validateScenario(s *Scenario, index int, knownAlerts []string) error {
	if s.Name == "" {
		return fmt.Errorf("scenario %d has no name", index)
	}

	if len(s.AlertTests) == 0 && len(s.PromQLTests) == 0 {
		return fmt.Errorf("scenario %q asserts nothing: it has no alert tests and no promql tests", s.Name)
	}

	if _, err := promDuration(s.Interval); err != nil {
		return fmt.Errorf("scenario %q has an unusable interval: %w", s.Name, err)
	}

	for _, at := range s.AlertTests {
		if err := validateAlertTest(s.Name, at, knownAlerts); err != nil {
			return err
		}
	}

	for _, pt := range s.PromQLTests {
		if err := validatePromQLTest(s.Name, pt); err != nil {
			return err
		}
	}

	return nil
}

func validatePromQLTest(scenarioName string, pt PromQLTest) error {
	if pt.Expr == "" {
		return fmt.Errorf("scenario %q has a promql test with no expression", scenarioName)
	}

	if _, err := promDuration(pt.EvalTime); err != nil {
		return fmt.Errorf("scenario %q promql test %q has an unusable eval time: %w", scenarioName, pt.Expr, err)
	}

	if pt.ExpectEmpty && len(pt.ExpSamples) > 0 {
		return fmt.Errorf("scenario %q promql test %q sets both ExpSamples and ExpectEmpty; set exactly one",
			scenarioName, pt.Expr)
	}

	if !pt.ExpectEmpty && len(pt.ExpSamples) == 0 {
		return fmt.Errorf("scenario %q promql test %q has no expected samples; "+
			"set ExpSamples, or set ExpectEmpty to assert the expression returns nothing",
			scenarioName, pt.Expr)
	}

	return nil
}

func validateAlertTest(scenarioName string, at AlertTest, knownAlerts []string) error {
	if at.AlertName == "" {
		return fmt.Errorf("scenario %q has an alert test with no alert name", scenarioName)
	}

	if !slices.Contains(knownAlerts, at.AlertName) {
		known := slices.Clone(knownAlerts)
		slices.Sort(known)

		return fmt.Errorf("scenario %q references unknown alert %q; registry contains: %s",
			scenarioName, at.AlertName, strings.Join(known, ", "))
	}

	if _, err := promDuration(at.EvalTime); err != nil {
		return fmt.Errorf("scenario %q alert test %q has an unusable eval time: %w", scenarioName, at.AlertName, err)
	}

	return nil
}
