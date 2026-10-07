package ruletest_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/ruletest"
)

// recordingT captures what the builder reports, so the reporting path can be
// tested without failing the surrounding suite.
type recordingT struct {
	name   string
	errors []string
	fatals []string
}

func (r *recordingT) Helper()                   {}
func (r *recordingT) Name() string              { return r.name }
func (r *recordingT) Errorf(f string, a ...any) { r.errors = append(r.errors, fmt.Sprintf(f, a...)) }
func (r *recordingT) Fatalf(f string, a ...any) { r.fatals = append(r.fatals, fmt.Sprintf(f, a...)) }

// skippableT is a recordingT that can also skip, like *testing.T and GinkgoT().
type skippableT struct {
	recordingT
	skips []string
}

func (r *skippableT) Skipf(f string, a ...any) { r.skips = append(r.skips, fmt.Sprintf(f, a...)) }

// Compile-time proof that the real test types satisfy TestingT.
var (
	_ ruletest.TestingT = &recordingT{}
	_ ruletest.TestingT = &testing.T{}
	_ ruletest.TestingT = GinkgoT()
)

var _ = Describe("Builder without promtool", func() {
	var rt *recordingT

	BeforeEach(func() {
		rt = &recordingT{name: "TestSomething"}

		GinkgoT().Setenv("PATH", GinkgoT().TempDir())
		GinkgoT().Setenv("PROMTOOL", "")
	})

	It("should fatal with installation guidance", func() {
		// This spec is about the default behaviour, so it must not inherit an
		// opt-out from the ambient environment.
		GinkgoT().Setenv("RULETEST_SKIP_IF_MISSING", "")

		ruletest.New(rt).
			WithRegistry(newTestRegistry()).
			ExpectNoAlert(time.Minute, "ExampleAlertDown")

		Expect(rt.fatals).To(HaveLen(1))
		Expect(strings.ToLower(rt.fatals[0])).To(ContainSubstring("promtool"))
		Expect(rt.fatals[0]).To(ContainSubstring("RULETEST_SKIP_IF_MISSING"))
	})

	It("should skip when RULETEST_SKIP_IF_MISSING is set and the type can skip", func() {
		GinkgoT().Setenv("RULETEST_SKIP_IF_MISSING", "1")

		st := &skippableT{recordingT: recordingT{name: "TestSomething"}}

		ruletest.New(st).
			WithRegistry(newTestRegistry()).
			ExpectNoAlert(time.Minute, "ExampleAlertDown")

		Expect(st.skips).To(HaveLen(1))
		Expect(st.fatals).To(BeEmpty())
		Expect(st.errors).To(BeEmpty())
	})

	// A TestingT that cannot skip must not pass silently: returning quietly
	// would be a test that verified nothing and still went green.
	It("should error rather than pass silently when the type cannot skip", func() {
		GinkgoT().Setenv("RULETEST_SKIP_IF_MISSING", "1")

		ruletest.New(rt).
			WithRegistry(newTestRegistry()).
			ExpectNoAlert(time.Minute, "ExampleAlertDown")

		Expect(rt.errors).To(HaveLen(1))
		Expect(rt.errors[0]).To(ContainSubstring("cannot skip"))
	})
})
