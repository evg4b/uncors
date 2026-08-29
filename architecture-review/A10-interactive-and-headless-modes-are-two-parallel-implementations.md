# A10 — Interactive and headless modes are two parallel implementations of the same application

**Severity:** High
**Area:** Application structure / UI separation

---

## 1. What is wrong with the current approach

The choice of UI decides which *application* runs:

```go
// internal/cli/run_uncors.go:31-36
if uncorsConfig.Interactive {
	runError = runIneractive(ctx, container, uncorsConfig, path)
} else {
	runError = runNonIneractive(ctx, container, uncorsConfig, path)
}
```

Both branches then independently implement the same startup sequence. Lining them
up:

| Step | Headless ([`run_non_ineractive.go`](../internal/cli/run_non_ineractive.go)) | Interactive ([`uncors_app/app.go`](../internal/uncors_app/app.go)) |
| --- | --- | --- |
| Print logo | `tui.PrintLogo(output, version)` (L26) | `tui.PrintLogo(m.output, ...)` (`startServerCmd`) |
| Print disclaimer | `output.WarnBox(tui.DisclaimerMessage)` (L28) | same, duplicated |
| Print mappings | `output.InfoBox(cfg.Mappings.String())` (L30) | same, duplicated |
| Build targets | `container.Targets(cfg)` (L33) | `m.container.Targets(m.cfg)` |
| Start server | `srv.Start(ctx, targets)` (L40) | `m.srv.Start(m.appContext(), targets)` |
| Version check | `go startVersionChecker(...)` + `time.Sleep(50ms)` (L45, L118) | `versionCheckCmd()` + `time.Sleep(50ms)` |
| Config watch | inline goroutine (L47) | `handleServerStarted` |
| Reload | `reloadServer` (L88) | `restart` |
| Signal handling | inline `signal.Notify` goroutine (L57) | **none** — relies on BubbleTea's ctrl+c |
| Shutdown | `srv.Shutdown(shutdownCtx)`, 15 s (L18) | `srv.Shutdown(ctx)`, **5 s** (`shutdownTimeout` in `app.go`) |
| Request display | **nothing** (see [A01](A01-request-tracker-deadlocks-headless-mode.md)) | `handleRequestEvent` |
| Watcher cleanup | never closed | `handleShutdown` |

Every row that differs is a defect in one of the two, not a deliberate
distinction:

- Headless never displays requests **and deadlocks** because of it.
- Interactive never handles `SIGTERM`/`SIGHUP`, so `docker stop`, `kill`, and
  systemd shutdown do not gracefully close a TUI-mode instance.
- Two different shutdown timeouts (5 s vs 15 s) with no rationale.
- The 50 ms sleep before the version check is copy-pasted into both, in both cases
  as a workaround for output ordering rather than as a real synchronisation.
- Headless leaks the fsnotify watcher; interactive closes it.
- Only headless handles a config-reload error correctly
  ([A03](A03-hot-reload-is-implemented-twice-and-unsafely.md)).

