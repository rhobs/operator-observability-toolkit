// Package matchers provides Gomega matchers for ruletest results.
package matchers

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/types"

	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
)

// Pass succeeds when every scenario in a ruletest.Results passed.
//
// It is intended to be used with ruletest.Evaluate:
//
//	Expect(ruletest.Evaluate(registry, scenario)).To(matchers.Pass())
func Pass() types.GomegaMatcher {
	return &passMatcher{}
}

type passMatcher struct{}

func (m *passMatcher) Match(actual any) (bool, error) {
	results, ok := actual.(ruletest.Results)
	if !ok {
		return false, fmt.Errorf("pass matcher expects a ruletest.Results, got %T", actual)
	}

	return !results.Failed(), nil
}

func (m *passMatcher) FailureMessage(actual any) string {
	results, ok := actual.(ruletest.Results)
	if !ok {
		return fmt.Sprintf("Expected a ruletest.Results, got %T", actual)
	}

	var b strings.Builder
	b.WriteString("Expected all rule test scenarios to pass, but these failed:\n")

	for _, s := range results.Scenarios {
		if len(s.Failures) == 0 {
			continue
		}

		fmt.Fprintf(&b, "\n%s:\n", s.Name)

		for _, f := range s.Failures {
			fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(f, "\n", "\n  "))
		}
	}

	return b.String()
}

func (m *passMatcher) NegatedFailureMessage(_ any) string {
	return "Expected at least one rule test scenario to fail, but all passed"
}
