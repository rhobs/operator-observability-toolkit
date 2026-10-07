# Rule Unit Testing

The toolkit lets you write
[Prometheus rule unit tests](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/)
as Go code, next to the rules they cover, instead of as separate YAML files.

The rules under test are taken from your `operatorrules.Registry`, so the
expressions you test are the expressions you ship.

Tests are executed by `promtool`, which must be installed. See
[Installing promtool](#installing-promtool).

## Writing a test

Put the test beside the rule definition:

```go
// rules/operator_alerts_test.go
func TestGuestbookOperatorDown(t *testing.T) {
	SetupRules()
	ruletest.SetRegistry(operatorRegistry)

	rt := ruletest.New(t)

	rt.WithSeriesInterval(time.Minute)
	rt.WithSeries(`up{namespace="guestbook-operator", pod="guestbook-operator-abc"}`, "0+0x15")

	rt.ExpectAlert(5*time.Minute, "GuestbookOperatorDown",
		ruletest.AlertLabels{
			"severity":   "critical",
			"controller": "guestbook",
		},
		ruletest.AlertAnnotations{
			"summary":     "Guestbook operator is down",
			"description": "Guestbook operator is down for more than 5 minutes.",
		})
}
```

`SetRegistry` is set once -- in `TestMain`, a `BeforeEach`, or at the top of
each test. A single builder can override it with `WithRegistry`.

`New(t)` names the scenario after the test. Use `Named("...")` to override.

**Every `Expect...` method runs immediately.** There is no `Run()` to remember:
each expectation evaluates the rules against the series configured so far and
reports to the test. Methods return the builder, so you can keep configuring
and asserting:

```go
rt := ruletest.New(t).WithSeries(`up{job="a"}`, "0+0x5")

rt.ExpectAlert(time.Minute, "TargetDown", ...)   // one series

rt.WithSeries(`up{job="b"}`, "0+0x5")

rt.ExpectAlerts(time.Minute, "TargetDown", ...)  // now two
```

**`AlertLabels` and `AlertAnnotations` must match exhaustively.** promtool
compares the full set, not a subset, so every label the alert carries --
including ones added by a recording rule, like `controller` above -- has to
appear. Omitting one is the most common source of confusing failures; the
`exp`/`got` diff in the output shows exactly what was missing.

They are named types rather than bare `map[string]string` because the two
arguments are positionally indistinguishable, and swapping them would be a
silently wrong test.

The `alertname` label is added automatically and must not be set.

`Values` uses promtool's
[expanding notation](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/#series),
for example `0+1x10`, `1 2 _ stale`.

See [_examples/rules/operator_alerts_test.go](../_examples/rules/operator_alerts_test.go)
for a complete working example.

## Alerts firing for more than one series

`ExpectAlert` describes exactly one firing instance. An alert that fires for
several series at once needs `ExpectAlerts`, because promtool matches the
firing set exhaustively:

```go
rt.ExpectAlerts(time.Minute, "TargetDown",
	ruletest.ExpectedAlert{
		Labels:      ruletest.AlertLabels{"severity": "critical", "job": "a"},
		Annotations: ruletest.AlertAnnotations{"summary": "a is down"},
	},
	ruletest.ExpectedAlert{
		Labels:      ruletest.AlertLabels{"severity": "critical", "job": "b"},
		Annotations: ruletest.AlertAnnotations{"summary": "b is down"},
	})
```

## Asserting an alert does not fire

```go
rt.ExpectNoAlert(5*time.Minute, "GuestbookOperatorDown")
```

## Testing expressions directly

```go
rt.ExpectSamples(5*time.Minute, "guestbook_operator_number_of_pods",
	ruletest.Sample{
		Labels: `guestbook_operator_number_of_pods{controller="guestbook"}`,
		Value:  0,
	})
```

To assert an expression returns nothing:

```go
rt.ExpectNoSamples(5*time.Minute, "guestbook_operator_retired_metric")
```

Use `ExpectNoSamples` rather than `ExpectSamples` with no samples. An
expression that returns nothing because the metric name is misspelled looks
identical to one that returns nothing by design, so the intent has to be
stated; `ExpectSamples` with an empty list is rejected.

## Durations

Durations are `time.Duration` and are converted to Prometheus's format on the
way out. Prometheus durations admit no decimal point, so anything finer than a
millisecond (`1500 * time.Microsecond`) is rejected with an error naming the
scenario and field.

## Ginkgo and Gomega

The builder takes `ginkgo.GinkgoT()`, and names the scenario after the full
spec text:

```go
var _ = Describe("Guestbook alerts", func() {
	BeforeEach(func() {
		SetupRules()
		ruletest.SetRegistry(operatorRegistry)
	})

	It("fires GuestbookOperatorDown when no pods are up", func() {
		rt := ruletest.New(GinkgoT())

		rt.WithSeriesInterval(time.Minute)
		rt.WithSeries(`up{namespace="guestbook-operator", pod="guestbook-operator-abc"}`, "0+0x15")

		rt.ExpectAlert(5*time.Minute, "GuestbookOperatorDown",
			ruletest.AlertLabels{"severity": "critical", "controller": "guestbook"},
			ruletest.AlertAnnotations{
				"summary":     "Guestbook operator is down",
				"description": "Guestbook operator is down for more than 5 minutes.",
			})
	})
})
```

Call `New(GinkgoT())` **inside** the spec body, not in the surrounding
`Describe`: `GinkgoT()` is bound to the running spec.

A `Builder` holds no package-level state, so specs using it are safe under
parallel Ginkgo and `t.Parallel()`.

For a Gomega-matcher style, `Evaluate` returns the results instead of
reporting them:

```go
Expect(ruletest.Evaluate(operatorRegistry, scenario)).To(matchers.Pass())
```

`ruletest.TestingT` is a minimal interface rather than `testing.TB`, because
`testing.TB` has an unexported method that prevents `GinkgoT()` from
satisfying it.

## Installing promtool

`promtool` cannot be installed with `go install`, because the
`prometheus/prometheus` module contains `replace` directives. Use the installer
shipped with this toolkit:

```makefile
PROMTOOL_VERSION ?= 3.15.0
PROMTOOL ?= $(shell pwd)/bin/promtool

$(PROMTOOL):
	go run github.com/rhobs/operator-observability-toolkit/cmd/promtoolinstall@latest \
		-version $(PROMTOOL_VERSION) -o $@

test-rules: $(PROMTOOL)
	PROMTOOL=$(PROMTOOL) go test ./rules/...
```

The installer verifies the download against the published `sha256sums.txt`.

Note the version mapping: the Go module is published as `v0.315.0`, but the
release you download is `v3.15.0`. Pass the release form (`3.15.0`).

Windows is not supported by the installer; download promtool manually and set
`PROMTOOL`.

## Environment variables

| Variable | Effect |
|---|---|
| `PROMTOOL` | Path to the promtool binary. Takes precedence over `PATH`. |
| `RULETEST_SKIP_IF_MISSING` | Skip rule tests instead of failing when promtool is absent. |
| `RULETEST_KEEP_ARTIFACTS` | Keep the generated YAML and print its directory. |

By default, a missing promtool **fails** the test rather than skipping it, so a
green run always means the rules were actually checked.

## Troubleshooting

Set `RULETEST_KEEP_ARTIFACTS=1` to keep the generated files, then replay the
exact command by hand:

```bash
RULETEST_KEEP_ARTIFACTS=1 go test ./rules/...
# ruletest: keeping generated files in /tmp/ruletest-1234
promtool test rules /tmp/ruletest-1234/scenario-0/tests.yaml
```

Each scenario gets its own `scenario-N` directory, because promtool takes one
evaluation interval per file and batching scenarios would evaluate all but the
first at the wrong resolution.

Note that promtool only *warns* when a `rule_files` entry matches nothing and
still exits 0, so a mistyped path reports `SUCCESS` having loaded no rules at
all. The generated `tests.yaml` therefore references the rule file by absolute
path, and you should run the command exactly as printed.
