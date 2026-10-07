package ruletest

import (
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func itoa(i int) string { return strconv.Itoa(i) }

var _ = Describe("parseJUnit", func() {
	It("should parse a passing run", func() {
		const in = `<testsuites><testsuite name="tests.yaml" tests="1" failures="0" errors="0">` +
			`<testcase name="operator is down"></testcase></testsuite></testsuites>`

		res, err := parseJUnit([]byte(in))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Scenarios).To(HaveLen(1))
		Expect(res.Scenarios[0].Name).To(Equal("operator is down"))
		Expect(res.Failed()).To(BeFalse())
	})

	It("should parse a failing run and keep the failure detail", func() {
		const in = `<testsuites><testsuite name="tests.yaml" tests="1" failures="1" errors="0">` +
			`<testcase name="operator is down"><failure>exp: 1E+00&#xA;got: 0E+00</failure></testcase>` +
			`</testsuite></testsuites>`

		res, err := parseJUnit([]byte(in))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeTrue())
		Expect(res.Scenarios[0].Failures).To(HaveLen(1))
		Expect(res.Scenarios[0].Failures[0]).To(ContainSubstring("got: 0E+00"))
	})

	// Review Focus 4: <error> means our generated YAML was bad, which is a bug
	// in this package rather than a failed assertion.
	It("should treat an error element as a harness error", func() {
		const in = `<testsuites><testsuite name="tests.yaml" tests="1" failures="0" errors="1">` +
			`<testcase name="unknown"><error>yaml: line 2: mapping values are not allowed</error></testcase>` +
			`</testsuite></testsuites>`

		_, err := parseJUnit([]byte(in))
		Expect(err).To(MatchError(ContainSubstring("mapping values are not allowed")))
	})

	It("should return an error for malformed xml", func() {
		_, err := parseJUnit([]byte("<testsuites"))
		Expect(err).To(HaveOccurred())
	})

	// C1: promtool is not a stable API and PROMTOOL/PATH can point at any
	// version. A report whose shape we do not recognise must never read as
	// "nothing failed".
	It("should treat a skipped testcase as a harness error", func() {
		const in = `<testsuites><testsuite name="tests.yaml" tests="1" failures="0" errors="0" skipped="1">` +
			`<testcase name="s"><skipped/></testcase></testsuite></testsuites>`

		_, err := parseJUnit([]byte(in))
		Expect(err).To(MatchError(ContainSubstring("skipped")))
	})
})

var _ = Describe("runPromtool", func() {
	// writeStub creates a fake promtool that writes the given junit body and
	// exits with the given code.
	writeStub := func(dir, junitBody string, exitCode int) string {
		bin := filepath.Join(dir, "fake-promtool")
		script := "#!/bin/sh\n" +
			"for a in \"$@\"; do case \"$a\" in --junit=*) out=\"${a#--junit=}\";; esac; done\n" +
			"cat > \"$out\" <<'XMLEOF'\n" + junitBody + "\nXMLEOF\n" +
			"exit " + itoa(exitCode) + "\n"
		Expect(os.WriteFile(bin, []byte(script), 0o700)).To(Succeed())

		return bin
	}

	// C1: a promtool whose report we cannot interpret, exiting non-zero, must
	// not come back as a clean empty pass.
	It("should fail when promtool exits non-zero but reports no failures", func() {
		dir := GinkgoT().TempDir()
		bin := writeStub(dir, `<testsuites><suite name="x"><case name="y"><failure>BOOM</failure></case></suite></testsuites>`, 1)

		_, err := runPromtool(bin, dir, filepath.Join(dir, "tests.yaml"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exit"))
	})

	It("should succeed when promtool exits zero with a passing report", func() {
		dir := GinkgoT().TempDir()
		bin := writeStub(dir, `<testsuites><testsuite name="tests.yaml"><testcase name="s"></testcase></testsuite></testsuites>`, 0)

		res, err := runPromtool(bin, dir, filepath.Join(dir, "tests.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Scenarios).To(HaveLen(1))
		Expect(res.Failed()).To(BeFalse())
	})

	It("should report a genuine assertion failure without erroring", func() {
		dir := GinkgoT().TempDir()
		bin := writeStub(dir, `<testsuites><testsuite name="tests.yaml"><testcase name="s"><failure>exp 1 got 0</failure></testcase></testsuite></testsuites>`, 1)

		res, err := runPromtool(bin, dir, filepath.Join(dir, "tests.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Failed()).To(BeTrue())
	})
})

var _ = Describe("resolvePromtool", func() {
	It("should use PROMTOOL when it points at an existing file", func() {
		dir := GinkgoT().TempDir()
		bin := filepath.Join(dir, "promtool")
		Expect(os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700)).To(Succeed())

		GinkgoT().Setenv(envPromtool, bin)

		got, err := resolvePromtool()
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal(bin))
	})

	It("should return an error when PROMTOOL points at a missing file", func() {
		GinkgoT().Setenv(envPromtool, filepath.Join(GinkgoT().TempDir(), "nope"))

		_, err := resolvePromtool()
		Expect(err).To(MatchError(ContainSubstring("PROMTOOL")))
	})
})
