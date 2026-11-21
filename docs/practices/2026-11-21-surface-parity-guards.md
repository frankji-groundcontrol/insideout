# Surface parity guards: request fields and response keys

Two tests make a drift between the API and its surfaces fail the build
instead of waiting for a review (review 5, `PARC-03` and `COMP-02`).

## Trigger

- You add or rename a JSON field in a request an API handler decodes, add a
  write route, or change what a CLI verb or MCP tool sends.
- You add a field to a roadmap node response, or change `RoadmapNode` in
  `client/lib/api/models_roadmap.dart`.

## Guard 1: every request field is reachable

[`server/internal/surfaceparity`](../../server/internal/surfaceparity/) holds
`TestEveryRequestFieldIsReachable`. It needs no database.

1. It parses `server/internal/api` and finds every route from the
   `mux.HandleFunc` calls, the handler each wraps, and the JSON keys that
   handler decodes through `httpx.DecodeJSON` (in the handler or a function it
   calls). A field whose type is a struct, a pointer to one or a slice of them
   adds its own keys below it: `items[].title`, `tree.title`. Nothing is
   listed by hand.
2. It calls every exported `apiclient.Client` method, and every method of a
   struct argument that runs on a `*Client` (the read-then-write `Apply`),
   by reflection against a server that only records. Each argument, and each
   field inside one (a struct argument's fields, and the fields of a struct a
   pointer, slice or field holds), is then zeroed in turn: a key that changes
   depends on the innermost one that changes it, so `items[].title` depends
   on `ProposeItem.Title`, not on the `items` argument. A key below a
   `json.RawMessage` the client forwards (the drafted `tree` and `children`)
   depends on that argument: the caller writes everything inside it.
3. It parses `cmd/insideout` (CLI) and `internal/mcpserver` (MCP). A surface
   reaches a key when it calls the client method that sends it and passes a
   non-zero argument there, or sets the struct field in an `apiclient.T{...}`
   literal or an assignment.
4. Every key not reached on a surface must be in `omissions`
   ([`omissions_test.go`](../../server/internal/surfaceparity/omissions_test.go))
   with a one-line reason. An entry marked `humanOnly` must match a `humanOnly`
   route in `agentRoutePolicy` (`server/internal/api/agent_scope.go`), and
   every field of a `humanOnly` route must be a `humanOnly` MCP omission. An
   entry that is reachable after all fails as stale.

When it fails: wire the field on the surface (the usual fix), or add an
omission whose reason a reviewer would accept. If wiring is too large for the
change at hand, record a dated issue under [`docs/issues/`](../issues/README.md)
and list the field as an omission that names the issue.

## Guard 2: every response key is read

`server/internal/api/testdata/roadmap_node_full.json` is a roadmap node
captured from the real API with every optional field set.

1. `TestRoadmapNodeWithEveryFieldKeepsItsCapturedShape`
   (`server/internal/api/roadmap_node_shape_test.go`, `DATABASE_URL`-gated)
   builds the same node and fails when the server's answer changes shape.
2. `client/test/api/roadmap_node_fixture_test.dart` feeds the fixture to
   `RoadmapNode.fromJson` through a map that records every key read, and fails
   on a key never read unless `_unreadOnPurpose` names it with a reason.

When the server shape test fails on purpose: re-capture the fixture from the
same sequence, then make the Dart test pass by reading the new key.

## Verify

```bash
(cd server && go test ./internal/surfaceparity/ -count=1)
(cd client && flutter test test/api/roadmap_node_fixture_test.dart)
```

To see the walk bite, add a `json` field to any request struct in
`server/internal/api`: the test names the route, the key and each surface.

## Failure signals

- A new route or field passes the walk with no CLI flag or MCP argument and no
  omission: the scan is broken (it also fails when it finds fewer than 50
  routes or 30 client routes).
- An omission reason that only says "not needed": it must say why the surface
  must not, or need not, reach the field.
