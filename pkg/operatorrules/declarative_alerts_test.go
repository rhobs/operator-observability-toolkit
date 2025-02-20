package operatorrules_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
)

var _ = Describe("Operator Rules", func() {
	var tempDir string
	var originalLoadCallerDiv func() (string, error)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "test-alerts")
		Expect(err).NotTo(HaveOccurred())

		originalLoadCallerDiv = operatorrules.LoadCallerDiv
		operatorrules.LoadCallerDiv = func() (string, error) {
			return tempDir, nil
		}
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())

		operatorrules.LoadCallerDiv = originalLoadCallerDiv
	})

	Describe("loadDeclarativeAlerts", func() {
		It("should load alerts from YAML files and convert them to Prometheus rules", func() {
			alertYaml := `
common_labels:
  - name: controller
    value: example-operator

alerts:
  - alert: CustomAlertForIncident
    expr: custom_incident_count > 0
    for: 5m
    annotations:
      summary: Custom Incident Alert
      description: This alert is triggered when a custom incident is detected.
      runbook_url: https://customer.com/runbooks/CustomAlertForIncident
    labels:
      severity: critical
`
			alertFilePath := filepath.Join(tempDir, "test_alerts.yaml")
			err := os.WriteFile(alertFilePath, []byte(alertYaml), 0644)
			Expect(err).NotTo(HaveOccurred())

			alerts, err := operatorrules.LoadDeclarativeAlerts()
			Expect(err).NotTo(HaveOccurred())
			Expect(alerts).To(HaveLen(1))

			alert := alerts[0]
			Expect(alert.Alert).To(Equal("CustomAlertForIncident"))
			Expect(alert.Expr).To(Equal(intstr.FromString("custom_incident_count > 0")))
			Expect(string(*alert.For)).To(Equal("5m"))
			Expect(alert.Labels["severity"]).To(Equal("critical"))
			Expect(alert.Labels["controller"]).To(Equal("example-operator"))
			Expect(alert.Annotations["summary"]).To(Equal("Custom Incident Alert"))
			Expect(alert.Annotations["description"]).To(Equal("This alert is triggered when a custom incident is detected."))
			Expect(alert.Annotations["runbook_url"]).To(Equal("https://customer.com/runbooks/CustomAlertForIncident"))
		})

		It("should return an error if the YAML is malformed", func() {
			alertYaml := `
common_labels:
  - name: controller
    value: example-operator

alerts:
  - alert: CustomAlertForIncident
    expr: custom_incident_count > 0
    for: 5m
    annotations:
      summary: Custom Incident Alert
      description: This alert is triggered when a custom incident is detected.
      runbook_url: https://customer.com/runbooks/CustomAlertForIncident
    labels:
      severity: critical
    malformed
`
			alertFilePath := filepath.Join(tempDir, "malformed_alerts.yaml")
			err := os.WriteFile(alertFilePath, []byte(alertYaml), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = operatorrules.LoadDeclarativeAlerts()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing yaml file"))
		})
	})
})
