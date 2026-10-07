package ruletest

import (
	"fmt"
	"strings"
	"time"
)

// promDuration renders d in the duration format Prometheus accepts.
//
// time.Duration.String() cannot be used: it emits a decimal point for any
// duration with a fractional-second component ("2.5s"), and Prometheus's
// duration grammar admits only integers, rejecting it with
// `unknown unit "." in duration "2.5s"`.
func promDuration(d time.Duration) (string, error) {
	if d < 0 {
		return "", fmt.Errorf("negative duration %s", d)
	}

	if d%time.Millisecond != 0 {
		return "", fmt.Errorf("duration %s is finer than a millisecond, which Prometheus cannot represent", d)
	}

	if d == 0 {
		return "0s", nil
	}

	units := []struct {
		suffix string
		size   time.Duration
	}{
		{"h", time.Hour},
		{"m", time.Minute},
		{"s", time.Second},
		{"ms", time.Millisecond},
	}

	var b strings.Builder

	for _, u := range units {
		n := d / u.size
		if n == 0 {
			continue
		}

		fmt.Fprintf(&b, "%d%s", n, u.suffix)
		d -= n * u.size
	}

	return b.String(), nil
}
