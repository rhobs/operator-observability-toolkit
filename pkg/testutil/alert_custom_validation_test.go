package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/rhobs/operator-observability-toolkit/pkg/testutil"
)

var _ = Describe("Custom Validators", func() {
	var linter *testutil.Linter

	BeforeEach(func() {
		linter = testutil.New()
	})

	Context("Custom Alert Validations", func() {
		It("should not return error if all custom validations were added and all exist", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity":                      "critical",
					"operator_health_impact":        "critical",
					"kubernetes_operator_part_of":   "example_part_of",
					"kubernetes_operator_component": "example_component",
				},
				Annotations: map[string]string{
					"summary":     "Example summary",
					"description": "Example description",
					"runbook_url": "example/runbook/url",
				},
			}
			linter.AddCustomAlertValidations(
				testutil.ValidateAlertNameLength,
				testutil.ValidateAlertHasDescriptionAnnotation,
				testutil.ValidateAlertRunbookURLAnnotation,
				testutil.ValidateAlertHealthImpactLabel,
				testutil.ValidateAlertPartOfAndComponentLabels,
			)
			problems := linter.LintAlert(alert)
			Expect(problems).To(BeEmpty())
		})

		It("should return error if alert name length custom validation was added and alert name is too long", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlertWithVeryLongNameExtendedToMeetRequiredLength",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary": "Example summary",
				},
			}
			linter.AddCustomAlertValidations(testutil.ValidateAlertNameLength)
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert name exceeds 50 characters"))
		})

		It("should return error if description custom validation was added and is missing", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary": "Example summary",
				},
			}
			linter.AddCustomAlertValidations(testutil.ValidateAlertHasDescriptionAnnotation)
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a description annotation"))
		})

		It("should return error if runbook_url custom validation was added and is missing", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary": "Example summary",
				},
			}
			linter.AddCustomAlertValidations(testutil.ValidateAlertRunbookURLAnnotation)
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a runbook_url annotation"))
		})

		It("should return error if operator_health_impact custom validation was added and is missing or invalid", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity":               "critical",
					"operator_health_impact": "invalid_operator_health_impact",
				},
				Annotations: map[string]string{
					"summary": "Example summary",
				},
			}
			linter.AddCustomAlertValidations(testutil.ValidateAlertHealthImpactLabel)
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(1))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a operator_health_impact label with value critical, warning, or none"))
		})

		It("should return error if operator_part_of and operator_component custom validation was added and both are missing", func() {
			alert := &promv1.Rule{
				Alert: "ExampleAlert",
				Expr:  intstr.FromString("sum(rate(http_requests_total[5m]))"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"summary": "Example summary",
				},
			}
			linter.AddCustomAlertValidations(testutil.ValidateAlertPartOfAndComponentLabels)
			problems := linter.LintAlert(alert)
			Expect(problems).To(HaveLen(2))
			Expect(problems[0].Description).To(ContainSubstring("alert must have a kubernetes_operator_part_of label"))
			Expect(problems[1].Description).To(ContainSubstring("alert must have a kubernetes_operator_component label"))
		})
	})
})
