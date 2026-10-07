package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// failingReader yields some bytes and then fails, simulating a download
// interrupted midway.
type failingReader struct{ served bool }

func (f *failingReader) Read(p []byte) (int, error) {
	if f.served {
		return 0, errors.New("connection reset")
	}

	f.served = true
	n := copy(p, []byte("partial binary content"))

	return n, nil
}

var _ = Describe("assetName", func() {
	It("should build the linux amd64 asset name", func() {
		name, err := assetName("3.15.0", "linux", "amd64")
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal("prometheus-3.15.0.linux-amd64.tar.gz"))
	})

	It("should build the darwin arm64 asset name", func() {
		name, err := assetName("3.15.0", "darwin", "arm64")
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal("prometheus-3.15.0.darwin-arm64.tar.gz"))
	})

	It("should map arm to armv7", func() {
		name, err := assetName("3.15.0", "linux", "arm")
		Expect(err).ToNot(HaveOccurred())
		Expect(name).To(Equal("prometheus-3.15.0.linux-armv7.tar.gz"))
	})

	It("should reject windows with actionable guidance", func() {
		_, err := assetName("3.15.0", "windows", "amd64")
		Expect(err).To(MatchError(ContainSubstring("PROMTOOL")))
	})

	It("should reject an unknown architecture", func() {
		_, err := assetName("3.15.0", "linux", "sparc")
		Expect(err).To(MatchError(ContainSubstring("sparc")))
	})
})

var _ = Describe("verifySHA256", func() {
	payload := []byte("hello")
	sum := sha256.Sum256(payload)
	line := hex.EncodeToString(sum[:]) + "  prometheus-3.15.0.linux-amd64.tar.gz"

	It("should accept a matching checksum", func() {
		Expect(verifySHA256(payload, line+"\n", "prometheus-3.15.0.linux-amd64.tar.gz")).To(Succeed())
	})

	It("should reject a mismatched checksum", func() {
		err := verifySHA256([]byte("tampered"), line+"\n", "prometheus-3.15.0.linux-amd64.tar.gz")
		Expect(err).To(MatchError(ContainSubstring("checksum mismatch")))
	})

	It("should reject a missing entry", func() {
		err := verifySHA256(payload, line+"\n", "prometheus-3.15.0.darwin-arm64.tar.gz")
		Expect(err).To(MatchError(ContainSubstring("no checksum entry")))
	})
})

var _ = Describe("extractPromtool", func() {
	buildArchive := func(entries map[string]string) []byte {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)

		for name, body := range entries {
			Expect(tw.WriteHeader(&tar.Header{
				Name:     name,
				Mode:     0o755,
				Size:     int64(len(body)),
				Typeflag: tar.TypeReg,
			})).To(Succeed())
			_, err := tw.Write([]byte(body))
			Expect(err).ToNot(HaveOccurred())
		}

		Expect(tw.Close()).To(Succeed())
		Expect(gz.Close()).To(Succeed())

		return buf.Bytes()
	}

	It("should extract promtool and make it executable", func() {
		archive := buildArchive(map[string]string{
			"prometheus-3.15.0.linux-amd64/prometheus": "not this",
			"prometheus-3.15.0.linux-amd64/promtool":   "the binary",
		})

		dest := filepath.Join(GinkgoT().TempDir(), "promtool")
		Expect(extractPromtool(bytes.NewReader(archive), dest)).To(Succeed())

		got, err := os.ReadFile(dest)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(got)).To(Equal("the binary"))

		info, err := os.Stat(dest)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode().Perm() & 0o100).ToNot(BeZero())
	})

	// A truncated 0755 file at the make target's path would be treated as
	// up-to-date on the next run, so the install must be all-or-nothing.
	It("should leave no file behind when the copy fails partway", func() {
		dest := filepath.Join(GinkgoT().TempDir(), "promtool")

		err := writeBinary(&failingReader{}, dest)
		Expect(err).To(HaveOccurred())
		Expect(dest).ToNot(BeAnExistingFile())
	})

	It("should replace an existing non-executable file with an executable one", func() {
		dest := filepath.Join(GinkgoT().TempDir(), "promtool")
		Expect(os.WriteFile(dest, []byte("stale"), 0o600)).To(Succeed())

		archive := buildArchive(map[string]string{
			"prometheus-3.15.0.linux-amd64/promtool": "fresh",
		})
		Expect(extractPromtool(bytes.NewReader(archive), dest)).To(Succeed())

		info, err := os.Stat(dest)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode().Perm() & 0o100).ToNot(BeZero())

		got, err := os.ReadFile(dest)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(got)).To(Equal("fresh"))
	})

	It("should error when the archive has no promtool", func() {
		archive := buildArchive(map[string]string{
			"prometheus-3.15.0.linux-amd64/prometheus": "nope",
		})

		dest := filepath.Join(GinkgoT().TempDir(), "promtool")
		Expect(extractPromtool(bytes.NewReader(archive), dest)).To(MatchError(ContainSubstring("not found in archive")))
	})
})
