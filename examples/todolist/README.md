# TodoList example

A small Connect-based service that exercises `go-eventually`'s DDD / Event
Sourcing primitives end-to-end. It serves as a real-world litmus test for
the library's API: if something feels awkward in this example, it probably
needs rethinking in the library.

## What this example demonstrates

- An aggregate root (`TodoList`) with a child entity (`Item`), built on
  `aggregate.BaseRoot` and `aggregate.RecordThat`.
- Commands (`CreateTodoList`, `AddTodoListItem`) implementing
  `command.Handler[Cmd]`, persisted through an
  `aggregate.EventSourcedRepository` backed by `event.NewInMemoryStore`.
- A query (`GetTodoList`) implementing `query.Handler[Q, R]`, reusing the
  same repository's `Get`.
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
cd examples/todolist
go run .
# Server listens on :8080 by default
```

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
- **One proto file per message** (the "1-1-1" Buf convention). Keeps each
  RPC contract isolated and easy to evolve.
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
- **`MarkTodoItemAsDone` / `MarkTodoItemAsPending` / `DeleteTodoItem`**
  exist in the domain but don't yet have command handlers nor wired
  Connect handlers. Adding them follows the same pattern as
  `AddTodoListItem`; left as an exercise / follow-up PR.

## Regenerating the protos

```sh
cd examples/todolist
buf generate proto
```

Committed output lives under `gen/`; regenerate after any proto change.
