# A13 — Three parallel output systems, and by default all diagnostics are discarded

**Severity:** High (observability)
**Area:** Logging / diagnostics

---

## 1. What is wrong with the current approach

There are three unrelated ways for code to say something, and they do not
interoperate:

1. **`contracts.Output`** — the user-facing channel, implemented by
   `tui.CliOutput` ([`internal/tui/output.go:38`](../internal/tui/output.go#L38)) and
   by `uncorsapp.tuiOutput` ([`internal/uncors_app/output.go:11`](../internal/uncors_app/output.go#L11)).
2. **The standard `log` package** — used directly for internal diagnostics in at
   least eight files (`static`, `mock`, `har/writer`, `server/host_cert_manager`,
   `config/watcher`, `version`, `uncors_app/app`, `cli/run_non_ineractive`).
3. **`contracts.Logger`** — a 10-method interface that nothing implements and
   nothing consumes ([`internal/contracts/logger.go:3`](../internal/contracts/logger.go#L3)).

The critical problem is what happens to (2):

```go
// internal/infra/loggings.go:15-28
func SetupLogging() {
	path := os.Getenv("UNCORS_LOGGING")
	if path == "" {
		log.SetOutput(io.Discard)      // ← default: everything is thrown away
		return
	}
	logFile, err := os.OpenFile(filepath.Clean(path), logFileFlags, logFilePerm)
	if err != nil {
		log.SetOutput(io.Discard)      // ← and silently thrown away on failure too
		return
	}
	log.SetOutput(logFile)
}
```

`SetupLogging()` is the very first statement in `main`
([`main.go:19`](../main.go#L19)). So unless the user happens to know about an
undocumented environment variable, **every internal diagnostic in the program is
silently dropped**, including:

- `"ERROR: Static handler error: %v, url: %s"` ([`internal/handler/static/middleware.go:44`](../internal/handler/static/middleware.go#L44))
- `"ERROR: Mock handler error: …"` ([`internal/handler/mock/handler.go:41`](../internal/handler/mock/handler.go#L41))
- `"har: cannot write temp file …"` and the other three HAR write failures
  ([`internal/handler/har/writer.go:130`](../internal/handler/har/writer.go#L130)) — so a
  HAR recording that never gets written looks identical to one that works
- `"config watcher error: %v"` ([`internal/config/watcher.go:117`](../internal/config/watcher.go#L117)) —
  so a watcher that dies mid-session silently stops reloading
- every failure inside the version checker ([`internal/version/check_new_version.go`](../internal/version/check_new_version.go))
- **`net/http`'s own server errors**: `http.Server.ErrorLog` is nil in
  `PortListener` ([`internal/server/port_listener.go:11`](../internal/server/port_listener.go#L11)),
  so it falls back to the standard logger — meaning **TLS handshake failures and
  panics recovered inside handlers are discarded**. A user whose browser rejects
  the dev certificate sees a connection error and uncors prints nothing at all.

Meanwhile the `contracts.Output` side has its own problems:

- `CliOutput.print` **panics if the write fails**
  ([`internal/tui/output.go:152`](../internal/tui/output.go#L152)) — a broken pipe
  (`uncors | head`) crashes the process.
- There is no level filtering. `Info`, `Warn`, `Error` differ only by a coloured
  label; there is no `--log-level`, no `--quiet`, no `--verbose`.
- `Output` is simultaneously an `io.Writer`, a leveled reporter, a
  request-line renderer and a factory (`NewPrefixOutput`), so any component that
  wants to print one line must accept all four roles
  ([A12](A12-package-boundaries-and-layering-violations.md)).
- Prefix propagation runs through a `context.Value` carrying a
  `func(string)` callback ([`internal/infra/prefix.go:9`](../internal/infra/prefix.go#L9)),
  mutating a captured local in the server goroutine
  ([`internal/server/server.go:193`](../internal/server/server.go#L193)) — a
  side-channel where a struct field would do.

## 2. Why it is an architectural problem

- **A developer tool whose entire purpose is diagnosing HTTP problems discards
  its own diagnostics by default.** This inverts the tool's value proposition:
  when uncors misbehaves, the user has strictly less information than if they had
  used `curl`.
- **Silence is indistinguishable from success.** Dropped HAR entries, failed HAR
  writes, dead config watchers, and TLS handshake rejections all present as
  "nothing happened". This is the mechanism that let [A01](A01-request-tracker-deadlocks-headless-mode.md)
  (a total service stall) ship undetected.
- **Two output paths mean two policies.** Whether a message reaches the user
  depends on which API the author happened to use, not on its importance. There is
  no way to route "all errors" anywhere.
- **The `UNCORS_LOGGING` environment variable is the only diagnostic control and
  it is documented nowhere** — not in `README.md`, not in `docs/Troubleshooting.md`
  (which instead tells users to pass a nonexistent `--debug` flag —
  [D01](D01-debug-flag-does-not-exist.md)), only in `CLAUDE.md`.

## 3. What the recommended approach is instead

**One logging façade, two sinks, level-controlled.**

1. **Adopt `log/slog`** (standard library since Go 1.21; the module already
   targets a much newer Go). A single `*slog.Logger` is created in `main` and
   injected — or, pragmatically, set as `slog.Default()` — and every current
   `log.Printf` becomes `slog.Debug/Info/Warn/Error` with structured fields:

```go
slog.Error("har write failed", "path", w.path, "err", err)
```

2. **Default to `stderr` at `warn` level, not to `io.Discard`.** Errors must
   always be visible. Add flags:

```
--log-level  debug|info|warn|error   (default: info)
--log-file   PATH                    (default: stderr)
--quiet                              (errors only)
```

   and keep `UNCORS_LOGGING` as a documented alias for `--log-file`.

3. **Route the TUI's stderr correctly.** In interactive mode stderr would corrupt
   the alt-screen, so the TUI presenter installs an `slog.Handler` that forwards
   records into the event stream ([A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md))
   instead of writing to the terminal. That is the one legitimate reason the
   current code redirects logs — but the fix is a handler, not a global discard.

4. **Wire `http.Server.ErrorLog`**:

```go
Server: http.Server{ ErrorLog: slog.NewLogLogger(handler, slog.LevelWarn), ... }
```

   so TLS and connection-level failures surface.

5. **Separate "user-facing UI output" from "diagnostics".** `contracts.Output`
   keeps the boxes, the logo, the request lines — the deliberate presentation.
   `slog` carries everything else. A component should never have to choose.

6. **Make `CliOutput` non-panicking**: swallow or record write errors; a broken
   pipe is normal.

## 4. Why the proposed approach is better

- **Failures become visible by default**, which is the single highest-leverage
  observability change available here — it is what would have surfaced
  [A01](A01-request-tracker-deadlocks-headless-mode.md),
  [A02](A02-di-container-leaks-resources-on-every-config-reload.md) and silent HAR
  write failures during development.
- **Levels give users a real troubleshooting story**, which the documentation is
  already (incorrectly) promising.
- **Structured records enable `--output=json`** for CI consumers without a second
  formatting path.
- **`slog` is standard**, so no new dependency, and handlers compose (the TUI
  handler above is ~20 lines).
- Deleting `contracts.Logger` and the `UNCORS_LOGGING`-only mechanism removes two
  of the three parallel systems.

## 5. Trade-offs and migration considerations

- **Turning logging on by default will surface noise that has been hidden for a
  long time.** Expect an initial pass to reclassify messages (most current
  `log.Printf` calls are `Debug`, a few are genuinely `Error`). Do that
  reclassification deliberately rather than switching everything to `Info`.
- **The TUI must be migrated at the same time**, or interactive mode will have its
  display corrupted by stderr writes. Ship the slog handler and the default change
  together.
- **Log volume in the proxy hot path matters.** Per-request logging should be
  `Debug` and guarded by level checks; avoid formatting strings that are then
  discarded.
- `helpers.SanitizeLogValue` ([`internal/helpers/log.go:8`](../internal/helpers/log.go#L8))
  becomes unnecessary for structured fields (slog handles escaping), but keep it
  for any remaining text output.
- Update `docs/Troubleshooting.md` in the same change so the documented flags exist
  ([D01](D01-debug-flag-does-not-exist.md)).

## 6. Code references

| What | Where |
| --- | --- |
| Logging discarded by default | [`internal/infra/loggings.go:15`](../internal/infra/loggings.go#L15), [`main.go:19`](../main.go#L19) |
| `http.Server.ErrorLog` never set | [`internal/server/port_listener.go:11`](../internal/server/port_listener.go#L11) |
| Discarded handler errors | [`internal/handler/static/middleware.go:44`](../internal/handler/static/middleware.go#L44), [`internal/handler/mock/handler.go:41`](../internal/handler/mock/handler.go#L41) |
| Discarded HAR write failures | [`internal/handler/har/writer.go:130`](../internal/handler/har/writer.go#L130) |
| Discarded watcher failures | [`internal/config/watcher.go:117`](../internal/config/watcher.go#L117) |
| Output panics on write error | [`internal/tui/output.go:152`](../internal/tui/output.go#L152) |
| Unused `Logger` interface | [`internal/contracts/logger.go:3`](../internal/contracts/logger.go#L3) |
| Prefix passed via context callback | [`internal/infra/prefix.go:9`](../internal/infra/prefix.go#L9), [`internal/server/server.go:193`](../internal/server/server.go#L193) |
| Second `Output` implementation | [`internal/uncors_app/output.go:11`](../internal/uncors_app/output.go#L11) |
