# 🔍 Kubernetes Operator Observability Toolkit

This repository contains a set of opinionated observability utilities and
wrappers for Kubernetes Operators using
[Prometheus Golang client](https://github.com/prometheus/client_golang).

The goal is to help developers of Kubernetes Operators instrument their
operators, while avoiding common pitfalls and mistakes, and keep their codebase
organised, clean and well documented.

### 🎯 Our Mission:
Empower Kubernetes Operator developers with tools that align with the
[Operator SDK Observability Best Practices](https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/).

### 🚀 Get Started:
Explore the [examples](_examples) directory for hands-on guidance on leveraging
these utilities and wrappers.

## Design

### Metrics

Operator developers can make use of the utilities provided here to uniformize
the way metrics are registered and their values set. In many projects,
inconsistent handling of metrics registration and setting can obscure the bigger
picture. Developers define, register and set metrics in multiple ways and
places. This makes it hard to have a global view of the existing metrics, their
values, and how they are set. This tool aims to bring clarity and consistency to
the way metrics are handled.

#### Usage

**Scope Your Metrics:** Differentiate metrics based on their relevance. For
instance, metrics like `...reconcile_count` pertain to the operator's workload,
while metrics like `...out_of_band_modifications_count` relate to the custom
resources the operator manages. Grouping these metrics in separate files ensures
clarity and separation of concerns.

```go
// metrics/operator_metrics.go

var (
  operatorMetrics = []operatormetrics.Metric{
    reconcileCount,
  }

  reconcileCount = operatormetrics.NewCounter(
    operatormetrics.MetricOpts{
      Name: metricPrefix + "reconcile_count",
      Help: "Number of times the operator has executed the reconcile loop",
      ConstLabels: map[string]string{
        "controller": "guestbook",
      },
      ExtraFields: map[string]string{
        "StabilityLevel": "STABLE",
      },
    },
  )
)

func IncrementReconcileCountMetric() {
  reconcileCount.Inc()
}
```

**Registration:** All metrics should be registered, ideally within a
`SetupMetrics()` function. This ensures a centralized point of control for all
your metrics.

```go
// metrics/metrics.go
import (
  runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func SetupMetrics() {
  // When using controller-runtime metrics, you must register the metrics
  // with the controller-runtime metrics registry 
  operatormetrics.Register = runtimemetrics.Registry.Register
  
  err := operatormetrics.RegisterMetrics(operatorMetrics, crMetrics, ...)
...
```

**Business Logic Separation:** While setting metric values, it's crucial to keep
monitoring logic distinct from the core business logic of the operator. This
ensures that the primary functionality remains uncluttered.  The operator
developer would use the `IncrementReconcileCountMetric()` to increment the
`...reconcile_count` metric in the reconcile loop.

Remember that for metrics that require more logic to set their values, we should
still make an effort to avoid adding monitoring logic code to the business logic
of the operator.

#### Collectors

Need to fetch data from Kubernetes resources or external systems like Cloud
Provider APIs? Create a custom collector. Adhering to the principles outlined
for metrics, these collectors come with a callback function triggered during
metric collection. This function serves as the bridge to external systems,
fetching data and setting metric values accordingly.

By using this approach, developers do not need to worry about the implementing
the Prometheus Golang client collector interface with the Describe and Collect
methods. This package takes care of that for them. Developers only need to
define the list of metrics they want to create and a callback function to set
the values of those metrics.

Also, in the Prometheus Golang client, collectors are free to create and push any new
metric. Most of the time, that leads to confusion and inconsistency. This
package enforces a strict way to define collectors by explicitly specifying the
metrics that the collector will push. This ensures that the created metrics are
consistent, making them easier to track, validate, and document.

```go
err = operatormetrics.RegisterCollector(customResourceCollector, ...)
...
```

```go
// metrics/custom_resource_collector.go

...
func SetupCustomResourceCollector(k8sClient *kubernetes.Clientset) {
  collectorK8sClient = k8sClient
}

var (
  customResourceCollector = operatormetrics.Collector{
    Metrics: []operatormetrics.Metric{
      crCount, // NewGaugeVec
      metric2, // NewCounter
      metric3, // NewHistogram
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

  ...
)

func customResourceCollectorCallback() []operatormetrics.CollectorResult {
  result := unstructured.UnstructuredList{}
  err := collectorK8sClient.List(context.TODO(), &result, client.InNamespace("default"))
  ...

  crCountValue = float64(len(result.Items))
  ...

  return []operatormetrics.CollectorResult{
    { Metric: crCount, Value: crCountValue, Labels: []string{"default"} },
    { Metric: metric2, Value: metric2Value },
    { Metric: metric3, Value: metric3Value },
  }
}
```

### Prometheus Rules

This section describes how to create and manage Prometheus rules to be
reconciled by your Kubernetes Operator. Prometheus' rules are a crucial part of
observability, enabling you to define alerts and record new time series based
on existing metric data.

#### Recording Rules

Recording rules allow you to precompute frequently needed or computationally
expensive expressions and save their result as a new set of time series.

Unlike the Prometheus Golang client, this package provides an opinionated way to
define recording rules. They are considered as first-class metrics and should be
defined in a similar fashion as the metrics. By using the proposed approach, we
improve code modularity and organization, and make versioning and evolution
easier.

Having strict rules for the definition of recording rules ensures enhanced
metadata and documentation, improved user experience, and better integration
with external tools.

The file holding the recording rules related to the operator workload might look
like this:

```go
// rules/operator_recording_rules.go

var operatorRecordingRules = []operatorrules.RecordingRule{
  ...
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
  ...
}
```

#### Alerts

Alerts notify you when specific conditions are met, such as when a metric value
exceeds a certain threshold or when a system component is unavailable. You can
configure alerts using Prometheus alerting rules.

```go
// rules/operator_alerts.go


var operatorAlerts = []promv1.Rule{
  ...
  {
    Alert: "GuestbookOperatorNotReady",
    Expr:  intstr.FromString(fmt.Sprintf("%snumber_of_ready_pods < %snumber_of_pods", recordingRulesPrefix, recordingRulesPrefix)),
    For:   "5m",
    Annotations: map[string]string{
      "summary":     "Guestbook operator is not ready",
      "description": "Guestbook operator is not ready for more than 5 minutes.",
    },
    Labels: map[string]string{
      "severity": "critical",
    },
  },
}

```

#### Setup

Register your rules during the initialization phase with functions like
`SetupRules()`. This centralizes rule management and ensures that all rules are
consistently loaded and applied.

```go
func SetupRules() *promv1.PrometheusRule {
  err := operatorrules.RegisterRecordingRules(recordingRules...)
  ...
  
  err = operatorrules.RegisterAlerts(alerts...)
  ...
  
  prometheusRuleObj, err := operatorrules.BuildPrometheusRule(
    "guestbook-operator-prometheus-rules",          // name
    "default",                                      // namespace
    map[string]string{"app": "guestbook-operator"}, // labels
  )
  
  // create PrometheusRule object
  ...
```

### Documentation

Having all resources in one place makes it easy to document them and track the
changes. The documentation of the metrics, recording rules, and alerts can be
generated from the code using docs utilities. The utilities will generate a
string with the documentation that you can later print or save to a file. The
documentation includes a default template that you can customize.

For metrics and recording rules:
```go
func main() {
  metrics.SetupMetrics()
  rules.SetupRules()

  docsString := docs.BuildMetricsDocs(metrics.ListMetrics(), rules.ListRecordingRules())
  fmt.Println(docsString)
}
```

For alerts:
```go
func main() {
  rules.SetupRules()
  docsString := docs.BuildAlertsDocs(alerts.ListAlerts())
  fmt.Println(docsString)
}
```

## Documentation

- Alert and Recording Rules validation: [docs/AlertsAndRecordingRulesValidation.md](docs/AlertsAndRecordingRulesValidation.md)
- Use a Different Registry for metrics and collectors: [docs/UseDifferentRegistry.md](docs/UseDifferentRegistry.md)

## Next Steps

- Add validation for metrics, and improve for recording rules

- Add a declarative way to define metrics, recording rules, and alerts

- Create a set of macros to make it easier to define metrics, recording rules,
and alerts expressions

- Propose design changes to the Operator SDK examples

- Also add utils for Events and E2E tests

- Build a Kubebuilder/Operator SDK plugin to allow developers to effortlessly
add observability to their operators

- Add Kubebuilder/Operator SDK command line instructions to generate code for
new Metrics, Alerts, and Events
