package metrics

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	collectorK8sClient client.Client
)

func SetupCustomResourceCollector(k8sClient client.Client) {
	collectorK8sClient = k8sClient
}

var (
	customResourceCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			crCount,
		},
		CollectCallback: customResourceCollectorCallback,
	}

	crCount = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name:        metricPrefix + "cr_count",
			Help:        "Number of existing guestbook custom resources",
			ConstLabels: map[string]string{"controller": "guestbook"},
			ExtraFields: map[string]string{
				"StabilityLevel":    "DEPRECATED",
				"DeprecatedVersion": "1.14.0",
			},
		},
		[]string{"namespace"},
	)
)

func customResourceCollectorCallback() []operatormetrics.CollectorResult {
	result := unstructured.UnstructuredList{}

	if collectorK8sClient == nil {
		return []operatormetrics.CollectorResult{}
	}

	err := collectorK8sClient.List(context.TODO(), &result, client.InNamespace("default"))
	if err != nil {
		return []operatormetrics.CollectorResult{}
	}

	return []operatormetrics.CollectorResult{
		{
			Metric: crCount,
			Labels: []string{"default"},
			Value:  float64(len(result.Items)),
		},
	}
}
