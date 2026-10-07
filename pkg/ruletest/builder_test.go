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
)

// recordingRuleRegistry has a recording rule as well as an alert, for the
// PromQL expectations.
func recordingRuleRegistry() *operatorrules.Registry {
	registry := operatorrules.NewRegistry()

	Expect(registry.RegisterRecordingRules([]operatorrules.RecordingRule{{
		MetricsOpts: operatormetrics.MetricOpts{Name: "job:up:count"},
		MetricType:  operatormetrics.GaugeType,
		Expr:        intstr.FromString("sum(up) or vector(0)"),
	}})).To(Succeed())

	Expect(registry.RegisterAlerts([]promv1.Rule{{
		Alert:       "ExampleAlertDown",
		Expr:        intstr.FromString("up == 0"),
		Labels:      map[string]string{"severity": "critical"},
		Annotations: map[string]string{"summary": "target is down"},
	}})).To(Succeed())

	return registry
}

var _ = Describe("Builder", func() {
	var rt *recordingT

	BeforeEach(func() {
		requirePromtool()
		rt = &recordingT{name: "TestSomething"}
	})

	Describe("configuration", func() {
		It("should take its scenario name from the test by default", func() {
			b := ruletest.New(rt).WithRegistry(newTestRegistry())
			Expect(b.ScenarioName()).To(Equal("TestSomething"))
		})

		It("should let Named override the test name", func() {
			b := ruletest.New(rt).WithRegistry(newTestRegistry()).Named("custom name")
			Expect(b.ScenarioName()).To(Equal("custom name"))
		})

		It("should use the registry set with SetRegistry", func() {
			ruletest.SetRegistry(newTestRegistry())
			DeferCleanup(func() { ruletest.SetRegistry(nil) })

			ruletest.New(rt).
				WithSeries(`up{job="x"}`, "0+0x5").
				ExpectAlert(time.Minute, "ExampleAlertDown",
					ruletest.AlertLabels{"severity": "critical", "job": "x"},
					ruletest.AlertAnnotations{"summary": "target is down"})

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})

		It("should fatal when no registry is configured", func() {
			ruletest.SetRegistry(nil)

			ruletest.New(rt).ExpectNoAlert(time.Minute, "ExampleAlertDown")

			Expect(rt.fatals).To(HaveLen(1))
			Expect(rt.fatals[0]).To(ContainSubstring("SetRegistry"))
		})

		It("should let WithRegistry override the package registry", func() {
			ruletest.SetRegistry(operatorrules.NewRegistry())
			DeferCleanup(func() { ruletest.SetRegistry(nil) })

			ruletest.New(rt).
				WithRegistry(newTestRegistry()).
				WithSeries(`up{job="x"}`, "1+0x5").
				ExpectNoAlert(time.Minute, "ExampleAlertDown")

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})
	})

	Describe("alert expectations", func() {
		var b *ruletest.Builder

		BeforeEach(func() {
			b = ruletest.New(rt).
				WithRegistry(newTestRegistry()).
				WithSeriesInterval(time.Minute).
				WithSeries(`up{job="x"}`, "0+0x5")
		})

		It("should pass when the alert fires as expected", func() {
			b.ExpectAlert(time.Minute, "ExampleAlertDown",
				ruletest.AlertLabels{"severity": "critical", "job": "x"},
				ruletest.AlertAnnotations{"summary": "target is down"})

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})

		It("should report a failure when the labels do not match", func() {
			b.ExpectAlert(time.Minute, "ExampleAlertDown",
				ruletest.AlertLabels{"severity": "warning", "job": "x"},
				ruletest.AlertAnnotations{"summary": "target is down"})

			Expect(rt.errors).To(HaveLen(1))
			Expect(rt.errors[0]).To(ContainSubstring("severity"))
		})

		It("should fatal on an unknown alert name", func() {
			b.ExpectNoAlert(time.Minute, "NoSuchAlert")

			Expect(rt.fatals).To(HaveLen(1))
			Expect(rt.fatals[0]).To(ContainSubstring("NoSuchAlert"))
		})

		It("should report a failure when ExpectNoAlert is wrong", func() {
			b.ExpectNoAlert(time.Minute, "ExampleAlertDown")

			Expect(rt.errors).To(HaveLen(1))
		})

		It("should expect several simultaneous alerts", func() {
			ruletest.New(rt).
				WithRegistry(newTestRegistry()).
				WithSeries(`up{job="x"}`, "0+0x5").
				WithSeries(`up{job="y"}`, "0+0x5").
				ExpectAlerts(time.Minute, "ExampleAlertDown",
					ruletest.ExpectedAlert{
						Labels:      ruletest.AlertLabels{"severity": "critical", "job": "x"},
						Annotations: ruletest.AlertAnnotations{"summary": "target is down"},
					},
					ruletest.ExpectedAlert{
						Labels:      ruletest.AlertLabels{"severity": "critical", "job": "y"},
						Annotations: ruletest.AlertAnnotations{"summary": "target is down"},
					})

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})

		// A single ExpectAlert cannot describe a two-instance alert, because
		// promtool matches the firing set exhaustively.
		It("should fail when one alert is expected but two fire", func() {
			ruletest.New(rt).
				WithRegistry(newTestRegistry()).
				WithSeries(`up{job="x"}`, "0+0x5").
				WithSeries(`up{job="y"}`, "0+0x5").
				ExpectAlert(time.Minute, "ExampleAlertDown",
					ruletest.AlertLabels{"severity": "critical", "job": "x"},
					ruletest.AlertAnnotations{"summary": "target is down"})

			Expect(rt.errors).To(HaveLen(1))
		})
	})

	Describe("promql expectations", func() {
		It("should pass when the samples match", func() {
			ruletest.New(rt).
				WithRegistry(recordingRuleRegistry()).
				WithSeries(`up{job="x"}`, "1+0x5").
				WithSeries(`up{job="y"}`, "1+0x5").
				ExpectSamples(time.Minute, "job:up:count",
					ruletest.Sample{Labels: "job:up:count", Value: 2})

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})

		It("should report a failure when the value is wrong", func() {
			ruletest.New(rt).
				WithRegistry(recordingRuleRegistry()).
				WithSeries(`up{job="x"}`, "1+0x5").
				ExpectSamples(time.Minute, "job:up:count",
					ruletest.Sample{Labels: "job:up:count", Value: 99})

			Expect(rt.errors).To(HaveLen(1))
		})

		It("should assert an expression returns nothing", func() {
			ruletest.New(rt).
				WithRegistry(recordingRuleRegistry()).
				WithSeries(`up{job="x"}`, "1+0x5").
				ExpectNoSamples(time.Minute, "a_metric_that_does_not_exist")

			Expect(rt.errors).To(BeEmpty())
			Expect(rt.fatals).To(BeEmpty())
		})
	})

	Describe("accumulation", func() {
		It("should let later expectations see series added after an earlier one", func() {
			b := ruletest.New(rt).
				WithRegistry(recordingRuleRegistry()).
				WithSeries(`up{job="x"}`, "1+0x5")

			b.ExpectSamples(time.Minute, "job:up:count",
				ruletest.Sample{Labels: "job:up:count", Value: 1})
			Expect(rt.errors).To(BeEmpty(), "first expectation should see one series")

			b.WithSeries(`up{job="y"}`, "1+0x5")

			b.ExpectSamples(time.Minute, "job:up:count",
				ruletest.Sample{Labels: "job:up:count", Value: 2})
			Expect(rt.errors).To(BeEmpty(), "second expectation should see both series")
		})

		It("should return the builder so expectations can be chained", func() {
			returned := ruletest.New(rt).
				WithRegistry(newTestRegistry()).
				WithSeries(`up{job="x"}`, "0+0x5").
				ExpectAlert(time.Minute, "ExampleAlertDown",
					ruletest.AlertLabels{"severity": "critical", "job": "x"},
					ruletest.AlertAnnotations{"summary": "target is down"}).
				ExpectNoAlert(time.Minute, "ExampleAlertDown")

			Expect(returned).ToNot(BeNil())
			// The second expectation is wrong, so exactly one failure.
			Expect(rt.errors).To(HaveLen(1))
		})
	})
})
