// Package ruletest runs Prometheus rule unit tests written in Go against the
// rules registered in an operatorrules.Registry.
//
// Tests are executed by the promtool binary, which must be available either on
// PATH or via the PROMTOOL environment variable.
package ruletest

import "time"

// defaultInterval is the evaluation interval used when a Scenario does not set
// one. It matches the interval used in the Prometheus unit testing docs.
const defaultInterval = time.Minute

// Scenario is a single rule unit test, equivalent to one entry under `tests`
// in a promtool unit test file.
type Scenario struct {
	// Name identifies the scenario in test output. Required.
	Name string
	// Interval is the evaluation interval. Defaults to one minute.
	Interval time.Duration
	// InputSeries are the series loaded before evaluation.
	InputSeries []Series
	// ExternalLabels are added to the evaluation context.
	ExternalLabels map[string]string
	// ExternalURL is the external URL available to alert templates.
	ExternalURL string
	// AlertTests assert on alerts firing at a point in time.
	AlertTests []AlertTest
	// PromQLTests assert on the result of an expression at a point in time.
	PromQLTests []PromQLTest
}

// Series is one input time series. Values uses promtool's expanding notation,
// for example "0+1x10" or "1 2 _ stale".
type Series struct {
	Series string
	Values string
}

// AlertTest asserts which alerts of a given name are firing at EvalTime.
// An empty ExpAlerts asserts that the alert does not fire.
type AlertTest struct {
	EvalTime  time.Duration
	AlertName string
	ExpAlerts []ExpectedAlert
}

// ExpectedAlert is a single alert expected to be firing. The alertname label is
// added automatically from AlertTest.AlertName and must not be set here.
type ExpectedAlert struct {
	Labels      AlertLabels
	Annotations AlertAnnotations
}

// PromQLTest asserts the samples an expression returns at EvalTime.
//
// Exactly one of ExpSamples and ExpectEmpty must be set. An expression that
// returns nothing because the metric name is misspelled looks identical to one
// that returns nothing by design, so the intent has to be stated.
type PromQLTest struct {
	Expr       string
	EvalTime   time.Duration
	ExpSamples []Sample
	// ExpectEmpty asserts the expression returns no samples at all.
	ExpectEmpty bool
}

// Sample is a single expected sample. Labels is a metric selector string, for
// example `metric_name{label="value"}`.
type Sample struct {
	Labels string
	Value  float64
}

// Results is the outcome of evaluating a set of scenarios.
type Results struct {
	Scenarios []ScenarioResult
}

// ScenarioResult is the outcome of a single scenario. An empty Failures slice
// means the scenario passed.
type ScenarioResult struct {
	Name     string
	Failures []string
}

// Failed reports whether any scenario produced a failure.
func (r Results) Failed() bool {
	for _, s := range r.Scenarios {
		if len(s.Failures) > 0 {
			return true
		}
	}

	return false
}
