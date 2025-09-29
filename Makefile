GOLANDCI_LINT_VERSION ?= v1.56.0

all: goimport goclean lint test

sanity: goimport goclean check-diff

goclean:
	go version
	go fmt ./...
	go mod tidy -v
	go mod vendor
	git add -N vendor

goimport:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w -local="github.com/rhobs/operator-observability-toolkit"  $(shell find . -type f -name '*.go' ! -path "*/vendor/*" )

test:
	go test -v ./pkg/...

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANDCI_LINT_VERSION}
	golangci-lint run

check-diff:
	git difftool -y --trust-exit-code

e2e-functional:
	go test -v ./e2e/functional
