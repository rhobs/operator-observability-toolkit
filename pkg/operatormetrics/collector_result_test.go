package operatormetrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = Describe("CollectorResult", func() {
	var (
		counter = operatormetrics.NewCounterVec(
			operatormetrics.MetricOpts{
				Name: "collector_test_counter",
				Help: "A test counter",
			},
			[]string{"label_key"},
		)

		cr = operatormetrics.CollectorResult{
			Metric: counter,
			Labels: []string{"label_value"},
			Value:  5,
		}
	)

	Context("GetLabelValue", func() {
		It("should return the label value of an existing label key", func() {
			labelValue, err := cr.GetLabelValue("label_key")
			Expect(err).NotTo(HaveOccurred())
			Expect(labelValue).To(Equal("label_value"))
		})

		It("should return an error for a non-existing label key", func() {
			_, err := cr.GetLabelValue("non_existing_label_key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("label not found"))
		})

		It("should return an error for an empty label key", func() {
			_, err := cr.GetLabelValue("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("label not found"))
		})
	})
})
