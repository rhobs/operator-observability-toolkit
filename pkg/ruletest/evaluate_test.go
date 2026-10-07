package ruletest_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
)

func newTestRegistry() *operatorrules.Registry {
	registry := operatorrules.NewRegistry()

	Expect(registry.RegisterAlerts([]promv1.Rule{{
		Alert:       "ExampleAlertDown",
		Expr:        intstr.FromString("up == 0"),
		Labels:      map[string]string{"severity": "critical"},
		Annotations: map[string]string{"summary": "target is down"},
	}})).To(Succeed())

	return registry
}

// requirePromtool applies to this package's own specs the same rule the
// package imposes on consumers: a missing promtool fails, and only skips when
// RULETEST_SKIP_IF_MISSING opts in. Skipping by default would let CI report
// green having exercised none of the promtool path.
func requirePromtool() {
	GinkgoHelper()

	if ruletest.PromtoolAvailable() {
		return
	}

	if os.Getenv("RULETEST_SKIP_IF_MISSING") != "" {
		Skip("promtool not available and RULETEST_SKIP_IF_MISSING is set")
	}

	Fail("promtool not available: run `make test-rules`, or set PROMTOOL, " +
		"or set RULETEST_SKIP_IF_MISSING=1 to skip these specs")
}

var _ = Describe("Evaluate", func() {
	BeforeEach(requirePromtool)

	It("should pass for a correct scenario", func() {
		res, err := ruletest.Evaluate(newTestRegistry(), ruletest.Scenario{
			Name:        "target down fires the alert",
			InputSeries: []ruletest.Series{{Series: `up{job="x"}`, Values: "0+0x5"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  time.Minute,
				AlertName: "ExampleAlertDown",
				ExpAlerts: []ruletest.ExpectedAlert{{
					Labels:      map[string]string{"severity": "critical", "job": "x"},
					Annotations: map[string]string{"summary": "target is down"},
				}},
			}},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeFalse())
	})

	It("should report a failure for an incorrect expectation", func() {
		res, err := ruletest.Evaluate(newTestRegistry(), ruletest.Scenario{
			Name:        "wrong expectation",
			InputSeries: []ruletest.Series{{Series: `up{job="x"}`, Values: "0+0x5"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  time.Minute,
				AlertName: "ExampleAlertDown",
				ExpAlerts: []ruletest.ExpectedAlert{},
			}},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeTrue())
		Expect(res.Scenarios[0].Name).To(Equal("wrong expectation"))
		Expect(res.Scenarios[0].Failures[0]).ToNot(BeEmpty())
	})

	It("should return an error for an unknown alert name without running promtool", func() {
		_, err := ruletest.Evaluate(newTestRegistry(), ruletest.Scenario{
			Name:       "typo",
			AlertTests: []ruletest.AlertTest{{AlertName: "ExampleAlertTypo", EvalTime: time.Minute}},
		})

		Expect(err).To(MatchError(ContainSubstring(`unknown alert "ExampleAlertTypo"`)))
	})

	It("should return an error for an empty registry", func() {
		_, err := ruletest.Evaluate(operatorrules.NewRegistry(), ruletest.Scenario{
			Name:        "no rules",
			PromQLTests: []ruletest.PromQLTest{{Expr: "up", EvalTime: time.Minute, ExpectEmpty: true}},
		})

		Expect(err).To(HaveOccurred())
	})

	It("should report one result per scenario, named and ordered as submitted", func() {
		res, err := ruletest.Evaluate(newTestRegistry(),
			ruletest.Scenario{
				Name:        "first",
				InputSeries: []ruletest.Series{{Series: `up{job="x"}`, Values: "0+0x5"}},
				AlertTests: []ruletest.AlertTest{{
					EvalTime: time.Minute, AlertName: "ExampleAlertDown",
					ExpAlerts: []ruletest.ExpectedAlert{{
						Labels:      map[string]string{"severity": "critical", "job": "x"},
						Annotations: map[string]string{"summary": "target is down"},
					}},
				}},
			},
			ruletest.Scenario{
				Name:        "second",
				InputSeries: []ruletest.Series{{Series: `up{job="y"}`, Values: "1+0x5"}},
				AlertTests: []ruletest.AlertTest{{
					EvalTime: time.Minute, AlertName: "ExampleAlertDown",
					ExpAlerts: []ruletest.ExpectedAlert{},
				}},
			},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(res.Scenarios).To(HaveLen(2))
		Expect(res.Scenarios[0].Name).To(Equal("first"))
		Expect(res.Scenarios[1].Name).To(Equal("second"))
		Expect(res.Failed()).To(BeFalse())
	})

	// C2 regression: a scenario's verdict must not depend on a sibling's
	// position in the argument list. `wants10s` needs a 10s evaluation
	// interval; previously the file-level interval came from scenarios[0], so
	// prepending a 1m scenario silently under-evaluated it.
	It("should not let a sibling scenario change another scenario's verdict", func() {
		wants10s := ruletest.Scenario{
			Name:        "wants 10s",
			Interval:    10 * time.Second,
			InputSeries: []ruletest.Series{{Series: `up{job="x"}`, Values: "0+0x30"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  20 * time.Second,
				AlertName: "ExampleAlertDown",
				ExpAlerts: []ruletest.ExpectedAlert{{
					Labels:      map[string]string{"severity": "critical", "job": "x"},
					Annotations: map[string]string{"summary": "target is down"},
				}},
			}},
		}

		coarse := ruletest.Scenario{
			Name:        "coarse sibling",
			Interval:    time.Minute,
			InputSeries: []ruletest.Series{{Series: `up{job="y"}`, Values: "1+0x5"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  time.Minute,
				AlertName: "ExampleAlertDown",
				ExpAlerts: []ruletest.ExpectedAlert{},
			}},
		}

		alone, err := ruletest.Evaluate(newTestRegistry(), wants10s)
		Expect(err).ToNot(HaveOccurred())
		Expect(alone.Failed()).To(BeFalse(), "scenario should pass on its own")

		withSibling, err := ruletest.Evaluate(newTestRegistry(), coarse, wants10s)
		Expect(err).ToNot(HaveOccurred())
		Expect(withSibling.Failed()).To(BeFalse(), "prepending an unrelated scenario must not change the verdict")
	})
})
