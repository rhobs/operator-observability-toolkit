package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rhobs/operator-observability/examples/metrics"
)

func main() {
	metrics.SetupMetrics()

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	fmt.Println("Server started on port 2112")

	v := 0.0

	for {
		metrics.IncrementReconcileCountMetric()
		metrics.IncrementReconcileActionMetric("sleep")

		metrics.SetPerSecondData("source1", v)
		v = v + 1

		time.Sleep(1 * time.Second)
	}
}
