package ruletest

import (
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

// ruleFile mirrors the native Prometheus rule file format. The PrometheusRule
// CRD types are not reused: they carry JSON tags rather than YAML tags, and
// CRD-only fields that Prometheus's strict rule parser rejects.
type ruleFile struct {
	Groups []ruleGroup `yaml:"groups"`
}

type ruleGroup struct {
	Name     string `yaml:"name"`
	Interval string `yaml:"interval,omitempty"`
	Rules    []rule `yaml:"rules"`
}

type rule struct {
	Record        string            `yaml:"record,omitempty"`
	Alert         string            `yaml:"alert,omitempty"`
	Expr          string            `yaml:"expr"`
	For           string            `yaml:"for,omitempty"`
	KeepFiringFor string            `yaml:"keep_firing_for,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Annotations   map[string]string `yaml:"annotations,omitempty"`
}

// buildRuleFile derives a native Prometheus rule file from the rules registered
// in reg, using the same builder that produces the shipped PrometheusRule.
func buildRuleFile(reg *operatorrules.Registry) (*ruleFile, error) {
	spec, err := reg.BuildPrometheusRuleSpec()
	if err != nil {
		return nil, fmt.Errorf("building rule spec: %w", err)
	}

	groups := make([]ruleGroup, 0, len(spec.Groups))
	for _, g := range spec.Groups {
		groups = append(groups, convertGroup(g))
	}

	return &ruleFile{Groups: groups}, nil
}

func convertGroup(g promv1.RuleGroup) ruleGroup {
	out := ruleGroup{Name: g.Name, Rules: make([]rule, 0, len(g.Rules))}

	if g.Interval != nil {
		out.Interval = string(*g.Interval)
	}

	for _, r := range g.Rules {
		out.Rules = append(out.Rules, convertRule(r))
	}

	return out
}

func convertRule(r promv1.Rule) rule {
	out := rule{
		Record:      r.Record,
		Alert:       r.Alert,
		Expr:        r.Expr.String(),
		Labels:      r.Labels,
		Annotations: r.Annotations,
	}

	if r.For != nil {
		out.For = string(*r.For)
	}

	if r.KeepFiringFor != nil {
		out.KeepFiringFor = string(*r.KeepFiringFor)
	}

	return out
}

// groupNames returns the group names in evaluation order.
func (f *ruleFile) groupNames() []string {
	names := make([]string, 0, len(f.Groups))
	for _, g := range f.Groups {
		names = append(names, g.Name)
	}

	return names
}

// alertNames returns every alert name defined in the file.
func (f *ruleFile) alertNames() []string {
	var names []string

	for _, g := range f.Groups {
		for _, r := range g.Rules {
			if r.Alert != "" {
				names = append(names, r.Alert)
			}
		}
	}

	return names
}
