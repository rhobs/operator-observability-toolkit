// Command promtoolinstall downloads a pinned promtool binary from the
// Prometheus releases and verifies it against the published checksums.
//
// promtool cannot be installed with `go install`, because the
// prometheus/prometheus module contains replace directives.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const releaseBaseURL = "https://github.com/prometheus/prometheus/releases/download"

func main() {
	version := flag.String("version", "", "Prometheus release version, for example 3.15.0")
	out := flag.String("o", "promtool", "output path for the promtool binary")
	flag.Parse()

	if *version == "" {
		fmt.Fprintln(os.Stderr, "error: -version is required")
		os.Exit(1)
	}

	if err := install(*version, *out); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("installed promtool %s to %s\n", *version, *out)
}

func install(version, out string) error {
	asset, err := assetName(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	base := fmt.Sprintf("%s/v%s", releaseBaseURL, version)

	tarball, err := download(base + "/" + asset)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset, err)
	}

	sums, err := download(base + "/sha256sums.txt")
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}

	if err := verifySHA256(tarball, string(sums), asset); err != nil {
		return err
	}

	return extractPromtool(bytes.NewReader(tarball), out)
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
