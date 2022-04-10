GO_TEST_FLAGS := -v -race -coverpkg=./... -coverprofile=coverage.out
GOLANGCI_YML  ?= $(shell find ~+ -name .golangci.yml)

.PHONY: run-linter
run-linter:
	@find . -name "go.mod" | sed "s/\/go.mod//g" | xargs -I % bash -c 'echo -e "Checking: %"; cd %; golangci-lint run -c $(GOLANGCI_YML)'

.PHONY: run-tests
run-tests:
	@find . -name "go.mod" | sed "s/\/go.mod//g" | xargs -I % bash -c 'echo -e "Testing: %"; cd %; go test ./... $(GO_TEST_FLAGS)'
