module github.com/get-eventually/go-eventually/eventstore/postgres

go 1.15

replace github.com/get-eventually/go-eventually => ../../

require (
	github.com/get-eventually/go-eventually v0.0.0-00010101000000-000000000000
	github.com/golang-migrate/migrate v3.5.4+incompatible
	github.com/lib/pq v1.10.1
	github.com/stretchr/testify v1.7.0
)
