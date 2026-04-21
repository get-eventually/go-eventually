MAKEFLAGS    += -s --always-make -C
SHELL        := bash
.SHELLFLAGS  := -Eeuo pipefail -c

ifndef DEBUG
.SILENT:
endif

GOLANGCI_LINT_FLAGS ?= -v
GO_TEST_FLAGS       ?= -v -cover -covermode=atomic -coverprofile=coverage.txt -coverpkg=./...

# GO_MODULES is the list of directories containing a go.mod, discovered via
# git so untracked scratch modules (e.g. vendored fixtures) never sneak in.
# The repo root lists itself as "." so the root module is always covered.
GO_MODULES := $(sort $(dir $(shell git ls-files '*go.mod')))

# GOLANGCI_LINT_CONFIG points each nested module's linter at the root config.
# Every module is linted with the same rules we apply to the library.
GOLANGCI_LINT_CONFIG := $(abspath .golangci.yaml)

# run_in_modules runs a shell command in each Go module. The first argument
# is a human-readable label for logs; the second is the command itself,
# which runs with $$mod pointing at the module directory.
define run_in_modules
	set -e; \
	for mod in $(GO_MODULES); do \
		echo "==> $(1) ($$mod)"; \
		( cd "$$mod" && $(2) ); \
	done
endef

go.lint:
	$(call run_in_modules,golangci-lint run,golangci-lint run --config $(GOLANGCI_LINT_CONFIG) $(GOLANGCI_LINT_FLAGS))

go.test:
	$(call run_in_modules,go test,go test $(GO_TEST_FLAGS) ./...)

go.test.unit:
	$(call run_in_modules,go test -short,go test -short $(GO_TEST_FLAGS) ./...)

go.build:
	$(call run_in_modules,go build,go build ./...)

go.mod.update:
	$(call run_in_modules,go get -u + go mod tidy,go get -u ./... && go mod tidy)
