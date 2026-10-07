package ruletest

import (
	"slices"
	"sync"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

// AlertLabels and AlertAnnotations name the two map arguments of ExpectAlert.
// They are
// positionally indistinguishable as bare maps, and swapping them would be a
// silently wrong test.
type (
	AlertLabels      map[string]string
	AlertAnnotations map[string]string
)

var (
	defaultRegistryMu sync.RWMutex
	defaultRegistry   *operatorrules.Registry
)

// SetRegistry sets the registry used by builders that do not specify one.
// Call it once, typically in TestMain or a BeforeEach.
func SetRegistry(reg *operatorrules.Registry) {
	defaultRegistryMu.Lock()
	defer defaultRegistryMu.Unlock()

	defaultRegistry = reg
}

func packageRegistry() *operatorrules.Registry {
	defaultRegistryMu.RLock()
	defer defaultRegistryMu.RUnlock()

	return defaultRegistry
}

// Builder accumulates the input series and configuration shared by a test's
// expectations.
//
// Each Expect method is terminal: it evaluates the rules immediately against
// the series configured so far and reports the outcome to the test. Later
// expectations see any series added in between.
//
// A Builder belongs to one test. It holds no package-level state, so tests
// using it are safe to run in parallel.
type Builder struct {
	t              TestingT
	registry       *operatorrules.Registry
	name           string
	interval       time.Duration
	series         []Series
	externalLabels map[string]string
	externalURL    string
}

// New starts a rule test for t. The scenario name defaults to t.Name().
//
// Under Ginkgo, call New(GinkgoT()) inside the spec body rather than in the
// surrounding container: GinkgoT() is bound to the running spec.
func New(t TestingT) *Builder {
	return &Builder{t: t, name: t.Name()}
}

// Named overrides the scenario name, which otherwise comes from the test.
func (b *Builder) Named(name string) *Builder {
	b.name = name

	return b
}

// WithRegistry uses reg instead of the one set by SetRegistry.
func (b *Builder) WithRegistry(reg *operatorrules.Registry) *Builder {
	b.registry = reg

	return b
}

// WithSeriesInterval sets the evaluation interval. It defaults to one minute.
func (b *Builder) WithSeriesInterval(d time.Duration) *Builder {
	b.interval = d

	return b
}

// WithSeries adds an input series. Values uses promtool's expanding notation,
// for example "0+1x10" or "1 2 _ stale".
func (b *Builder) WithSeries(series, values string) *Builder {
	b.series = append(b.series, Series{Series: series, Values: values})

	return b
}

// WithExternalLabels sets the external labels available during evaluation.
func (b *Builder) WithExternalLabels(l map[string]string) *Builder {
	b.externalLabels = l

	return b
}

// WithExternalURL sets the external URL available to alert templates.
func (b *Builder) WithExternalURL(u string) *Builder {
	b.externalURL = u

	return b
}

// ScenarioName returns the name this builder will report under.
func (b *Builder) ScenarioName() string {
	return b.name
}

// ExpectAlert asserts that exactly one instance of the named alert is firing
// at the given time, with the given labels and annotations.
//
// promtool matches the firing set exhaustively, so labels and annotations must
// list every one the alert carries, and an alert firing for more than one
// series needs ExpectAlerts.
//
// The alertname label is added automatically and must not be included.
func (b *Builder) ExpectAlert(at time.Duration, name string, l AlertLabels, a AlertAnnotations) *Builder {
	b.t.Helper()

	return b.ExpectAlerts(at, name, ExpectedAlert{Labels: l, Annotations: a})
}

// ExpectAlerts asserts that exactly the given instances of the named alert are
// firing at the given time. Passing no alerts asserts that it does not fire,
// though ExpectNoAlert says that more clearly.
func (b *Builder) ExpectAlerts(at time.Duration, name string, alerts ...ExpectedAlert) *Builder {
	b.t.Helper()

	scenario := b.scenario()
	scenario.AlertTests = []AlertTest{{
		EvalTime:  at,
		AlertName: name,
		ExpAlerts: alerts,
	}}

	return b.evaluate(scenario)
}

// ExpectNoAlert asserts the named alert is not firing at the given time.
func (b *Builder) ExpectNoAlert(at time.Duration, name string) *Builder {
	b.t.Helper()

	return b.ExpectAlerts(at, name)
}

// ExpectSamples asserts the expression returns exactly the given samples at
// the given time.
func (b *Builder) ExpectSamples(at time.Duration, expr string, samples ...Sample) *Builder {
	b.t.Helper()

	scenario := b.scenario()
	scenario.PromQLTests = []PromQLTest{{
		Expr:       expr,
		EvalTime:   at,
		ExpSamples: samples,
	}}

	return b.evaluate(scenario)
}

// ExpectNoSamples asserts the expression returns nothing at the given time.
func (b *Builder) ExpectNoSamples(at time.Duration, expr string) *Builder {
	b.t.Helper()

	scenario := b.scenario()
	scenario.PromQLTests = []PromQLTest{{
		Expr:        expr,
		EvalTime:    at,
		ExpectEmpty: true,
	}}

	return b.evaluate(scenario)
}

// scenario snapshots the builder's current configuration, so an expectation is
// unaffected by series added after it runs.
func (b *Builder) scenario() Scenario {
	return Scenario{
		Name:           b.name,
		Interval:       b.interval,
		InputSeries:    slices.Clone(b.series),
		ExternalLabels: b.externalLabels,
		ExternalURL:    b.externalURL,
	}
}

func (b *Builder) evaluate(s Scenario) *Builder {
	b.t.Helper()

	registry := b.registry
	if registry == nil {
		registry = packageRegistry()
	}

	if registry == nil {
		b.t.Fatalf("ruletest: no registry configured; " +
			"call ruletest.SetRegistry(reg) once, or New(t).WithRegistry(reg)")

		return b
	}

	results, err := Evaluate(registry, s)
	if err != nil {
		reportHarnessError(b.t, err)

		return b
	}

	report(b.t, results)

	return b
}
