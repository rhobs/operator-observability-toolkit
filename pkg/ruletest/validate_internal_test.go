package ruletest

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("validateScenarios", func() {
	known := []string{"GuestbookOperatorDown", "GuestbookOperatorNotReady"}

	validScenario := func() Scenario {
		return Scenario{
			Name: "operator is down",
			AlertTests: []AlertTest{{
				EvalTime:  5 * time.Minute,
				AlertName: "GuestbookOperatorDown",
			}},
		}
	}

	It("should accept a valid scenario", func() {
		Expect(validateScenarios([]Scenario{validScenario()}, known)).To(Succeed())
	})

	It("should reject an empty scenario list", func() {
		err := validateScenarios(nil, known)
		Expect(err).To(MatchError(ContainSubstring("no scenarios provided")))
	})

	It("should reject a scenario with no name", func() {
		s := validScenario()
		s.Name = ""
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("has no name")))
	})

	// Review Focus 3: a scenario asserting nothing would pass in promtool.
	It("should reject a scenario with no alert tests and no promql tests", func() {
		s := validScenario()
		s.AlertTests = nil
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("asserts nothing")))
	})

	It("should reject an unknown alert name and list the known ones", func() {
		s := validScenario()
		s.AlertTests[0].AlertName = "GuestbookOperatorTypo"
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring(`unknown alert "GuestbookOperatorTypo"`)))
		Expect(err).To(MatchError(ContainSubstring("GuestbookOperatorDown")))
		Expect(err).To(MatchError(ContainSubstring("GuestbookOperatorNotReady")))
	})

	It("should reject a negative eval time", func() {
		s := validScenario()
		s.AlertTests[0].EvalTime = -time.Second
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("eval time")))
		Expect(err).To(MatchError(ContainSubstring("negative")))
	})

	It("should reject a promql test with no expression", func() {
		s := validScenario()
		s.PromQLTests = []PromQLTest{{EvalTime: time.Minute}}
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("no expression")))
	})

	// A PromQLTest with no ExpSamples asserts "this returns nothing", which a
	// typo'd metric name also satisfies. Require the author to say which they
	// mean, so a typo cannot pass silently.
	It("should reject a promql test that expects nothing without saying so", func() {
		s := validScenario()
		s.PromQLTests = []PromQLTest{{Expr: "totally_made_up_metric", EvalTime: time.Minute}}
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("ExpectEmpty")))
	})

	It("should accept a promql test that expects nothing explicitly", func() {
		s := validScenario()
		s.PromQLTests = []PromQLTest{{Expr: "absent_metric", EvalTime: time.Minute, ExpectEmpty: true}}
		Expect(validateScenarios([]Scenario{s}, known)).To(Succeed())
	})

	It("should reject a promql test that sets ExpectEmpty alongside samples", func() {
		s := validScenario()
		s.PromQLTests = []PromQLTest{{
			Expr:        "some_metric",
			EvalTime:    time.Minute,
			ExpectEmpty: true,
			ExpSamples:  []Sample{{Labels: "some_metric", Value: 1}},
		}}
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("ExpectEmpty")))
	})

	// Prometheus durations admit no decimal point; catch it here, naming the
	// scenario, rather than letting promtool fail on YAML the user never sees.
	It("should reject an eval time Prometheus cannot represent", func() {
		s := validScenario()
		s.AlertTests[0].EvalTime = 1500 * time.Microsecond
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("operator is down")))
		Expect(err).To(MatchError(ContainSubstring("millisecond")))
	})

	It("should reject an interval Prometheus cannot represent", func() {
		s := validScenario()
		s.Interval = 1500 * time.Microsecond
		err := validateScenarios([]Scenario{s}, known)
		Expect(err).To(MatchError(ContainSubstring("millisecond")))
	})

	It("should accept a zero eval time", func() {
		s := validScenario()
		s.AlertTests[0].EvalTime = 0
		Expect(validateScenarios([]Scenario{s}, known)).To(Succeed())
	})
})

var _ = Describe("Results", func() {
	It("should report failure when any scenario has failures", func() {
		r := Results{Scenarios: []ScenarioResult{
			{Name: "a"},
			{Name: "b", Failures: []string{"boom"}},
		}}
		Expect(r.Failed()).To(BeTrue())
	})

	It("should report success when no scenario has failures", func() {
		r := Results{Scenarios: []ScenarioResult{{Name: "a"}, {Name: "b"}}}
		Expect(r.Failed()).To(BeFalse())
	})
})
