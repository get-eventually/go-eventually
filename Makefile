.PHONY: tests
tests:
	go test -race -v -coverprofile=cov.out ./...
	go tool cover -func=cov.out

.PHONY: unit-tests
unit-tests:
	go test -short -race -v -coverprofile=unitcov.out ./...
	go tool cover -func=unitcov.out
