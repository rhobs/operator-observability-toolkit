package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/examples/rules"
	"github.com/rhobs/operator-observability-toolkit/pkg/docs"
)

func main() {
	rules.SetupRules()
	docsString := docs.BuildAlertsDocs(rules.ListAlerts())
	fmt.Println(docsString)
}
