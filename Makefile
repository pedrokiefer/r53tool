GOLANGLINT_INSTALLED_VERSION := $(shell golangci-lint version 2>/dev/null | sed -ne 's/.*version\ \([0-9]*\.[0-9]*\.[0-9]*\).*/\1/p')
GOLANG_LINT_VERSION := 2.4.0

lint:
ifneq (${GOLANG_LINT_VERSION}, ${GOLANGLINT_INSTALLED_VERSION})
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${GOLANG_LINT_VERSION}
endif
	golangci-lint run

build: 
	goreleaser release --snapshot --clean

test:
	go test ./... -v

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html