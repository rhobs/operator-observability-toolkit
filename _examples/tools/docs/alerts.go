package main

import (
	"fmt"

	"github.com/rhobs/operator-observability/examples/rules"
	"github.com/rhobs/operator-observability/pkg/docs"
)

func main() {
	rules.SetupRules()
	docsString := docs.BuildAlertsDocs(rules.ListAlerts())
	fmt.Println(docsString)
}
