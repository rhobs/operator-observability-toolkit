package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/examples/metrics"
	"github.com/rhobs/operator-observability-toolkit/examples/rules"
	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

const tpl = `# Guestbook Operator Metrics

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

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

func main() {
	metrics.SetupMetrics()
	rules.SetupRules()

	//docsString := docs.BuildMetricsDocs(metrics.ListMetrics(), rules.ListRecordingRules())
	docsString := docs.BuildMetricsDocsWithCustomTemplate(metrics.ListMetrics(), rules.ListRecordingRules(), tpl)
	fmt.Println(docsString)
}
