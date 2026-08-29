# A03 — Hot reload is implemented twice, discards errors, and can restart the server with a `nil` config

**Severity:** High
**Area:** Configuration hot reload / application lifecycle

---

## 1. What is wrong with the current approach

There are two independent implementations of "watch the config file and restart
the server":

**Headless** — [`internal/cli/run_non_ineractive.go:55-62`](../internal/cli/run_non_ineractive.go#L56)
plus `reloadServer` at [`:88`](../internal/cli/run_non_ineractive.go#L88):

```go
go func() {
	watcher := config.NewWatcher(cfgPath)
	err := watcher.Watch(ctx, func() { reloadServer(ctx, container, srv) })
	if err != nil { output.Error(err) }
}()
```

**Interactive** — [`internal/uncors_app/app.go` `handleServerStarted`](../internal/uncors_app/app.go):

```go
watcher := config.NewWatcher(m.configPath)
err := watcher.Watch(m.appContext(), func() {
	defer helpers.PanicInterceptor(func(value any) {
		m.output.Errorf("Config reloading error: %v", value)
	})
	newCfg := m.loadConfig()
	err := m.restart(m.appContext(), newCfg)
	...
})
```

Both do the same four things (reload config → build targets → `srv.Restart` →
print a summary), with different error handling, different logging, different
lifetimes, and only one of them closing the watcher.

Specific defects that follow from the duplication:

**(a) The interactive path throws away config errors.** `loadConfig` is a closure
built in [`internal/cli/run_ineractive.go:23`](../internal/cli/run_ineractive.go#L23):

```go
func() *config.UncorsConfig {
	reloaded, _, _ := config.LoadConfiguration(container.Fs(), container.Version(), container.Args())
	return reloaded
}
```

Both the path and the **error** are discarded. When the user saves a config with
a typo, `LoadConfiguration` returns `(nil, "", err)` and this closure returns
`nil`. `restart(ctx, nil)` then dereferences it (`m.container.Targets(cfg)` →
`cfg.Mappings.GroupByPort()`) and panics. The panic is caught by
`helpers.PanicInterceptor` and rendered as `Config reloading error: runtime error:
invalid memory address or nil pointer dereference` — a stack-trace message where
the user should have seen `mappings[0].from is not a valid host`. In non-release
builds `PanicInterceptor` re-panics
([`internal/helpers/panic_debug.go`](../internal/helpers/panic_debug.go)), taking
the TUI down.

The headless path handles the same case correctly
([`internal/cli/run_non_ineractive.go:91`](../internal/cli/run_non_ineractive.go#L91)).
Two implementations, one of them wrong — the textbook cost of duplication.

**(b) The headless watcher is never closed.** The interactive path stores the
watcher and closes it on shutdown
([`internal/uncors_app/app.go` `handleShutdown`](../internal/uncors_app/app.go)).
The headless path creates it inside an anonymous goroutine and drops the
reference. `Watcher.Close()` is unreachable; the `fsnotify` inotify/kqueue handle
is only released at process exit.

**(c) Reload is destructive on failure.** `reloadServer` calls
`srv.Restart(ctx, targets)` which does `Shutdown` **then** `Start`
([`internal/server/server.go:147`](../internal/server/server.go#L147)). If the
new `Start` fails — e.g. the edited config moved a mapping onto a port something
else has since taken — the old listeners are already gone. The user is left with
a running process serving nothing, and the error is only printed.

**(d) Every save is a full teardown.** The watcher debounces 10 ms
([`internal/config/watcher.go:16`](../internal/config/watcher.go#L16)) and then
restarts *all* listeners for *all* port groups, even when the change touched a
single mock body on a single mapping. Every in-flight connection on every port is
dropped, all TLS sessions are renegotiated, the whole router graph is rebuilt,
and (per [A02](A02-di-container-leaks-resources-on-every-config-reload.md)) a new
HAR writer is leaked.

**(e) `Watch` returns `nil` when there is no config file** ([`internal/config/watcher.go:41`](../internal/config/watcher.go#L41))
but still leaves `isWatching == false`, so the "already watching" guard at
[`:37`](../internal/config/watcher.go#L37) is inconsistent with the `filePath == ""`
early return. Minor, but symptomatic of the type being under-specified.

## 2. Why it is an architectural problem

Hot reload is a *cross-cutting application concern*, not a UI concern. Placing it
inside the BubbleTea model on one side and inside a CLI helper function on the
other means:

- The reload contract (what happens on a bad config, what gets torn down, in what
  order) is defined twice and can drift — and has already drifted.
- Neither implementation can be unit-tested in isolation; testing reload today
  requires booting a real TUI or a real listener (see
  [`internal/cli/run_uncors_test.go`](../internal/cli/run_uncors_test.go), which
  spins a real port and `time.Sleep`s for the debounce).
- The "reload succeeded / failed" state is not represented anywhere; it exists
  only as a printed line. A TUI that wanted to show a persistent "config invalid
  since 14:32" banner has nothing to bind to.

## 3. What the recommended approach is instead

Extract a single, UI-agnostic reload engine that owns the whole generation
lifecycle:

```go
// internal/runtime (new package)
type Runtime struct { /* container, server, current generation */ }

type Event interface{} // ReloadStarted | ReloadFailed{err} | ReloadApplied{cfg} | Stopped

func (r *Runtime) Run(ctx context.Context) error   // starts server + watcher
func (r *Runtime) Events() <-chan Event            // for whoever wants to render
func (r *Runtime) Reload() error                   // also used by the TUI's 'r' key
```

Rules the engine enforces in one place:

1. **Load and validate first.** If `LoadConfiguration` errors, emit
   `ReloadFailed{err}` and keep serving the previous generation. Never call
   `Restart` with an unvalidated config, and never discard the error.
2. **Build before tearing down.** Construct the new targets (and their HAR
   writers, caches, HTTP clients) *before* shutting anything down; on build
   failure, close the half-built generation and keep the old one.
3. **Restart only what changed.** Diff the new `PortGroups`
   ([`internal/config/mappings.go:57`](../internal/config/mappings.go#L57)) against the
   current ones: ports whose group is byte-identical keep their listener and just
   swap the `http.Handler` (an `atomic.Pointer[http.Handler]` behind
   `Server.handleRequest` makes this trivial); only added/removed/changed ports
   are opened or closed.
4. **Own the watcher.** One `Watcher`, created and `Close()`d by the engine.

Both front-ends then become thin:

```go
// headless
rt := runtime.New(container, cfg, cfgPath)
go printEvents(rt.Events(), output)
return rt.Run(ctx)

// interactive
rt := runtime.New(container, cfg, cfgPath)
tea.NewProgram(uncorsapp.New(rt)).Run()  // model renders rt.Events()
```

## 4. Why the proposed approach is better

- **One reload contract.** The `nil`-config panic simply cannot be written twice
  if the code exists once, and the correct headless handling becomes the
  behaviour of both modes.
- **Failed reloads stop being destructive.** Editing a config with a typo becomes
  a no-op with a red line, which is what a developer expects from a file watcher.
- **Reload becomes testable without sockets or a terminal.** `Runtime` can be
  driven with a fake filesystem and asserted on its event stream.
- **Restart cost drops from "all listeners" to "changed listeners"**, which
  matters because editors fire the watcher on every keystroke-triggered autosave.
- It composes with [A02](A02-di-container-leaks-resources-on-every-config-reload.md):
  the generation scope and the reload engine are the same boundary.

## 5. Trade-offs and migration considerations

- **Swapping handlers on a live listener changes shutdown semantics.** In-flight
  requests finish against the old handler; that is normally desirable but means a
  reload is no longer a hard cut. If a hard cut is wanted for a given port, close
  it explicitly.
- **Diffing port groups needs a stable comparison.** `config.Mapping` already has
  `Clone()` everywhere; add a cheap `Hash()`/`Equal()` or compare the marshalled
  YAML of each group. Getting the diff wrong is worse than not diffing, so ship
  the "build-before-teardown" fix first and the diffing optimisation second.
- **A new `internal/runtime` package must not import `internal/tui` or
  `internal/uncors_app`** or the layering problem in
  [A12](A12-package-boundaries-and-layering-violations.md) reappears. Communication
  is via the event channel only.
- `helpers.PanicInterceptor` should stop being the reload error handler once real
  errors are propagated; keep it only as a last-resort guard.

## 6. Code references

| What | Where |
| --- | --- |
| Headless watcher + reload | [`internal/cli/run_non_ineractive.go:56`](../internal/cli/run_non_ineractive.go#L56), [`:88`](../internal/cli/run_non_ineractive.go#L88) |
| Interactive watcher + reload | [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) (`handleServerStarted`, `restart`, `restartCmd`) |
| Error-discarding config loader | [`internal/cli/run_ineractive.go:23`](../internal/cli/run_ineractive.go#L23) |
| `Restart` = Shutdown-then-Start | [`internal/server/server.go:147`](../internal/server/server.go#L147) |
| Watcher (never closed headless) | [`internal/config/watcher.go:34`](../internal/config/watcher.go#L34), [`:76`](../internal/config/watcher.go#L76) |
| Debounce constant | [`internal/config/watcher.go:16`](../internal/config/watcher.go#L16) |
| Port grouping used for restart | [`internal/config/mappings.go:57`](../internal/config/mappings.go#L57) |
| Panic-as-error-handling | [`internal/helpers/panic_debug.go`](../internal/helpers/panic_debug.go), [`internal/helpers/panic.go`](../internal/helpers/panic.go) |
