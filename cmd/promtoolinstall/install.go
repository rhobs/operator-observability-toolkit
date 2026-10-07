package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// archMap maps GOARCH values to the architecture names used in Prometheus
// release asset names.
var archMap = map[string]string{
	"amd64":   "amd64",
	"arm64":   "arm64",
	"386":     "386",
	"ppc64le": "ppc64le",
	"s390x":   "s390x",
	"riscv64": "riscv64",
	"arm":     "armv7",
}

// assetName builds the release asset name for a platform. Note that the
// release tag uses the 3.y.z form, not the v0.3yz Go module version.
func assetName(version, goos, goarch string) (string, error) {
	if goos == "windows" {
		return "", errors.New("windows releases ship as a zip and are not supported by this installer; " +
			"download promtool manually and set PROMTOOL to its path")
	}

	arch, ok := archMap[goarch]
	if !ok {
		return "", fmt.Errorf("unsupported architecture %q", goarch)
	}

	return fmt.Sprintf("prometheus-%s.%s-%s.tar.gz", version, goos, arch), nil
}

// verifySHA256 checks data against the entry for asset in a sha256sums.txt body.
func verifySHA256(data []byte, sums, asset string) error {
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != asset {
			continue
		}

		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != fields[0] {
			return fmt.Errorf("checksum mismatch for %s", asset)
		}

		return nil
	}

	return fmt.Errorf("no checksum entry for %s", asset)
}

// extractPromtool finds the promtool binary in a gzipped tar and writes it to dest.
func extractPromtool(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("reading gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != "promtool" {
			continue
		}

		return writeBinary(tr, dest)
	}

	return errors.New("promtool not found in archive")
}

// writeBinary writes r to dest atomically.
//
// The install must be all-or-nothing: a truncated file left at a Makefile
// target's path is treated as up-to-date on the next run, so an interrupted
// download would otherwise be silently reused as a corrupt binary. Writing to
// a sibling temp file and renaming also replaces an existing file's mode,
// which O_TRUNC alone does not do.
func writeBinary(r io.Reader, dest string) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(dest)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpName := tmp.Name()

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmp, r); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}

	if err := tmp.Chmod(0o755); err != nil {
		return fmt.Errorf("setting mode on %s: %w", dest, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("installing %s: %w", dest, err)
	}

	return nil
}
