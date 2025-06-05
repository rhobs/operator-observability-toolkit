package docs_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/rhobs/operator-observability/pkg/docs"
	"github.com/rhobs/operator-observability/pkg/operatormetrics"
	"github.com/rhobs/operator-observability/pkg/operatorrules"
)

const tpl = `#metrics-doc-test-title
{{- range . }}

{{ $deprecatedVersion := "" -}}
{{- with index .ExtraFields "DeprecatedVersion" -}}
    {{- $deprecatedVersion = printf " in %s" . -}}
{{- end -}}

{{- $stabilityLevel := "" -}}
{{- if and (.ExtraFields.StabilityLevel) (ne .ExtraFields.StabilityLevel "STABLE") -}}
	{{- $stabilityLevel = printf "[%s%s] " .ExtraFields.StabilityLevel $deprecatedVersion -}}
{{- end -}}

### {{ .Name }}
{{ print $stabilityLevel }}{{ .Help }}. Type: {{ .Type -}}.

{{- end }}

## metrics-doc-test-body
`

var recordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{Name: "CExampleRecordingRule"},
		Expr:        intstr.FromString("sum(rate(http_requests_total[5m]))"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{Name: "AExampleRecordingRule"},
		Expr:        intstr.FromString("count(rate(http_requests_total[5m]))"),
	},
}

var metrics = []operatormetrics.Metric{
	operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "BExampleGauge",
			Help: "test doc gauge",
		},
	),
	operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "DExampleCounterVec",
			Help: "test doc counterVec",
			ExtraFields: map[string]string{
				"StabilityLevel":    "ALPHA",
				"DeprecatedVersion": "1.4.0",
			},
		},
		[]string{"test-doc"},
	),
}

var _ = Describe("Metrics Documentation", func() {
	Context("Metrics and Recording Rules", func() {
		It("Checks that metrics and recording rules are documented", func() {
			docMetrics := docs.BuildMetricsDocs(metrics, recordingRules)
			Expect(docMetrics).To(ContainSubstring("CExampleRecordingRule"))
			Expect(docMetrics).To(ContainSubstring("AExampleRecordingRule"))
			Expect(docMetrics).To(ContainSubstring("BExampleGauge"))
			Expect(docMetrics).To(ContainSubstring("DExampleCounterVec"))
		})

		It("Checks that metrics and recording rules are documented with custom template", func() {
			templateDocMetrics := docs.BuildMetricsDocsWithCustomTemplate(metrics, recordingRules, tpl)
			Expect(templateDocMetrics).To(ContainSubstring("metrics-doc-test-title"))
			Expect(templateDocMetrics).To(ContainSubstring("metrics-doc-test-body"))
			Expect(templateDocMetrics).To(ContainSubstring("CExampleRecordingRule"))
			Expect(templateDocMetrics).To(ContainSubstring("AExampleRecordingRule"))
			Expect(templateDocMetrics).To(ContainSubstring("BExampleGauge"))
			Expect(templateDocMetrics).To(ContainSubstring("DExampleCounterVec"))
		})

		It("Checks that the metrics doc is sorted by metrics and recording rules name", func() {
			templateDocMetrics := docs.BuildMetricsDocsWithCustomTemplate(metrics, recordingRules, tpl)
			indexOfA := strings.Index(templateDocMetrics, "AExampleRecordingRule")
			indexOfB := strings.Index(templateDocMetrics, "BExampleGauge")
			indexOfC := strings.Index(templateDocMetrics, "CExampleRecordingRule")
			indexOfD := strings.Index(templateDocMetrics, "DExampleCounterVec")

			Expect(indexOfA).To(BeNumerically("<", indexOfB))
			Expect(indexOfB).To(BeNumerically("<", indexOfC))
			Expect(indexOfC).To(BeNumerically("<", indexOfD))
		})

		It("Checks that metrics are documented in the right format", func() {
			templateDocMetrics := docs.BuildMetricsDocsWithCustomTemplate(metrics, nil, tpl)
			Expect(templateDocMetrics).To(ContainSubstring("BExampleGauge\ntest doc gauge. Type: Gauge."))
			Expect(templateDocMetrics).To(ContainSubstring("[ALPHA in 1.4.0] test doc counterVec. Type: Counter."))
		})
	})
})
