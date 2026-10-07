package ruletest

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ruleFileName = "rules.yaml"
	testFileName = "tests.yaml"
)

// testFile mirrors the promtool unit test file format.
type testFile struct {
	RuleFiles          []string   `yaml:"rule_files"`
	EvaluationInterval string     `yaml:"evaluation_interval,omitempty"`
	GroupEvalOrder     []string   `yaml:"group_eval_order,omitempty"`
	Tests              []testCase `yaml:"tests"`
}

type testCase struct {
	Name            string            `yaml:"name,omitempty"`
	Interval        string            `yaml:"interval"`
	InputSeries     []inputSeries     `yaml:"input_series"`
	ExternalLabels  map[string]string `yaml:"external_labels,omitempty"`
	ExternalURL     string            `yaml:"external_url,omitempty"`
	AlertRuleTests  []alertRuleTest   `yaml:"alert_rule_test,omitempty"`
	PromqlExprTests []promqlExprTest  `yaml:"promql_expr_test,omitempty"`
}

type inputSeries struct {
	Series string `yaml:"series"`
	Values string `yaml:"values"`
}

type alertRuleTest struct {
	EvalTime  string          `yaml:"eval_time"`
	Alertname string          `yaml:"alertname"`
	ExpAlerts []expectedAlert `yaml:"exp_alerts"`
}

type expectedAlert struct {
	ExpLabels      map[string]string `yaml:"exp_labels,omitempty"`
	ExpAnnotations map[string]string `yaml:"exp_annotations,omitempty"`
}

type promqlExprTest struct {
	Expr       string           `yaml:"expr"`
	EvalTime   string           `yaml:"eval_time"`
	ExpSamples []expectedSample `yaml:"exp_samples"`
}

type expectedSample struct {
	Labels string  `yaml:"labels"`
	Value  float64 `yaml:"value"`
}

// buildTestFile converts a single scenario into the promtool unit test file
// format, with the rule file referenced by absolute path so the generated file
// can be replayed by hand from any directory.
//
// group_eval_order is set explicitly so recording rules always evaluate before
// the alerts that consume them.
//
// Exactly one scenario per file is deliberate. promtool takes a single
// file-level evaluation_interval, which is what actually drives rule
// evaluation, so batching scenarios with different intervals into one file
// would silently evaluate all but the first at the wrong resolution.
func buildTestFile(dir string, f *ruleFile, s Scenario) (*testFile, error) {
	interval, err := promDuration(intervalOf(s))
	if err != nil {
		return nil, fmt.Errorf("scenario %q interval: %w", s.Name, err)
	}

	tc, err := convertScenario(s, interval)
	if err != nil {
		return nil, err
	}

	return &testFile{
		RuleFiles:          []string{filepath.Join(dir, ruleFileName)},
		EvaluationInterval: interval,
		GroupEvalOrder:     f.groupNames(),
		Tests:              []testCase{tc},
	}, nil
}

func intervalOf(s Scenario) time.Duration {
	if s.Interval == 0 {
		return defaultInterval
	}

	return s.Interval
}

func convertScenario(s Scenario, interval string) (testCase, error) {
	tc := testCase{
		Name:           s.Name,
		Interval:       interval,
		InputSeries:    make([]inputSeries, 0, len(s.InputSeries)),
		ExternalLabels: s.ExternalLabels,
		ExternalURL:    s.ExternalURL,
	}

	for _, in := range s.InputSeries {
		tc.InputSeries = append(tc.InputSeries, inputSeries(in))
	}

	for _, at := range s.AlertTests {
		converted, err := convertAlertTest(at)
		if err != nil {
			return testCase{}, fmt.Errorf("scenario %q: %w", s.Name, err)
		}

		tc.AlertRuleTests = append(tc.AlertRuleTests, converted)
	}

	for _, pt := range s.PromQLTests {
		converted, err := convertPromQLTest(pt)
		if err != nil {
			return testCase{}, fmt.Errorf("scenario %q: %w", s.Name, err)
		}

		tc.PromqlExprTests = append(tc.PromqlExprTests, converted)
	}

	return tc, nil
}

func convertAlertTest(at AlertTest) (alertRuleTest, error) {
	evalTime, err := promDuration(at.EvalTime)
	if err != nil {
		return alertRuleTest{}, fmt.Errorf("alert test %q eval time: %w", at.AlertName, err)
	}

	// Always non-nil: an empty list is the documented way to assert that an
	// alert does not fire, and nil would marshal as null.
	expected := make([]expectedAlert, 0, len(at.ExpAlerts))
	for _, a := range at.ExpAlerts {
		expected = append(expected, expectedAlert{ExpLabels: a.Labels, ExpAnnotations: a.Annotations})
	}

	return alertRuleTest{
		EvalTime:  evalTime,
		Alertname: at.AlertName,
		ExpAlerts: expected,
	}, nil
}

func convertPromQLTest(pt PromQLTest) (promqlExprTest, error) {
	evalTime, err := promDuration(pt.EvalTime)
	if err != nil {
		return promqlExprTest{}, fmt.Errorf("promql test %q eval time: %w", pt.Expr, err)
	}

	samples := make([]expectedSample, 0, len(pt.ExpSamples))
	for _, s := range pt.ExpSamples {
		samples = append(samples, expectedSample(s))
	}

	return promqlExprTest{
		Expr:       pt.Expr,
		EvalTime:   evalTime,
		ExpSamples: samples,
	}, nil
}

// writeFiles writes the rule file and the test file for one scenario into dir
// and returns the path of the test file.
func writeFiles(dir string, f *ruleFile, s Scenario) (string, error) {
	rulesPath := filepath.Join(dir, ruleFileName)
	if err := marshalTo(rulesPath, f); err != nil {
		return "", fmt.Errorf("writing rule file: %w", err)
	}

	tf, err := buildTestFile(dir, f, s)
	if err != nil {
		return "", err
	}

	testsPath := filepath.Join(dir, testFileName)
	if err := marshalTo(testsPath, tf); err != nil {
		return "", fmt.Errorf("writing test file: %w", err)
	}

	return testsPath, nil
}

func marshalTo(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
