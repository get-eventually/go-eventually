MAKEFLAGS    += -s --always-make -C
SHELL        := bash
.SHELLFLAGS  := -Eeuo pipefail -c

ifndef DEBUG
.SILENT:
endif

GOLANGCI_LINT_FLAGS ?= -v
GO_TEST_FLAGS       ?= -v -cover -covermode=atomic -coverprofile=coverage.txt -coverpkg=./...

go.lint:
	golangci-lint run $(GOLANGCI_LINT_FLAGS)

go.test:
	go test $(GO_TEST_FLAGS) ./...

go.test.unit:
	go test -short $(GO_TEST_FLAGS) ./...