On top of this, the *output* abstraction is forked as well: `tui.CliOutput` writes
to an `io.Writer`, and `uncors_app.tuiOutput` implements the same
`contracts.Output` interface by **constructing a throwaway `tui.CliOutput` per
call, rendering into a `bytes.Buffer`, and pushing the resulting string onto a
channel** ([`internal/uncors_app/output.go:96-110`](../internal/uncors_app/output.go#L96)).
Every log line in TUI mode allocates a buffer, a `CliOutput`, a mutex, and a
string.

## 2. Why it is an architectural problem

- **The UI is not a presentation layer; it *is* the application** in interactive
  mode. `UncorsApp` owns the server, the container, the config, the watcher, the
  shutdown context and the version checker — a BubbleTea `Model` is supposed to
  own view state and nothing else. Its `Update` method is 60 lines of application
  control flow interleaved with widget plumbing
  ([`internal/uncors_app/app.go` `Update`](../internal/uncors_app/app.go)).
- **There is no seam where "the app" ends and "the display" begins**, so nothing
  can be tested without picking a UI, and any behaviour added to one mode has to
  be manually mirrored into the other. The evidence that this does not happen is
  in the table above.
- **A third mode is currently impossible.** JSON log output for CI, a `--quiet`
  mode, or a future HTTP admin endpoint would each require a third copy of the
  startup sequence.

## 3. What the recommended approach is instead

**Extract the application; make both UIs consumers of it.** This is the same
`Runtime` proposed in [A03](A03-hot-reload-is-implemented-twice-and-unsafely.md):

```go
// internal/runtime
type Runtime struct{ /* container, server, generation, watcher */ }

func New(c *app.Container, cfg *config.UncorsConfig, cfgPath string) *Runtime
func (r *Runtime) Start(ctx context.Context) error
func (r *Runtime) Reload() error
func (r *Runtime) Shutdown(ctx context.Context) error
func (r *Runtime) Events() <-chan Event   // Started, Reloaded, ReloadFailed, Request, Log, Stopped
```

- **Signal handling, shutdown timeout, watcher ownership, version checking and
  reload policy live in `Runtime`, once.**
- **`main` selects a presenter, not an application:**

```go
rt := runtime.New(container, cfg, path)
var present func(context.Context, *runtime.Runtime) error
if cfg.Interactive && term.IsTerminal(os.Stdout.Fd()) {
	present = tuipresenter.Run
} else {
	present = plainpresenter.Run   // renders Events() to CliOutput
}
return present(ctx, rt)
```

- **`uncorsapp.UncorsApp` shrinks to a real Model**: it holds widgets, consumes
  `rt.Events()`, and calls `rt.Reload()` / `rt.Shutdown()` on key presses. No
  container, no server, no watcher.
- **`tuiOutput` disappears.** The `Runtime` emits structured events
  (`RequestEvent`, `LogEvent{level, text}`); the plain presenter formats them with
  `tui.CliOutput`, the TUI presenter formats them with lipgloss directly. Neither
  needs to render into a buffer to hand a string to the other.
- **Add the TTY check shown above.** `--interactive` defaulting to `true` means
  `uncors | tee log.txt`, `npx uncors` in a CI script, and the Docker image all
  start a full-screen alt-screen TUI against a non-terminal
  ([D09](D09-docker-instructions-cannot-work.md)). Falling back to the plain
  presenter when stdout is not a TTY is standard CLI behaviour and removes a whole
  category of user confusion.

## 4. Why the proposed approach is better

- **Behaviour stops depending on the UI.** Signals, shutdown timeouts, reload
  semantics and request display become properties of uncors, not of whichever
  branch you took.
- **The bugs in the table close by construction** — there is no second
  implementation left to forget something.
- **Both modes become testable** without a terminal or a socket by asserting on the
  event stream.
- **Per-line allocations in TUI mode drop** once events carry data instead of
  pre-rendered strings.
- **A `--output=json` or `--quiet` presenter becomes a ~50-line addition**, which
  is a real ask for a tool used in scripts.

## 5. Trade-offs and migration considerations

- **This is the largest refactor proposed in this review**, and it subsumes
  [A03](A03-hot-reload-is-implemented-twice-and-unsafely.md). Do it in stages:
  1. Move signal handling + shutdown timeout into a shared helper used by both
     branches (fixes SIGTERM in TUI mode immediately).
  2. Introduce `Runtime` with `Start`/`Shutdown`/`Reload`/`Events`; port the
     headless branch to it first (it is the simpler consumer).
  3. Port the TUI branch, deleting `tuiOutput` in the process.
  4. Add the TTY fallback.
- **Event granularity needs care.** Too coarse and the TUI can't render progress;
  too fine and the channel becomes the bottleneck from
  [A01](A01-request-tracker-deadlocks-headless-mode.md). Keep it to
  request-lifecycle + log-line + lifecycle-state, all non-blocking sends.
- **The 50 ms version-check sleep should be deleted, not moved.** Emit the
  "new version available" line as an ordinary event; ordering then falls out of the
  event stream rather than out of a race.
- Existing tests in [`internal/cli/run_uncors_test.go`](../internal/cli/run_uncors_test.go)
  and `internal/uncors_app/*_test.go` will need reworking, but most of them are
  currently testing plumbing that disappears.

## 6. Code references

| What | Where |
| --- | --- |
| Mode fork | [`internal/cli/run_uncors.go:31`](../internal/cli/run_uncors.go#L31) |
| Headless startup | [`internal/cli/run_non_ineractive.go:20`](../internal/cli/run_non_ineractive.go#L20) |
| Headless signal handling (absent in TUI) | [`internal/cli/run_non_ineractive.go:57`](../internal/cli/run_non_ineractive.go#L57) |
| Headless shutdown timeout 15 s | [`internal/cli/run_non_ineractive.go:19`](../internal/cli/run_non_ineractive.go#L19) |
| TUI owns the whole app | [`internal/uncors_app/app.go:31`](../internal/uncors_app/app.go#L31) |
| TUI shutdown timeout 5 s | [`internal/uncors_app/app.go:23`](../internal/uncors_app/app.go#L23) |
| Duplicated version-check sleep | [`internal/cli/run_non_ineractive.go:118`](../internal/cli/run_non_ineractive.go#L118), [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) (`versionCheckCmd`) |
| Output re-rendered per line | [`internal/uncors_app/output.go:96`](../internal/uncors_app/output.go#L96) |
| Interactive default | [`internal/config/flags.go:20`](../internal/config/flags.go#L20) |
