# TodoList example

A small Connect-based service that exercises `go-eventually`'s DDD / Event
Sourcing primitives end-to-end. It serves as a real-world litmus test for
the library's API: if something feels awkward in this example, it probably
needs rethinking in the library.

## What this example demonstrates

- An aggregate root (`todolist.TodoList`) with a child entity
  (`todolist.Item`), built on `aggregate.BaseRoot` and
  `aggregate.RecordThat`.
- Commands (`todolist.CreateCommand`, `todolist.AddItemCommand`)
  implementing `command.Handler[Cmd]`, persisted through an
  `aggregate.EventSourcedRepository` backed by `event.NewInMemoryStore`.
- A query (`todolist.GetQuery`) implementing `query.Handler[Q, R]`,
  reusing the same repository's `Get`.
- BDD-style test scenarios using `aggregate.Scenario`, `command.Scenario`,
  and (implicitly through the command scenarios) the event streaming
  plumbing.
- A Connect service exposing the above over HTTP/2 (h2c), with gRPC
  health + gRPC reflection wired up.

Because the repository internally streams events through the new
`message.Stream[event.Persisted]` iterator, any request that triggers
`AddTodoItem` (which loads the existing aggregate) exercises the iterator
end-to-end.

## Running

```sh
go run ./examples/todolist
# or from the example dir:
cd examples/todolist && go run .
# Server listens on :8080 by default
```

The example is a member of the repository's Go workspace (`go.work` at
repo root). The workspace is what resolves the `go-eventually` import
to the in-repo library code. The example's `go.mod` still carries a
`require github.com/get-eventually/go-eventually v0.4.0` line as a
nominal floor — `GOWORK=off` would fall back to that released version,
which is NOT what you want when evaluating in-progress library changes.

Hit it with a Connect client, `grpcurl`, or the built-in reflection:

```sh
grpcurl -plaintext localhost:8080 list
grpcurl -plaintext -d '{"todo_list_id":"...","title":"chores","owner":"me"}' \
  localhost:8080 todolist.v1.TodoListService/CreateTodoList
```

## Design choices worth noting

- **Commands return `google.protobuf.Empty`.** Clients generate IDs and
  pass them in; the server acknowledges. Idempotent on retries with the
  same ID.
- **Request/response messages colocated with the service** in
  `todo_list_service.proto`; domain messages live in their own files.
  Small service surface benefits more from colocation than from strict
  one-message-per-file splitting.
- **Connect only.** No HTTP/REST gateway, no `google.api.http`
  annotations. The Connect protocol itself already speaks gRPC, gRPC-Web,
  and Connect-over-HTTP; that's enough surface for an example.
- **In-memory store.** State is lost on restart. The example is about
  wiring, not persistence; swap `event.NewInMemoryStore()` in `main.go`
  for a `postgres.NewEventStore(...)` to get durability.
- **Error handling is example-grade.** Domain errors are mapped to Connect
  codes (`InvalidArgument`, `AlreadyExists`, `NotFound`, `Internal`) and
  the full error chain is propagated to the client. A real service would
  sanitize messages before they cross the wire.
- **Package by domain, not by layer.** Everything TodoList-related —
  aggregate, events, commands, queries, handlers, Connect transport,
  proto conversion — lives in a single `internal/todolist` package. Names
  like `CreateCommand`, `GetQuery`, `AddItemCommandHandler`,
  `ConnectServiceHandler`, and `ToProto` ride on top of the package
  prefix to keep call sites terse and the domain boundary obvious at a
  glance.

## Regenerating the protos

```sh
cd examples/todolist
buf generate
```

Committed output lives under `gen/`; regenerate after any proto change.
The buf configuration uses the v2 schema (see `buf.yaml` + `buf.gen.yaml`
at the module root).

## CI coverage

`make go.lint` and `make go.test` at the repo root iterate over every
Go workspace member (discovered from `go.work`). This example is
therefore lint-gated and test-gated on every PR — a library change
that breaks the example fails CI.
