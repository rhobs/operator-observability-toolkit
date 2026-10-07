package ruletest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

var _ = Describe("buildRuleFile", func() {
	var registry *operatorrules.Registry

	BeforeEach(func() {
		registry = operatorrules.NewRegistry()
	})

	It("should convert recording rules and alerts into a rule file", func() {
		Expect(registry.RegisterRecordingRules([]operatorrules.RecordingRule{{
			MetricsOpts: operatormetrics.MetricOpts{
				Name:        "guestbook_operator_number_of_pods",
				ConstLabels: map[string]string{"controller": "guestbook"},
			},
			MetricType: operatormetrics.GaugeType,
			Expr:       intstr.FromString("sum(up) or vector(0)"),
		}})).To(Succeed())

		forDuration := promv1.Duration("5m")

		Expect(registry.RegisterAlerts([]promv1.Rule{{
			Alert:       "GuestbookOperatorDown",
			Expr:        intstr.FromString("guestbook_operator_number_of_pods == 0"),
			For:         &forDuration,
			Labels:      map[string]string{"severity": "critical"},
			Annotations: map[string]string{"summary": "down"},
		}})).To(Succeed())

		f, err := buildRuleFile(registry)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Groups).To(HaveLen(2))

		Expect(f.Groups[0].Name).To(Equal("recordingRules.rules"))
		Expect(f.Groups[0].Rules).To(HaveLen(1))
		Expect(f.Groups[0].Rules[0].Record).To(Equal("guestbook_operator_number_of_pods"))
		Expect(f.Groups[0].Rules[0].Expr).To(Equal("sum(up) or vector(0)"))
		Expect(f.Groups[0].Rules[0].Labels).To(HaveKeyWithValue("controller", "guestbook"))
		Expect(f.Groups[0].Rules[0].Alert).To(BeEmpty())

		Expect(f.Groups[1].Name).To(Equal("alerts.rules"))
		Expect(f.Groups[1].Rules[0].Alert).To(Equal("GuestbookOperatorDown"))
		Expect(f.Groups[1].Rules[0].For).To(Equal("5m"))
		Expect(f.Groups[1].Rules[0].Record).To(BeEmpty())
	})

	It("should leave For empty when the alert has no For", func() {
		Expect(registry.RegisterAlerts([]promv1.Rule{{
			Alert: "NoFor",
			Expr:  intstr.FromString("up == 0"),
		}})).To(Succeed())

		f, err := buildRuleFile(registry)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Groups[0].Rules[0].For).To(BeEmpty())
	})

	// Review Focus 2: an empty registry must fail clearly.
	It("should return an error for an empty registry", func() {
		f, err := buildRuleFile(registry)
		Expect(err).To(HaveOccurred())
		Expect(f).To(BeNil())
		Expect(err).To(MatchError(ContainSubstring("no registered recording rule or alert")))
	})

	It("should report group and alert names", func() {
		Expect(registry.RegisterAlerts([]promv1.Rule{
			{Alert: "AlertOne", Expr: intstr.FromString("up == 0")},
			{Alert: "AlertTwo", Expr: intstr.FromString("up == 1")},
		})).To(Succeed())

		f, err := buildRuleFile(registry)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.groupNames()).To(Equal([]string{"alerts.rules"}))
		Expect(f.alertNames()).To(ConsistOf("AlertOne", "AlertTwo"))
	})
})
