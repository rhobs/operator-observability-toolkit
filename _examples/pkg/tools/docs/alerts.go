package main

import (
	"fmt"

	"github.com/machadovilaca/operator-observability/examples/rules"
	"github.com/machadovilaca/operator-observability/pkg/docs"
)

func main() {
	rules.SetupRules()
	docsString := docs.BuildAlertsDocs(rules.ListAlerts())
	fmt.Println(docsString)
}
