package ruletest

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("promDuration", func() {
	DescribeTable("should render durations Prometheus can parse",
		func(d time.Duration, expected string) {
			got, err := promDuration(d)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(expected))
		},
		Entry("zero", time.Duration(0), "0s"),
		Entry("milliseconds", 500*time.Millisecond, "500ms"),
		Entry("seconds", 30*time.Second, "30s"),
		Entry("minutes", 5*time.Minute, "5m"),
		Entry("hours", 2*time.Hour, "2h"),
		Entry("compound", time.Hour+30*time.Minute, "1h30m"),
		Entry("compound with seconds", 90*time.Second, "1m30s"),
		// Go's String() would render these with a decimal point, which
		// Prometheus rejects with `unknown unit "." in duration`.
		Entry("fractional seconds", 2500*time.Millisecond, "2s500ms"),
		Entry("fractional minutes", 2*time.Minute+500*time.Millisecond, "2m500ms"),
	)

	It("should reject a negative duration", func() {
		_, err := promDuration(-time.Second)
		Expect(err).To(MatchError(ContainSubstring("negative")))
	})

	It("should reject sub-millisecond precision", func() {
		_, err := promDuration(1500 * time.Microsecond)
		Expect(err).To(MatchError(ContainSubstring("millisecond")))
	})
})
