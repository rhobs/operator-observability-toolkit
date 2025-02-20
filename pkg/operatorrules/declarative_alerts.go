package operatorrules

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type AlertFile struct {
	CommonLabels []CommonLabel `yaml:"common_labels"`
	Alerts       []Alert       `yaml:"alerts"`
}

type CommonLabel struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Alert struct {
	Alert         string            `yaml:"alert,omitempty"`
	Expr          string            `yaml:"expr,omitempty"`
	For           string            `yaml:"for,omitempty"`
	KeepFiringFor string            `yaml:"keep_firing_for,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// LoadAlertFile loads an alert file configuration and returns the list of alerts in it
func LoadAlertFile(match string) ([]promv1.Rule, error) {
	data, err := os.ReadFile(match)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", match, err)
	}

	var alertFile *AlertFile
	err = yaml.Unmarshal(data, &alertFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing yaml file %s: %v", match, err)
	}

	var alerts []promv1.Rule

	for _, alert := range alertFile.Alerts {
		addCommonLabelsToAlert(&alert, alertFile)
		alerts = append(alerts, convertToPrometheusRule(alert))
	}

	return alerts, nil
}

// LoadDeclarativeAlerts loads all the declarative alerts in the caller directory.
// Used to manage alerts in a declarative way on build time
func LoadDeclarativeAlerts() ([]promv1.Rule, error) {
	calledDiv, err := LoadCallerDiv()
	if err != nil {
		return nil, fmt.Errorf("failed to load caller directory: %v", err)
	}

	alertFiles, err := findYAMLFiles(calledDiv)
	if err != nil {
		return nil, fmt.Errorf("failed to load yaml files: %v", err)
	}

	var alerts []promv1.Rule

	for _, alertFile := range alertFiles {
		fileAlerts, err := LoadAlertFile(alertFile)
		if err != nil {
			return nil, fmt.Errorf("error loading alert file %s: %v", alertFile, err)
		}

		alerts = append(alerts, fileAlerts...)
	}

	return alerts, nil
}

var LoadCallerDiv = loadCallerDiv

func loadCallerDiv() (string, error) {
	_, file, _, ok := runtime.Caller(3)
	if !ok {
		return "", fmt.Errorf("failed to get caller information")
	}
	return filepath.Dir(file), nil
}

func findYAMLFiles(directory string) ([]string, error) {
	// Find all *_alerts.yaml files in the directory
	matches, err := filepath.Glob(filepath.Join(directory, "*_alerts.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error finding yaml files: %v", err)
	}

	return matches, nil
}

func addCommonLabelsToAlert(alert *Alert, alertFile *AlertFile) {
	for _, commonLabel := range alertFile.CommonLabels {
		if alert.Labels == nil {
			alert.Labels = map[string]string{}
		}
		alert.Labels[commonLabel.Name] = commonLabel.Value
	}
}

func convertToPrometheusRule(alert Alert) promv1.Rule {
	forValue := promv1.Duration(alert.For)
	keepFiringFor := promv1.NonEmptyDuration(alert.KeepFiringFor)

	return promv1.Rule{
		Alert:         alert.Alert,
		Expr:          intstr.FromString(alert.Expr),
		For:           &forValue,
		KeepFiringFor: &keepFiringFor,
		Labels:        alert.Labels,
		Annotations:   alert.Annotations,
	}
}
