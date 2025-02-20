# Declarative Alerts

## Overview

In operator-observability, custom alerts can be defined using YAML files
following a specific format. These alerts can be loaded and registered on
runtime or automatically loaded by the system on build time (if placed inside
the package where alerts are registered).

Each file can define a set of common labels and one or more Prometheus-style
alert rules. The system parses these files, converts them to Prometheus alert
rules, and registers them for monitoring.

This feature allows for flexible and dynamic monitoring setups, enabling
operators to define and modify alerts without changing code.

Example configuration:
```yaml
common_labels:
- name: controller
  value: example-operator

alerts:
- name: CustomAlertForIncident
  expr: custom_incident_count > 0
  for: 5m
  annotations:
    summary: Custom Incident Alert
    description: This alert is triggered when a custom incident is detected.
    runbook_url: https://customer.com/runbooks/CustomAlertForIncident
  labels:
    severity: critical
```

## Runtime

```go
runtimeAlerts, err := operatorrules.LoadAlertFile(alertConfigFile)
// handle err

err = operatorRegistry.RegisterAlerts(runtimeAlerts)
// handle err
```

See full code example: [Example Rules Registration](../_examples/pkg/rules/rules.go)

Example Configuration: [Custom Runtime Alerts](../_examples/data/runtime_alerts/custom_runtime_alerts.yaml)

## Build Time

> [!WARNING]
> All alerts defined in ***_alerts.yaml** files inside the package where alerts
> are registered are automatically loaded.
>
> Be cautious when adding or modifying alerts.

## How Alerts Are Processed

The system automatically processes and registers these alerts using the following steps:

1. **File Detection:** The application scans the directory where alerts are
registered alerts for all ***_alerts.yaml** files.

2. **YAML Parsing:** Each file is parsed into internal structures. If there are
any syntax errors or the file cannot be parsed, an error will be logged and the
registration process will fail.

3. **Registration:** The Prometheus alert rules are automatically registered
internally and will be rendered in the next `BuildPrometheusRule` call.

Example: [Custom Build Time Alerts](../_examples/pkg/rules/custom_buildtime_alerts.yaml)
