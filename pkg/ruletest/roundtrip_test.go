package ruletest_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest/matchers"
)

// These specs drive the generated YAML through a real promtool. The
// yaml_internal_test round-trips prove yaml.v3 is self-consistent; only
// promtool can prove it reads what we wrote.
var _ = Describe("promtool round-trip", func() {
	BeforeEach(requirePromtool)

	// Review Focus 1: Go template syntax in annotations.
	It("should preserve Go template syntax in annotations through promtool", func() {
		registry := operatorrules.NewRegistry()
		Expect(registry.RegisterAlerts([]promv1.Rule{{
			Alert:  "TemplatedAlert",
			Expr:   intstr.FromString("up == 0"),
			Labels: map[string]string{"severity": "warning"},
			Annotations: map[string]string{
				"summary": "{{ $labels.job }} is down, value {{ $value }}",
			},
		}})).To(Succeed())

		res, err := ruletest.Evaluate(registry, ruletest.Scenario{
			Name:        "templated annotation renders",
			InputSeries: []ruletest.Series{{Series: `up{job="api"}`, Values: "0+0x5"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  time.Minute,
				AlertName: "TemplatedAlert",
				ExpAlerts: []ruletest.ExpectedAlert{{
					Labels: map[string]string{"severity": "warning", "job": "api"},
					// promtool expands the template; if the braces had been
					// mangled in YAML this would not match.
					Annotations: map[string]string{"summary": "api is down, value 0"},
				}},
			}},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeFalse(), "failures: %v", res.Scenarios)
	})

	// Review Focus 5: colons in recording rule names, quotes in matchers.
	It("should preserve YAML-significant characters in expressions through promtool", func() {
		registry := operatorrules.NewRegistry()
		Expect(registry.RegisterRecordingRules([]operatorrules.RecordingRule{{
			MetricsOpts: operatormetrics.MetricOpts{Name: "job:guestbook:up_ratio"},
			MetricType:  operatormetrics.GaugeType,
			Expr:        intstr.FromString(`sum(up{namespace='guestbook', pod=~'guestbook-.*'}) or vector(0)`),
		}})).To(Succeed())

		res, err := ruletest.Evaluate(registry, ruletest.Scenario{
			Name: "colon-named recording rule evaluates",
			InputSeries: []ruletest.Series{
				{Series: `up{namespace="guestbook", pod="guestbook-a"}`, Values: "1+0x5"},
				{Series: `up{namespace="guestbook", pod="guestbook-b"}`, Values: "1+0x5"},
			},
			PromQLTests: []ruletest.PromQLTest{{
				Expr:     "job:guestbook:up_ratio",
				EvalTime: time.Minute,
				ExpSamples: []ruletest.Sample{{
					Labels: "job:guestbook:up_ratio",
					Value:  2,
				}},
			}},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeFalse(), "failures: %v", res.Scenarios)
	})

	// The documented Gomega form, which is the stated reason Evaluate returns
	// (Results, error) rather than reporting directly.
	It("should work with the documented Gomega matcher form", func() {
		Expect(ruletest.Evaluate(newTestRegistry(), ruletest.Scenario{
			Name:        "gomega form",
			InputSeries: []ruletest.Series{{Series: `up{job="x"}`, Values: "0+0x5"}},
			AlertTests: []ruletest.AlertTest{{
				EvalTime:  time.Minute,
				AlertName: "ExampleAlertDown",
				ExpAlerts: []ruletest.ExpectedAlert{{
					Labels:      map[string]string{"severity": "critical", "job": "x"},
					Annotations: map[string]string{"summary": "target is down"},
				}},
			}},
		})).To(matchers.Pass())
	})
})
