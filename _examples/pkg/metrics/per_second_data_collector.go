package metrics

import (
	"sync"
	"time"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	perSecondDataCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			perSecondData,
		},
		CollectCallback: perSecondDataCollectorCallback,
	}

	perSecondData = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: metricPrefix + "per_second_data",
			Help: "Data per second",
		},
		[]string{"source"},
	)

	queue = make([]data, 0)
	lock  = new(sync.Mutex)
)

type data struct {
	source    string
	value     float64
	timestamp time.Time
}

func SetPerSecondData(source string, value float64) {
	lock.Lock()
	defer lock.Unlock()

	queue = append(queue, data{source: source, value: value, timestamp: time.Now()})
}

func perSecondDataCollectorCallback() []operatormetrics.CollectorResult {
	lock.Lock()
	defer lock.Unlock()

	crs := make([]operatormetrics.CollectorResult, 0)

	for len(queue) > 0 {
		item := queue[0]
		crs = append(crs, operatormetrics.CollectorResult{
			Metric:    perSecondData,
			Labels:    []string{item.source},
			Value:     item.value,
			Timestamp: item.timestamp,
		})
		queue = queue[1:]
	}

	return crs
}
