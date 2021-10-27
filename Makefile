GO_TEST_FLAGS := -race -v -coverpkg=./...
GOLANGCI_YML  ?= $(shell find ~+ -name .golangci.yml)

.PHONY: run-linter
run-linter:
	@find . -name "go.mod" | sed "s/\/go.mod//g" | xargs -I % bash -c 'echo -e "Checking: %"; cd %; golangci-lint run -c $(GOLANGCI_YML)'

.PHONY: postgres-tests
postgres-tests:
	@cd ./extension/postgres && go test $(GO_TEST_FLAGS) -coverprofile=postgres.out ./...
	@cd ./extension/postgres && go tool cover -func=postgres.out

.PHONY: eventually-tests
eventually-tests:
	@go test $(GO_TEST_FLAGS) -coverprofile=eventually.out ./...
	@go tool cover -func=eventually.out
