package declarative

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/rhobs/operator-observability/pkg/operatormetrics"
	"github.com/rhobs/operator-observability/pkg/operatorrules"
)

type Config struct {
	Observability Observability `yaml:"observability"`
}

type Observability struct {
	CommonLabels prometheus.Labels `yaml:"common_labels"`
	Groups       []Group           `yaml:"groups"`
}

type Group struct {
	Name         string            `yaml:"name"`
	CommonLabels prometheus.Labels `yaml:"common_labels"`

	Metrics []operatormetrics.Metric      `yaml:"metrics"`
	Rules   []operatorrules.RecordingRule `yaml:"recording_rules"`
	Alerts  []promv1.Rule                 `yaml:"alerts"`
}
