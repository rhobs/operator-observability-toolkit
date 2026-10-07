package rules

import (
	"testing"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
)

const guestbookPod = `up{namespace="guestbook-operator", pod="guestbook-operator-abc"}`

// This test lives next to the alert it covers, in operator_alerts.go.
func TestGuestbookOperatorDown(t *testing.T) {
	SetupRules()
	ruletest.SetRegistry(operatorRegistry)

	rt := ruletest.New(t)

	rt.WithSeriesInterval(time.Minute)
	rt.WithSeries(guestbookPod, "0+0x15")

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

func TestGuestbookOperatorDownDoesNotFireWhenUp(t *testing.T) {
	SetupRules()
	ruletest.SetRegistry(operatorRegistry)

	ruletest.New(t).
		WithSeriesInterval(time.Minute).
		WithSeries(guestbookPod, "1+0x15").
		ExpectNoAlert(5*time.Minute, "GuestbookOperatorDown")
}

func TestGuestbookRecordingRules(t *testing.T) {
	SetupRules()
	ruletest.SetRegistry(operatorRegistry)

	ruletest.New(t).
		WithSeries(guestbookPod, "1+0x15").
		ExpectSamples(5*time.Minute, "guestbook_operator_number_of_pods",
			ruletest.Sample{
				Labels: `guestbook_operator_number_of_pods{controller="guestbook"}`,
				Value:  1,
			})
}
