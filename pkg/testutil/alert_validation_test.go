package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/rhobs/operator-observability/pkg/testutil"
)

var _ = Describe("Default Validators", func() {
	var linter *testutil.Linter

	BeforeEach(func() {
		linter = testutil.New()
	})

	Context("Alert Validation", func() {
		It("should validate alert with valid input", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary":     "Example summary",
					"description": "Example description",
				},
			}
			problems := linter.LintAlert(alert)
			Expect(problems).To(BeEmpty())
		})

		It("should return error if alert name is not in PascalCase format", func() {
			alert := &promv1.Rule{
				Alert: "example_alert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary":     "Example summary",
					"description": "Example description",
				},
			}
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a name in PascalCase format"))
		})

		It("should return error if alert expression is empty", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString(""),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary":     "Example summary",
					"description": "Example description",
				},
			}
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have an expression"))
		})

		It("should return error if severity label is missing or invalid", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "invalid_severity",
				},
				Annotations: map[string]string{
					"summary":     "Example summary",
					"description": "Example description",
				},
			}
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a severity label with value critical, warning, or info"))
		})

		It("should return error if summary annotation is missing", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"description": "Example description",
				},
			}
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a summary annotation"))
		})
	})
})
