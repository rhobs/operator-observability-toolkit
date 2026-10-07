package ruletest

import (
	"errors"
	"os"
)

// TestingT is the subset of the testing API this package needs. It is
// satisfied by *testing.T, *testing.B and ginkgo.GinkgoT().
//
// testing.TB is deliberately not used: it has an unexported method that
// prevents external implementations, so Ginkgo cannot satisfy it.
type TestingT interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Name() string
}

// skipper is implemented by testing types that can skip a test.
type skipper interface {
	Skipf(format string, args ...any)
}

func reportHarnessError(t TestingT, err error) {
	t.Helper()

	if !errors.Is(err, ErrPromtoolNotFound) {
		t.Fatalf("ruletest: %v", err)

		return
	}

	if os.Getenv(envSkipIfMissing) != "" {
		s, ok := t.(skipper)
		if !ok {
			// Returning quietly here would pass a test that verified nothing.
			t.Errorf("ruletest: %v; %s is set but %T cannot skip, so this test cannot be satisfied",
				err, envSkipIfMissing, t)

			return
		}

		s.Skipf("ruletest: %v; skipping because %s is set", err, envSkipIfMissing)

		return
	}

	t.Fatalf("ruletest: %v\n"+
		"Install it with `make test-rules`, or set %s to the binary path.\n"+
		"To skip these tests instead, set %s=1.",
		err, envPromtool, envSkipIfMissing)
}

func report(t TestingT, results Results) {
	t.Helper()

	for _, s := range results.Scenarios {
		for _, f := range s.Failures {
			t.Errorf("ruletest: %s\n%s", s.Name, f)
		}
	}
}
