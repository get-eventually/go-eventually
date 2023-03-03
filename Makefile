GO_TEST_FLAGS := -v -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
GOLANGCI_LINT_FLAGS ?=

.PHONY: run-linter
run-linter:
	@go work edit -json | jq -c -r '[.Use[].DiskPath] | map_values(. + "/...")[]' | xargs -I {} golangci-lint run $(GOLANGCI_LINT_FLAGS) {}

.PHONY: run-tests
run-tests:
	@find . -name "go.mod" | sed "s/\/go.mod//g" | xargs -I % bash -c 'echo -e "Testing: %"; cd %; go test ./... $(GO_TEST_FLAGS)'
