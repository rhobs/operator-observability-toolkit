package operatorrules_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

var _ = Describe("PrometheusRules", func() {
	Context("Building resource", func() {
		var or *operatorrules.Registry

		var recordingRules = []operatorrules.RecordingRule{
			{
				MetricsOpts: operatormetrics.MetricOpts{
					Name:        "number_of_pods",
					Help:        "Number of guestbook operator pods in the cluster",
					ConstLabels: map[string]string{"controller": "guestbook"},
				},
				MetricType: operatormetrics.GaugeType,
				Expr:       intstr.FromString("sum(up{namespace='default', pod=~'guestbook-operator-.*'}) or vector(0)"),
			},
			{
				MetricsOpts: operatormetrics.MetricOpts{
					Name:        "a_test_counter",
					Help:        "A test counter",
					ConstLabels: map[string]string{"controller": "guestbook"},
				},
				MetricType: operatormetrics.CounterType,
				Expr:       intstr.FromString("sum(rate(http_requests_total[5m]))"),
			},
			{
				MetricsOpts: operatormetrics.MetricOpts{
					Name:        "a_test_counter",
					Help:        "another rule to check the order",
					ConstLabels: map[string]string{"controller": "guestbook"},
				},
				MetricType: operatormetrics.CounterType,
				Expr:       intstr.FromString("sum(rate(http_requests_total[20m]))"),
			},
		}

		var alerts = []promv1.Rule{
			{
				Alert: "GuestbookOperatorDown",
				Expr:  intstr.FromString("number_of_pods == 0"),
				Annotations: map[string]string{
					"summary":     "Guestbook operator is down",
					"description": "Guestbook operator is down for more than 5 minutes.",
				},
				Labels: map[string]string{
					"severity": "critical",
				},
			},
			{
				Alert: "ATestAlert",
				Expr:  intstr.FromString("test_counter > 10"),
				Annotations: map[string]string{
					"summary":     "Test alert",
					"description": "Test alert description",
				},
				Labels: map[string]string{
					"severity": "warning",
				},
			},
		}

		BeforeEach(func() {
			or = operatorrules.NewRegistry()

			err := or.RegisterRecordingRules(recordingRules)
			Expect(err).To(Not(HaveOccurred()))

			err = or.RegisterAlerts(alerts)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should build PrometheusRule with valid input", func() {
			rules, err := or.BuildPrometheusRule(
				"guestbook-operator-prometheus-rules",
				"default",
				map[string]string{"app": "guestbook-operator"},
			)

			Expect(err).To(BeNil())
			Expect(rules).NotTo(BeNil())
			Expect(rules.Name).To(Equal("guestbook-operator-prometheus-rules"))
			Expect(rules.Namespace).To(Equal("default"))

			Expect(rules.Spec.Groups).To(HaveLen(2))

			Expect(rules.Spec.Groups[0].Name).To(Equal("recordingRules.rules"))
			Expect(rules.Spec.Groups[0].Rules).To(HaveLen(3))
			Expect(rules.Spec.Groups[0].Rules[2].Record).To(Equal("number_of_pods"))
			Expect(rules.Spec.Groups[0].Rules[2].Expr).To(Equal(intstr.FromString("sum(up{namespace='default', pod=~'guestbook-operator-.*'}) or vector(0)")))

			Expect(rules.Spec.Groups[1].Name).To(Equal("alerts.rules"))
			Expect(rules.Spec.Groups[1].Rules).To(HaveLen(2))
			Expect(rules.Spec.Groups[1].Rules[1].Alert).To(Equal("GuestbookOperatorDown"))
			Expect(rules.Spec.Groups[1].Rules[1].Expr).To(Equal(intstr.FromString("number_of_pods == 0")))
		})

		It("should sort the recording rules of alerts by name ('Record')", func() {
			rules, err := or.BuildPrometheusRule(
				"guestbook-operator-prometheus-rules",
				"default",
				map[string]string{"app": "guestbook-operator"},
			)

			Expect(err).To(BeNil())
			Expect(rules).NotTo(BeNil())

			Expect(rules.Spec.Groups).To(HaveLen(2))

			Expect(rules.Spec.Groups[0].Name).To(Equal("recordingRules.rules"))
			Expect(rules.Spec.Groups[0].Rules).To(HaveLen(3))
			Expect(rules.Spec.Groups[0].Rules[0].Record).To(Equal("a_test_counter"))
			Expect(rules.Spec.Groups[0].Rules[0].Expr.String()).To(Equal("sum(rate(http_requests_total[20m]))"))
			Expect(rules.Spec.Groups[0].Rules[1].Record).To(Equal("a_test_counter"))
			Expect(rules.Spec.Groups[0].Rules[1].Expr.String()).To(Equal("sum(rate(http_requests_total[5m]))"))
			Expect(rules.Spec.Groups[0].Rules[2].Record).To(Equal("number_of_pods"))
		})

		It("should sort the list of alerts by name ('Alert')", func() {
			rules, err := or.BuildPrometheusRule(
				"guestbook-operator-prometheus-rules",
				"default",
				map[string]string{"app": "guestbook-operator"},
			)

			Expect(err).To(BeNil())
			Expect(rules).NotTo(BeNil())

			Expect(rules.Spec.Groups).To(HaveLen(2))

			Expect(rules.Spec.Groups[1].Name).To(Equal("alerts.rules"))
			Expect(rules.Spec.Groups[1].Rules).To(HaveLen(2))
			Expect(rules.Spec.Groups[1].Rules[0].Alert).To(Equal("ATestAlert"))
			Expect(rules.Spec.Groups[1].Rules[1].Alert).To(Equal("GuestbookOperatorDown"))
		})
	})
})
