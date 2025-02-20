# Guestbook Operator Metrics

### guestbook_operator_cr_count
[DEPRECATED in 1.14.0] Number of existing guestbook custom resources. Type: Gauge.

### guestbook_operator_reconcile_action_count
[ALPHA] Number of times the operator has executed the reconcile loop with a given action. Type: Counter.

### guestbook_operator_reconcile_count
Number of times the operator has executed the reconcile loop. Type: Counter.

### guestbook_operator_number_of_pods
Number of guestbook operator pods in the cluster. Type: Gauge.

### guestbook_operator_number_of_ready_pods
[ALPHA] Number of ready guestbook operator pods in the cluster. Type: Gauge.

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
