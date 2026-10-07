package ruletest

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v3"
)

var _ = Describe("buildTestFile", func() {
	f := &ruleFile{Groups: []ruleGroup{
		{Name: "recordingRules.rules"},
		{Name: "alerts.rules"},
	}}

	It("should set rule_files to the absolute rule file path and set group_eval_order", func() {
		tf, err := buildTestFile("/tmp/example", f, Scenario{
			Name:       "s",
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: time.Minute}},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(tf.RuleFiles).To(Equal([]string{filepath.Join("/tmp/example", ruleFileName)}))
		Expect(tf.GroupEvalOrder).To(Equal([]string{"recordingRules.rules", "alerts.rules"}))
	})

	It("should default the interval to one minute", func() {
		tf, err := buildTestFile("/tmp/example", f, Scenario{
			Name:       "s",
			AlertTests: []AlertTest{{AlertName: "A"}},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(tf.Tests[0].Interval).To(Equal("1m"))
		Expect(tf.EvaluationInterval).To(Equal("1m"))
	})

	// C2 regression: evaluation_interval must come from the scenario being run,
	// never from a sibling. One scenario per file makes that structural.
	It("should take evaluation_interval from the scenario itself", func() {
		tf, err := buildTestFile("/tmp/example", f, Scenario{
			Name:       "s",
			Interval:   10 * time.Second,
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: 20 * time.Second}},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(tf.Tests[0].Interval).To(Equal("10s"))
		Expect(tf.EvaluationInterval).To(Equal("10s"))
		Expect(tf.Tests).To(HaveLen(1))
	})

	It("should render durations in Prometheus format, not Go format", func() {
		tf, err := buildTestFile("/tmp/example", f, Scenario{
			Name:       "s",
			Interval:   30 * time.Second,
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: 2500 * time.Millisecond}},
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(tf.Tests[0].Interval).To(Equal("30s"))
		Expect(tf.Tests[0].AlertRuleTests[0].EvalTime).To(Equal("2s500ms"))
	})

	It("should emit an empty exp_alerts list as a negative assertion", func() {
		tf, err := buildTestFile("/tmp/example", f, Scenario{
			Name:       "s",
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: time.Minute}},
		})
		Expect(err).ToNot(HaveOccurred())

		out, err := yaml.Marshal(tf)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(ContainSubstring("exp_alerts: []"))
	})
})

var _ = Describe("writeFiles", func() {
	It("should write both files and return the test file path", func() {
		dir := GinkgoT().TempDir()

		f := &ruleFile{Groups: []ruleGroup{{
			Name:  "alerts.rules",
			Rules: []rule{{Alert: "A", Expr: "up == 0"}},
		}}}

		path, err := writeFiles(dir, f, Scenario{
			Name:       "s",
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: time.Minute}},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(dir, testFileName)))
		Expect(filepath.Join(dir, ruleFileName)).To(BeAnExistingFile())
	})

	// Review Focus 1: Go template syntax in annotations is near-universal in
	// real alerts and must survive the YAML round-trip unmangled.
	It("should round-trip Go template syntax in annotations", func() {
		dir := GinkgoT().TempDir()

		const summary = "{{ $labels.pod }} is down, value {{ $value }}"

		f := &ruleFile{Groups: []ruleGroup{{
			Name: "alerts.rules",
			Rules: []rule{{
				Alert:       "A",
				Expr:        "up == 0",
				Annotations: map[string]string{"summary": summary},
			}},
		}}}

		_, err := writeFiles(dir, f, Scenario{
			Name:       "s",
			AlertTests: []AlertTest{{AlertName: "A", EvalTime: time.Minute}},
		})
		Expect(err).ToNot(HaveOccurred())

		data, err := os.ReadFile(filepath.Join(dir, ruleFileName))
		Expect(err).ToNot(HaveOccurred())

		var got ruleFile
		Expect(yaml.Unmarshal(data, &got)).To(Succeed())
		Expect(got.Groups[0].Rules[0].Annotations).To(HaveKeyWithValue("summary", summary))
	})

	// Review Focus 5: recording rule names contain colons and label matchers
	// contain quotes; both are YAML-significant.
	It("should round-trip expressions with YAML-significant characters", func() {
		dir := GinkgoT().TempDir()

		const expr = `sum(up{namespace='guestbook', pod=~'guestbook-.*'}) or vector(0)`

		f := &ruleFile{Groups: []ruleGroup{{
			Name:  "recordingRules.rules",
			Rules: []rule{{Record: "job:guestbook:ratio", Expr: expr}},
		}}}

		_, err := writeFiles(dir, f, Scenario{
			Name:        "s",
			PromQLTests: []PromQLTest{{Expr: expr, EvalTime: time.Minute, ExpectEmpty: true}},
		})
		Expect(err).ToNot(HaveOccurred())

		data, err := os.ReadFile(filepath.Join(dir, ruleFileName))
		Expect(err).ToNot(HaveOccurred())

		var got ruleFile
		Expect(yaml.Unmarshal(data, &got)).To(Succeed())
		Expect(got.Groups[0].Rules[0].Expr).To(Equal(expr))
		Expect(got.Groups[0].Rules[0].Record).To(Equal("job:guestbook:ratio"))
	})
})
