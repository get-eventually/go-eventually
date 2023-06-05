module github.com/get-eventually/go-eventually/examples/todolist

go 1.18

require (
	github.com/bufbuild/connect-go v1.5.2
	github.com/bufbuild/connect-grpchealth-go v1.0.0
	github.com/bufbuild/connect-grpcreflect-go v1.0.0
	github.com/get-eventually/go-eventually/core v0.0.0-20230301093954-efadfc924ad7
	github.com/google/uuid v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	go.uber.org/zap v1.24.0
	golang.org/x/net v0.7.0
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/get-eventually/go-eventually/core => ../../core
