PKG = github.com/get-eventually/go-eventually
GO_TEST_FLAGS := -race -v

.PHONY: postgres-tests
postgres-tests:
	@cd ./eventstore/postgres && go test $(GO_TEST_FLAGS) -coverprofile=postgres.out ./...
	@cd ./eventstore/postgres && go tool cover -func=postgres.out

.PHONY: eventually-tests
eventually-tests:
	@go test $(GO_TEST_FLAGS) -coverprofile=eventually.out ./...
	@go tool cover -func=eventually.out
