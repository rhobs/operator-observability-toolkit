package metrics

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

const metricPrefix = "guestbook_operator_"

var (
	// Add your custom metrics here
	metrics = [][]operatormetrics.Metric{
		operatorMetrics,
	}

	// Add your custom collectors here
	collectors = []operatormetrics.Collector{
		customResourceCollector,
		perSecondDataCollector,
	}
)

func SetupMetrics() {
	// When using controller-runtime metrics, you must register the metrics
	// with the controller-runtime metrics registry
	// operatormetrics.Register = runtimemetrics.Registry.Register

	// Add your custom metrics here
	err := operatormetrics.RegisterMetrics(metrics...)
	if err != nil {
		panic(err)
	}

	// Add your custom collectors here
	err = operatormetrics.RegisterCollector(collectors...)
	if err != nil {
		panic(err)
	}
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
