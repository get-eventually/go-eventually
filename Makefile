.PHONY: postgres-tests
postgres-tests:
	@cd ./eventstore/postgres && go test -race -v -coverprofile=postgres.out ./...
	@cd ./eventstore/postgres && go tool cover -func=postgres.out

.PHONY: eventually-tests
eventually-tests:
	@go test -short -race -v -coverprofile=eventually.out ./...
	@go tool cover -func=eventually.out
