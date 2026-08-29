# A01 — The request-tracker event channel has no consumer in headless mode, and the proxy permanently stalls

**Severity:** Critical (production-breaking for the non-interactive mode)
**Area:** Observability pipeline / server core
**Status:** Reproduced

---

## 1. What is wrong with the current approach

`Server.handleRequest` publishes 2–3 events per request onto a single
`RequestTracker` channel:

- [`internal/server/server.go:186`](../internal/server/server.go#L186) — request started
- [`internal/server/server.go:197`](../internal/server/server.go#L197) — prefix update (emitted from a context callback, once per handler that calls `infra.WithPrefix`)
- [`internal/server/server.go:211`](../internal/server/server.go#L211) — request finished

`Emit` is an **unconditional blocking send** on a channel with a fixed buffer of
1000:

```go
// internal/server/request_tracker.go:29,46
events: make(chan RequestEvent, requestEventsBufferSize) // 1000

func (t *RequestTracker) Emit(event RequestEvent) {
	t.events <- event
}
```

The channel has exactly one consumer in the whole codebase —
[`internal/uncors_app/app.go:93`](../internal/uncors_app/app.go#L93), the
BubbleTea TUI. In non-interactive mode
([`internal/cli/run_non_ineractive.go`](../internal/cli/run_non_ineractive.go))
**nothing ever reads the channel**. `server.RequestPrinter`
([`internal/server/request_printer.go:7`](../internal/server/request_printer.go#L7))
was clearly written to be that consumer, but it is dead code — no call site exists.

Consequence: after roughly `1000 / 3 ≈ 333` requests, the buffer is full and
every subsequent request goroutine blocks forever inside `Emit`, holding its
connection open. The proxy stops serving traffic and never recovers. Because
`log` output is discarded by default (see [A13](A13-logging-and-output-are-three-parallel-systems.md))
there is no diagnostic whatsoever — it simply hangs.

### Reproduction

A temporary integration test (600 sequential `GET` requests through a headless
proxy, using the existing `testing/integration` harness) stalls and never
completes:

```
=== RUN   TestZZHeadlessFlood
    zz_flood_test.go:49: HEADLESS PROXY STALLED - requests stopped completing
--- FAIL: TestZZHeadlessFlood (30.07s)
```

The existing integration suite never catches this because each `Env` issues only
a handful of requests before teardown.

### Why it has gone unnoticed

- The default mode is interactive (`--interactive` defaults to `true`,
  [`internal/config/flags.go:20`](../internal/config/flags.go#L20)), and the TUI
  drains the channel.
- Headless mode is what CI, Docker, `npx uncors` in scripts, and
  `tests/integration` use — precisely the automated contexts where nobody is
  watching the terminal to notice the hang.

## 2. Why it is an architectural problem

This is not a missing line of code; it is a **missing invariant in the design**.
The server core was given a hard, unbuffered-in-practice dependency on a
*presentation* concern:

- The request pipeline's liveness depends on somebody, somewhere, having wired up
  a UI consumer. Nothing in the type system or the constructor enforces that.
- A telemetry/observability channel must never be able to apply backpressure to
  the data path. Dropping an event is always the correct failure mode for a
  developer-tool activity log; blocking a proxied HTTP request is never correct.
- The producer emits 2–3 events per request when the consumer only cares about
  the terminal one (`handleRequestEvent` in
  [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) returns
  immediately unless `event.Done && event.Data != nil`). The channel budget is
  spent on events nobody reads.
- The `Close()` method on `RequestTracker` closes the channel while producers may
  still be sending — a `send on closed channel` panic waiting to happen. (It is
  currently never called, which is the only reason it has not fired.)

## 3. What the recommended approach is instead

**Step 1 — make emission non-blocking and event-count-per-request bounded.**

```go
func (t *RequestTracker) Emit(event RequestEvent) {
	select {
	case t.events <- event:
	default:
		t.dropped.Add(1) // observable, never blocking
	}
}
```

**Step 2 — make the sink explicit and always present.** Model the tracker as a
*sink interface* that the runtime always supplies, rather than a channel that
somebody may or may not drain:

```go
type RequestSink interface {
	Started(RequestEvent)
	Finished(RequestEvent)
}
```

- Headless mode injects a sink that writes straight to `contracts.Output`
  (i.e. resurrect `RequestPrinter`, but as the *default* implementation, not an
  optional goroutine).
- Interactive mode injects the channel-backed sink the TUI reads.
- Tests inject a no-op sink.

With a sink interface, "no consumer" is not a representable state.

**Step 3 — stop emitting the intermediate prefix event.** Collapse the prefix
update into the terminal event (the server already tracks `lastPrefix` in a
local variable at [`internal/server/server.go:193`](../internal/server/server.go#L193)).
That reduces channel traffic by ~3× and removes the closure-over-local-variable
data race between the prefix callback and the final `Emit`.

**Step 4 — add a regression test.** A headless load test (e.g. 5 000 requests
through the `testing/integration` harness with a deadline) belongs in the
integration suite. The current suite's request counts are far below the failure
threshold, which is why a total service stall ships green.

## 4. Why the proposed approach is better

- **The data path can no longer be stalled by the UI.** Backpressure from a
  terminal renderer to an HTTP proxy is inverted control; a `select`/`default`
  drop restores the correct direction.
- **The failure mode becomes visible instead of silent.** A dropped-event counter
  can be printed on shutdown ("N activity lines dropped"), which is honest and
  harmless. A hung proxy is neither.
- **Both run modes are forced to be complete.** Today the two modes were built
  independently and one of them simply forgot half the wiring (see
  [A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md)); an interface
  parameter makes the omission a compile error.
- Removes `RequestPrinter` from the dead-code inventory
  ([A20](A20-dead-code-and-vestigial-abstractions.md)) by giving it its intended job.

## 5. Trade-offs and migration considerations

- **Dropped events change behaviour under load.** In interactive mode a very
  fast burst may now omit some lines from the history widget rather than slowing
  the proxy down. This is the correct trade for a dev tool, but it should be
  stated in the docs and ideally surfaced ("… and 42 more").
- **Ordering.** With a non-blocking send, a `Finished` event can be dropped while
  its `Started` event was kept, leaving a "stuck" row in the tracker widget. Give
  each event the request ID (already present) and have the widget expire rows
  older than a threshold, or drop `Started`/`Finished` pairs atomically by only
  emitting the terminal event when the buffer is under pressure.
- **`RequestTracker.Close()` must go or be made safe.** Closing a channel from
  the consumer side while producers are live is unsound; prefer never closing it
  and letting GC reclaim it at process exit, or gate sends behind an
  `atomic.Bool` that `Close` flips first.
- Migration is small and self-contained: `internal/server/request_tracker.go`,
  three call sites in `internal/server/server.go`, plus one constructor argument
  in `internal/di`.

## 6. Code references

| What | Where |
| --- | --- |
| Blocking `Emit` | [`internal/server/request_tracker.go:46`](../internal/server/request_tracker.go#L46) |
| Buffer size (1000) | [`internal/server/request_tracker.go:10`](../internal/server/request_tracker.go#L10) |
| 3 emits per request | [`internal/server/server.go:186`](../internal/server/server.go#L186), [`:197`](../internal/server/server.go#L197), [`:211`](../internal/server/server.go#L211) |
| Only consumer (TUI) | [`internal/uncors_app/app.go:93`](../internal/uncors_app/app.go#L93), [`internal/uncors_app/app.go` `watchEventsCmd`](../internal/uncors_app/app.go) |
| Dead intended consumer | [`internal/server/request_printer.go:7`](../internal/server/request_printer.go#L7) |
| Headless run path with no consumer | [`internal/cli/run_non_ineractive.go:22`](../internal/cli/run_non_ineractive.go#L22) |
| Unsafe `Close` | [`internal/server/request_tracker.go:42`](../internal/server/request_tracker.go#L42) |
