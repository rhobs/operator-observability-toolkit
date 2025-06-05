package testutil_test

import (
	_ "embed"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/rhobs/operator-observability/pkg/testutil"
)

//go:embed testdata/metrics.txt
var metricsEndpointResponse string

var _ = Describe("FetchMetrics", func() {
	var server *ghttp.Server
	var metricsFetcher testutil.MetricsFetcher

	BeforeEach(func() {
		server = ghttp.NewServer()

		path := "/metrics"
		statusCode := 200

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", path),
				ghttp.RespondWithPtr(&statusCode, &metricsEndpointResponse),
			))

		metricsFetcher = testutil.NewMetricsFetcher(server.URL() + "/metrics")
	})

	AfterEach(func() {
		server.Close()
	})

	It("Should fetch all metrics with a given name", func() {
		metricsFetcher.AddNameFilter("kubevirt_vm_count_total")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).To(HaveKey("kubevirt_vm_count_total"))
		mr := metrics["kubevirt_vm_count_total"]

		Expect(mr).To(HaveLen(2))
		Expect(mr[0].Name).To(Equal("kubevirt_vm_count_total"))
		Expect(mr[1].Name).To(Equal("kubevirt_vm_count_total"))
	})

	It("Should fetch all metrics with a given name and a single label pair", func() {
		metricsFetcher.AddLabelFilter("status", "failed")
		metricsFetcher.AddNameFilter("kubevirt_migration_count")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).To(HaveKey("kubevirt_migration_count"))
		mr := metrics["kubevirt_migration_count"]

		Expect(mr).To(HaveLen(2))
		Expect(mr[0].Name).To(Equal("kubevirt_migration_count"))
		Expect(mr[0].Labels).To(HaveKeyWithValue("node", "node01"))
		Expect(mr[0].Labels).To(HaveKeyWithValue("status", "failed"))
		Expect(mr[0].Value).To(Equal(2.0))

		Expect(mr[1].Name).To(Equal("kubevirt_migration_count"))
		Expect(mr[1].Labels).To(HaveKeyWithValue("node", "node02"))
		Expect(mr[1].Labels).To(HaveKeyWithValue("status", "failed"))
		Expect(mr[1].Value).To(Equal(1.0))
	})

	It("Should fetch all metrics with a given name and multiple label pairs", func() {
		metricsFetcher.AddLabelFilter("namespace", "default", "vm_name", "vm1")
		metricsFetcher.AddNameFilter("kubevirt_vm_memory_usage_bytes")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).To(HaveKey("kubevirt_vm_memory_usage_bytes"))
		mr := metrics["kubevirt_vm_memory_usage_bytes"]

		Expect(mr).To(HaveLen(1))
		Expect(mr[0].Name).To(Equal("kubevirt_vm_memory_usage_bytes"))
		Expect(mr[0].Labels).To(HaveKeyWithValue("namespace", "default"))
		Expect(mr[0].Labels).To(HaveKeyWithValue("vm_name", "vm1"))
		Expect(mr[0].Value).To(Equal(204800.0))
	})

	It("Should return an empty map when no metrics match the filter", func() {
		metricsFetcher.AddNameFilter("non_existent_metric")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())
		Expect(metrics).To(BeEmpty())
	})

	It("Should return only metrics after a specific timestamp", func() {
		metricsFetcher.AddTimestampAfterFilter(time.Unix(1738861783, 0))
		metricsFetcher.AddNameFilter("kubevirt_vm_memory_usage_bytes")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).To(HaveKey("kubevirt_vm_memory_usage_bytes"))
		mr := metrics["kubevirt_vm_memory_usage_bytes"]

		Expect(mr).To(HaveLen(1))
		Expect(mr[0].Labels).To(HaveKeyWithValue("vm_name", "vm4"))
		Expect(mr[0].Value).To(Equal(2009600.0))
	})

	It("Should return only metrics before a specific timestamp", func() {
		metricsFetcher.AddTimestampBeforeFilter(time.Unix(1738861783, 0))
		metricsFetcher.AddNameFilter("kubevirt_vm_memory_usage_bytes")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).To(HaveKey("kubevirt_vm_memory_usage_bytes"))
		mr := metrics["kubevirt_vm_memory_usage_bytes"]

		Expect(mr).To(HaveLen(1))
		Expect(mr[0].Labels).To(HaveKeyWithValue("vm_name", "vm3"))
		Expect(mr[0].Value).To(Equal(1004800.0))
	})

	It("Should fetch metrics with given prefix name", func() {
		metricsFetcher.AddNameFilter("kubevirt_vm_")
		metrics, err := metricsFetcher.Run()
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics).ToNot(BeEmpty())
		Expect(metrics).To(HaveLen(2))
	})

})
