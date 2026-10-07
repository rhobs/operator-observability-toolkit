package matchers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest/matchers"
)

var _ = Describe("Pass", func() {
	It("should match results with no failures", func() {
		res := ruletest.Results{Scenarios: []ruletest.ScenarioResult{{Name: "a"}}}
		Expect(res).To(matchers.Pass())
	})

	It("should not match results with failures", func() {
		res := ruletest.Results{Scenarios: []ruletest.ScenarioResult{
			{Name: "a", Failures: []string{"exp: 1 got: 0"}},
		}}
		Expect(res).ToNot(matchers.Pass())
	})

	It("should name the failing scenario in the failure message", func() {
		res := ruletest.Results{Scenarios: []ruletest.ScenarioResult{
			{Name: "operator is down", Failures: []string{"exp: 1 got: 0"}},
		}}

		msg := matchers.Pass().FailureMessage(res)
		Expect(msg).To(ContainSubstring("operator is down"))
		Expect(msg).To(ContainSubstring("exp: 1 got: 0"))
	})

	It("should error for a non-Results actual", func() {
		ok, err := matchers.Pass().Match("not results")
		Expect(ok).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})
})
