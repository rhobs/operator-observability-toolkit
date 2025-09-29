package functional_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = Describe("Collectors", func() {
	var ts *httptest.Server

	BeforeEach(func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
		ts = httptest.NewServer(mux)
	})

	AfterEach(func() {
		ts.Close()
	})

	Context("registered collectors", func() {
		crCount := operatormetrics.NewGauge(
			operatormetrics.MetricOpts{
				Name: "demo_cr_count",
				Help: "Number of existing guestbook custom resources",
			},
		)

		customResourceCollectorCallback := func() []operatormetrics.CollectorResult {
			return []operatormetrics.CollectorResult{
				{
					Metric: crCount,
					Value:  1.0,
				},
			}
		}

		customResourceCollector := operatormetrics.Collector{
			Metrics:         []operatormetrics.Metric{crCount},
			CollectCallback: customResourceCollectorCallback,
		}

		BeforeEach(func() {
			err := operatormetrics.RegisterCollector(customResourceCollector)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).ToNot(HaveOccurred())
		})

		It("help and type should be exported correctly", func() {
			result := metricsHTTPRequest(ts.URL)

			Expect(result).To(ContainSubstring("# HELP demo_cr_count Number of existing guestbook custom resources"))
			Expect(result).To(ContainSubstring("# TYPE demo_cr_count gauge"))
		})

		It("value with label should be exported correctly", func() {
			result := metricsHTTPRequest(ts.URL)
			Expect(result).To(ContainSubstring("demo_cr_count 1"))
		})
	})
})
