package rules

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

const (
	recordingRulesPrefix = "guestbook_operator_"
	namespace            = "guestbook-operator"
)

var (
	operatorRegistry = operatorrules.NewRegistry()

	// Add your custom recording rules here
	recordingRules = [][]operatorrules.RecordingRule{
		operatorRecordingRules,
	}

	// Add your custom alerts here
	alerts = [][]promv1.Rule{
		operatorAlerts,
	}
)

func SetupRules() {
	err := operatorRegistry.RegisterRecordingRules(recordingRules...)
	if err != nil {
		panic(err)
	}

	err = operatorRegistry.RegisterAlerts(alerts...)
	if err != nil {
		panic(err)
	}
}

func BuildPrometheusRule() (*promv1.PrometheusRule, error) {
	rules, err := operatorRegistry.BuildPrometheusRule(
		"guestbook-operator-prometheus-rules",
		"default",
		map[string]string{"app": "guestbook-operator"},
	)
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func ListRecordingRules() []operatorrules.RecordingRule {
	return operatorRegistry.ListRecordingRules()
}

func ListAlerts() []promv1.Rule {
	return operatorRegistry.ListAlerts()
}
