# Changelog

## [0.2.1](https://github.com/get-eventually/go-eventually/compare/postgres-v0.2.0...postgres-v0.2.1) (2023-09-22)


### Features

* implement Firestore event.Store interface ([#136](https://github.com/get-eventually/go-eventually/issues/136)) ([5e1c10c](https://github.com/get-eventually/go-eventually/commit/5e1c10c04d5a51b89da7ba146665882fdfeba237))
* **postgres:** update to pgx/v5 and refactor RunMigrations ([#137](https://github.com/get-eventually/go-eventually/issues/137)) ([a74cc5d](https://github.com/get-eventually/go-eventually/commit/a74cc5d818ba390bc3b0ec19cee94a9c8d9de4f4))


### Bug Fixes

* golangci-lint linter execution ([#88](https://github.com/get-eventually/go-eventually/issues/88)) ([bff3e52](https://github.com/get-eventually/go-eventually/commit/bff3e5219f413465268811a6f7296a5f21ea122a))
* **postgres:** use eventually_schema_migrations table for migrations ([#87](https://github.com/get-eventually/go-eventually/issues/87)) ([4886b08](https://github.com/get-eventually/go-eventually/commit/4886b082d33db4741832bba12623bfd668790913))
* **postgres:** use pgxpool.Pool instead of pgx.Conn for db communication ([#83](https://github.com/get-eventually/go-eventually/issues/83)) ([076e4e8](https://github.com/get-eventually/go-eventually/commit/076e4e86145407b81caa03bc900babe61a584917))
* **postgres:** use SERIALIZABLE tx isolation level for AggregateRepository.Save ([#139](https://github.com/get-eventually/go-eventually/issues/139)) ([0cde29d](https://github.com/get-eventually/go-eventually/commit/0cde29d98de6a1cb38ec250d9dd822af6a5de477))
