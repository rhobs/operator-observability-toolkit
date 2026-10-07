package ruletest

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRuletest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ruletest Suite")
}
