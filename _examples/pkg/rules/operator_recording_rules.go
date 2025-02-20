package rules

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
)

var operatorRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name:        recordingRulesPrefix + "number_of_pods",
			Help:        "Number of guestbook operator pods in the cluster",
			ConstLabels: map[string]string{"controller": "guestbook"},
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(fmt.Sprintf("sum(up{namespace='%s', pod=~'guestbook-operator-.*'}) or vector(0)", namespace)),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name:        recordingRulesPrefix + "number_of_ready_pods",
			Help:        "Number of ready guestbook operator pods in the cluster",
			ExtraFields: map[string]string{"StabilityLevel": "ALPHA"},
			ConstLabels: map[string]string{"controller": "guestbook"},
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString(fmt.Sprintf("sum(up{namespace='%s', pod=~'guestbook-operator-.*', ready='true'}) or vector(0)", namespace)),
	},
}
