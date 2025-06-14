package main

import (
	"fmt"
	"os"

	"github.com/rhobs/operator-observability-toolkit/examples/rules"
	"github.com/rhobs/operator-observability-toolkit/pkg/testutil"
)

func main() {
	rules.SetupRules()
	alerts := rules.ListAlerts()
	problems := testutil.New().LintAlerts(alerts)

	if len(problems) == 0 {
		os.Exit(0)
	}

	for _, problem := range problems {
		fmt.Printf("%s: %s\n", problem.ResourceName, problem.Description)
	}
	os.Exit(1)
}
