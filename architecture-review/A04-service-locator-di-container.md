# A04 — The DI container is a service locator that the router reaches back into

**Severity:** High (design)
**Area:** Dependency injection / package structure

---

## 1. What is wrong with the current approach

`internal/di.Container` is a hand-rolled container that has grown into the
application's central hub. It:

- holds process state (`fs`, `stdout`, `args`, `version`) —
  [`internal/di/container.go:17-21`](../internal/di/container.go#L17)
- memoises singletons through a bespoke `factory[T]`/`factory1[T,D]` type —
  [`internal/di/factory.go`](../internal/di/factory.go)
- exposes 20+ construction methods, several of which are not "get a dependency"
  at all but application logic: `Router(...)` and `Targets(cfg)` build the entire
  request-handling graph and translate config into listener descriptors —
  [`internal/di/public_api.go:159`](../internal/di/public_api.go#L159), [`:172`](../internal/di/public_api.go#L172)
- accumulates closers as a side effect of construction —
  [`internal/di/public_api.go:136`](../internal/di/public_api.go#L136)
- supports mutation after construction via `Override` —
  [`internal/di/override.go:5`](../internal/di/override.go#L5)

And critically, the router **depends on the container**:

```go
// internal/handler/router/router.go:23-30
type DI interface {
	StaticMiddleware(path string, dir config.StaticDirectory) contracts.Middleware
	RewriteMiddleware(rewriting *config.RewritingOption) contracts.Middleware
	HARMiddleware(harConfig *config.HARConfig) contracts.Middleware
	ScriptHandler(scriptConfig *config.Script) contracts.Handler
	OptionsMiddleware(cfg config.OptionsHandling) contracts.Middleware
	MockHandler(response *config.Response) contracts.Handler
}

type Router struct {
	*mux.Router
	defaultHandler contracts.Handler
	container      DI                       // ← the router pulls from the container
	cacheMiddlewareFactory CacheMiddlewareFactory
}
```

`router.WithDiContainer(c)` ([`internal/handler/router/router_options.go:9`](../internal/handler/router/router_options.go#L9))
hands the container to a leaf package so that the leaf can call back and
construct whatever it decides it needs. This is the service-locator
anti-pattern: the dependency graph is not visible at any single point, it is
discovered at runtime by whichever `registerMapping` branch happens to execute.

Two further smells sit on top of it:

- **Mixed granularity.** Some things are cached singletons (`Server`,
  `RequestTracker`, `HostCertManager`), others are rebuilt on every call
  (`ProxyHandler`, `MockHandler`, `RewriteMiddleware`, `HARMiddleware`), and one
  (`Cache`) is a singleton that *pretends* to be parameterised
  ([A02](A02-di-container-leaks-resources-on-every-config-reload.md)). Nothing in the API
  distinguishes these.
- **The `Override` escape hatch exists solely for the TUI**
  ([`internal/uncors_app/app.go`](../internal/uncors_app/app.go) swaps
  `CliOutput` after the container is built, and
  [`main.go:30`](../main.go#L30)/[`:36`](../main.go#L36) mutate `args` after
  construction). Post-construction mutation of a container that already handed out
  memoised singletons is a race waiting to happen — `CliOutput` may already have
  been built and captured by another component before the override lands.

## 2. Why it is an architectural problem

- **The dependency graph is unreadable.** To answer "what does the mock handler
  need?" you must read `di.MockHandler`, which reads `c.fs`, which was set by an
  option in `main`, and pass through `router.registerMapping`. A plain
  constructor would answer it in one line.
- **Leaf packages depend on the composition root.** `internal/handler/router`
  importing a `DI` interface (structurally satisfied only by `*di.Container`)
  inverts the intended layering: the composition root should depend on the
  handlers, never the reverse. It also means the router package cannot be reused,
  tested, or reasoned about without a container — its own tests build one
  ([`internal/handler/router/router_test.go:88`](../internal/handler/router/router_test.go#L88)).
- **Construction has side effects.** `HARMiddleware` starts a goroutine and
  registers a closer; `Cache` allocates a ristretto instance. Calling a "getter"
  mutates global-ish state, which is why the reload leak in
  [A02](A02-di-container-leaks-resources-on-every-config-reload.md) is so easy to
  introduce and so hard to see.
- **`Targets` is business logic in the container.** Port grouping, address
  formatting, TLS selection and error aggregation
  ([`internal/di/public_api.go:172-192`](../internal/di/public_api.go#L172)) belong to
  the server/runtime layer, not to a DI helper. Today there is no other place
  where "a config becomes a set of listeners" is expressed.

Go does not need a container for a program of this size. 9 200 lines of
non-test code with ~10 injectable seams is squarely in "wire it by hand"
territory.

## 3. What the recommended approach is instead

**Replace the locator with an explicit builder + plain constructors.**

1. **Make the router take what it needs, not a container.** Everything the
   `DI` interface exposes is a *factory keyed by a config fragment*. Bundle them
   into one value object the composition root fills in:

```go
// internal/handler/router
type Deps struct {
	Static  func(path string, dir config.StaticDirectory) contracts.Middleware
	Rewrite func(*config.RewritingOption) contracts.Middleware
	HAR     func(*config.HARConfig) contracts.Middleware
	Options func(config.OptionsHandling) contracts.Middleware
	Cache   func(config.CacheGlobs) contracts.Middleware
	Mock    func(*config.Response) contracts.Handler
	Script  func(*config.Script) contracts.Handler
	Proxy   contracts.Handler
}

func New(mappings config.Mappings, deps Deps) (*Router, error)
```

The router now states its requirements at the type level, has no import of
`internal/di`, and is unit-testable with struct literals of closures.

2. **Move `Targets` out of `di` into the runtime/server layer** as
   `server.TargetsFor(cfg, buildHandler)` or as part of the `Runtime` from
   [A03](A03-hot-reload-is-implemented-twice-and-unsafely.md).

3. **Keep a thin composition root.** `internal/app` (or the existing
   `internal/di`, renamed) holds only:
   - the process-scoped values (`fs`, `stdout`, `version`, `args`)
   - the app-scoped singletons (`Server`, `HostCertManager`, `RequestSink`)
   - a `BuildRuntime(cfg) (*Runtime, error)` that wires the per-generation graph
     and returns it with its closers attached.

4. **Delete `Override`.** Choose the output implementation *before* constructing
   the container: `main` decides interactive vs headless from the parsed config,
   then builds the container with the right `contracts.Output`. That removes the
   post-construction mutation and the ordering hazard.

5. **Delete `factory[T]`/`factory1[T,D]`.** Two `sync.Once`-guarded generics for
   five singletons is more machinery than `sync.OnceValue` from the standard
   library, which does the same job with no custom type.

## 4. Why the proposed approach is better

- **Dependencies become readable and checkable by the compiler.** A missing
  dependency is a build error at the composition root, not a `nil` interface
  discovered at request time (the current code guards against exactly this with
  `helpers.AssertIsDefined` panics scattered through constructors — e.g.
  [`internal/handler/proxy/handler.go:25-27`](../internal/handler/proxy/handler.go#L26) —
  which would become unnecessary).
- **Layering is restored:** `handler/*` → `contracts`/`config`, composition root
  → everything. No cycles, no back-references.
- **Construction becomes side-effect free**, which is the precondition for the
  generation-scoped teardown in [A02](A02-di-container-leaks-resources-on-every-config-reload.md).
- **Tests get simpler.** `router_test.go` currently needs `di.NewContainer` plus a
  `testutils.Close` deferral just to build a router; with `Deps` it needs a
  struct literal.
- **`sync.OnceValue` removes ~45 lines** of bespoke generic machinery, including
  the `factory1` argument-ignoring bug.

## 5. Trade-offs and migration considerations

- **This is a wide refactor** touching `internal/di`, `internal/handler/router`,
  and every test that builds a container. It can be staged:
  1. Replace `router.DI` with `router.Deps` (mechanical; the container just fills
     the struct). This alone breaks the leaf→root dependency.
  2. Move `Targets` to the runtime layer.
  3. Replace `factory`/`factory1` with `sync.OnceValue`.
  4. Remove `Override` once mode selection moves ahead of container construction.
- **`Deps` as a struct of closures is slightly less discoverable than an
  interface** for someone reading the router in isolation; a short doc comment on
  each field covers this. The alternative — passing eight constructor arguments —
  is worse.
- **Do not replace this with a reflection-based DI library.** The problem is not
  that the container is hand-written; it is that it is a *locator*. A generated or
  reflective container would preserve the same inversion.
- Keep `WithFs`/`WithStdout`/`WithArgs`-style options: they are legitimate and
  make the test seams obvious.

## 6. Code references

| What | Where |
| --- | --- |
| Container definition | [`internal/di/container.go:16`](../internal/di/container.go#L16) |
| Bespoke factory generics | [`internal/di/factory.go:5`](../internal/di/factory.go#L5), [`:24`](../internal/di/factory.go#L24) |
| Container public surface | [`internal/di/public_api.go`](../internal/di/public_api.go) |
| `Targets` (business logic in DI) | [`internal/di/public_api.go:172`](../internal/di/public_api.go#L172) |
| Router depends on container | [`internal/handler/router/router.go:23`](../internal/handler/router/router.go#L23), [`:37`](../internal/handler/router/router.go#L37) |
| `WithDiContainer` option | [`internal/handler/router/router_options.go:9`](../internal/handler/router/router_options.go#L9) |
| Post-construction mutation | [`internal/di/override.go:5`](../internal/di/override.go#L5), [`main.go:30`](../main.go#L30), [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) |
| Runtime nil-checks compensating for the locator | [`internal/helpers/asset.go:8`](../internal/helpers/asset.go#L8) and its call sites |
