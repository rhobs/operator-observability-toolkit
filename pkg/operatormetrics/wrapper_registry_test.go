package operatormetrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability/pkg/operatormetrics"
)

var _ = Describe("Registry", func() {
	var (
		testCounterOpts = operatormetrics.MetricOpts{
			Name: "test_counter",
			Help: "A test counter",
		}
		testGaugeOpts = operatormetrics.MetricOpts{
			Name: "test_gauge",
			Help: "A test gauge",
		}
		testGaugeVecOpts = operatormetrics.MetricOpts{
			Name: "test_gauge_vec",
			Help: "A test gauge vec",
		}
	)

	Describe("RegisterMetrics", func() {
		BeforeEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should register metrics without error", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)
			gauge := operatormetrics.NewGauge(testGaugeOpts)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter, gauge})
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(2))
			Expect(metrics).To(ContainElement(counter))
			Expect(metrics).To(ContainElement(gauge))
		})

		It("should replace metrics with the same name in different RegisterMetrics call", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter})
			Expect(err).NotTo(HaveOccurred())

			err = operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter})
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(1))
			Expect(metrics).To(ContainElement(counter))
		})
	})

	Describe("UnregisterMetrics", func() {
		BeforeEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should unregister metrics without error", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)
			gauge := operatormetrics.NewGauge(testGaugeOpts)

			labels := []string{"label1", "label2"}
			gaugeVec := operatormetrics.NewGaugeVec(testGaugeVecOpts, labels)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter, gauge, gaugeVec})
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(3))
			Expect(metrics).To(ContainElement(counter))
			Expect(metrics).To(ContainElement(gauge))
			Expect(metrics).To(ContainElement(gaugeVec))

			err = operatormetrics.UnregisterMetrics([]operatormetrics.Metric{counter, gauge})
			Expect(err).NotTo(HaveOccurred())

			metrics = operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(1))
			Expect(metrics).To(ContainElement(gaugeVec))
		})
	})

	Describe("ListMetrics", func() {
		BeforeEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return a list of all registered metrics", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)
			gauge := operatormetrics.NewGauge(testGaugeOpts)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter, gauge})
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(2))
			Expect(metrics).To(ContainElement(counter))
			Expect(metrics).To(ContainElement(gauge))
		})

		It("should sort the list of metrics by name", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)
			gauge := operatormetrics.NewGauge(testGaugeOpts)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{gauge, counter})
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(2))
			Expect(metrics).To(Equal([]operatormetrics.Metric{counter, gauge}))
		})
	})

	Describe("RegisterCollector", func() {
		BeforeEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())
		})

		var customResourceCollectorCallback = func() []operatormetrics.CollectorResult {
			return []operatormetrics.CollectorResult{}
		}

		It("should register collectors without error", func() {
			collector := operatormetrics.Collector{
				Metrics: []operatormetrics.Metric{
					operatormetrics.NewCounter(testCounterOpts),
				},
				CollectCallback: customResourceCollectorCallback,
			}

			err := operatormetrics.RegisterCollector(collector)
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(1))
			Expect(metrics).To(ContainElement(collector.Metrics[0]))
		})

		It("should replace metrics with the same name in different RegisterCollector call", func() {
			collector := operatormetrics.Collector{
				Metrics: []operatormetrics.Metric{
					operatormetrics.NewCounter(testCounterOpts),
				},
				CollectCallback: customResourceCollectorCallback,
			}

			err := operatormetrics.RegisterCollector(collector)
			Expect(err).NotTo(HaveOccurred())

			err = operatormetrics.RegisterCollector(collector)
			Expect(err).NotTo(HaveOccurred())

			metrics := operatormetrics.ListMetrics()
			Expect(metrics).To(HaveLen(1))
			Expect(metrics).To(ContainElement(collector.Metrics[0]))
		})
	})

	Describe("CleanRegistry", func() {
		BeforeEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should remove all metrics from the registry", func() {
			counter := operatormetrics.NewCounter(testCounterOpts)
			gauge := operatormetrics.NewGauge(testGaugeOpts)

			err := operatormetrics.RegisterMetrics([]operatormetrics.Metric{counter, gauge})
			Expect(err).NotTo(HaveOccurred())

			err = operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())

			Expect(operatormetrics.ListMetrics()).To(BeEmpty())
		})

		It("should remove all collectors from the registry", func() {
			collector := operatormetrics.Collector{
				Metrics: []operatormetrics.Metric{
					operatormetrics.NewCounter(testCounterOpts),
				},
				CollectCallback: func() []operatormetrics.CollectorResult {
					return []operatormetrics.CollectorResult{}
				},
			}

			err := operatormetrics.RegisterCollector(collector)
			Expect(err).NotTo(HaveOccurred())

			err = operatormetrics.CleanRegistry()
			Expect(err).NotTo(HaveOccurred())

			Expect(operatormetrics.ListMetrics()).To(BeEmpty())
		})
	})
})
