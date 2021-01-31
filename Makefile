.PHONY: unit-tests
unit-tests:
	go test -short -race -v -coverprofile=unitcov.out ./...
	go tool cover -func=unitcov.out
