package functional_test

import (
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var constLabels = map[string]string{"controller": "demo"}

var _ = Describe("Metrics", func() {
	var ts *httptest.Server

	BeforeEach(func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
		ts = httptest.NewServer(mux)
	})

	AfterEach(func() {
		ts.Close()
	})

	Context("registered metrics", func() {
		reconcileCount := operatormetrics.NewCounter(
			operatormetrics.MetricOpts{
				Name:        "demo_reconcile_count",
				Help:        "Number of times the operator has executed the reconcile loop",
				ConstLabels: constLabels,
			},
		)

		reconcileDuration := operatormetrics.NewHistogram(
			operatormetrics.MetricOpts{
				Name:        "demo_reconcile_duration_seconds",
				Help:        "Duration of the reconcile loop in seconds",
				ConstLabels: constLabels,
			},
			prometheus.HistogramOpts{
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			},
		)

		BeforeEach(func() {
			metrics := []operatormetrics.Metric{
				reconcileCount,
				reconcileDuration,
			}

			err := operatormetrics.RegisterMetrics(metrics)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := operatormetrics.CleanRegistry()
			Expect(err).ToNot(HaveOccurred())
		})

		It("help and type should be exported correctly", func() {
			result := metricsHTTPRequest(ts.URL)

			Expect(result).To(ContainSubstring("# HELP demo_reconcile_count Number of times the operator has executed the reconcile loop"))
			Expect(result).To(ContainSubstring("# TYPE demo_reconcile_count counter"))

			Expect(result).To(ContainSubstring("# HELP demo_reconcile_duration_seconds Duration of the reconcile loop in seconds"))
			Expect(result).To(ContainSubstring("# TYPE demo_reconcile_duration_seconds histogram"))
		})

		It("counter value with label should be exported correctly", func() {
			reconcileCount.Inc()

			result := metricsHTTPRequest(ts.URL)
			Expect(result).To(ContainSubstring("demo_reconcile_count{controller=\"demo\"} 1"))
		})

		It("histogram value with label should be exported correctly", func() {
			reconcileDuration.Observe(0.05)
			reconcileDuration.Observe(0.2)

			result := metricsHTTPRequest(ts.URL)

			// buckets
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_bucket{controller=\"demo\",le=\"0.01\"} 0"))
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_bucket{controller=\"demo\",le=\"0.08\"} 1"))
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_bucket{controller=\"demo\",le=\"0.16\"} 1"))
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_bucket{controller=\"demo\",le=\"0.32\"} 2"))

			// sum and count
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_sum{controller=\"demo\"} 0.25"))
			Expect(result).To(ContainSubstring("demo_reconcile_duration_seconds_count{controller=\"demo\"} 2"))
		})
	})
})

func metricsHTTPRequest(baseURL string) string {
	resp, err := http.Get(baseURL + "/metrics")
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(body)
}
