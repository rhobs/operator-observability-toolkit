package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/machadovilaca/operator-observability/examples/pkg/metrics"
	"github.com/machadovilaca/operator-observability/examples/pkg/rules"
)

func main() {
	// Set up the rules (alerts and recording rules)
	var alertsConfigFile string
	flag.StringVar(&alertsConfigFile, "alerts-config", "data/runtime_alerts/custom_runtime_alerts.yaml", "Path to the alerts configuration file")
	flag.Parse()

	rules.SetupRules([]string{alertsConfigFile})

	// Set up the metrics
	metrics.SetupMetrics()

	// Example PrometheusRule that will be generated
	pr, err := rules.BuildPrometheusRule()
	if err != nil {
		panic(err)
	}
	fmt.Printf("The following PrometheusRule was generated: \n%+v\n\n", pr)

	startServer()
	startSetMetricsLoop()
}

func startServer() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)
	fmt.Println("Server started on port 2112")
}

func startSetMetricsLoop() {
	v := 0.0

	for {
		metrics.IncrementReconcileCountMetric()
		metrics.IncrementReconcileActionMetric("sleep")

		metrics.SetPerSecondData("source1", v)
		v = v + 1

		time.Sleep(1 * time.Second)
	}
}
