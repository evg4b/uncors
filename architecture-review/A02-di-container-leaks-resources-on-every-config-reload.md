# A02 — The DI container leaks a HAR writer per reload and permanently ignores cache-config changes

**Severity:** Critical (data loss + unbounded resource growth during hot reload)
**Area:** Dependency injection / lifecycle
**Status:** Confirmed by code inspection; contradicts the documented behaviour

---

## 1. What is wrong with the current approach

`di.Container` owns a flat, append-only slice of closers:

```go
// internal/di/container.go:29
closers []io.Closer
```

Two factories append to it **every time they are called**:

```go
// internal/di/public_api.go:134-137  (HAR)
w := har.NewWriter(harConfig.File)
c.closers = append(c.closers, w)

// internal/di/factories.go:35-37  (cache)
instance := cache.NewRistrettoCache(cfs.MaxSize, cfs.ExpirationTime)
c.closers = append(c.closers, instance)
```

`Container.Close()` ([`internal/di/container.go:79`](../internal/di/container.go#L79))
is called exactly once, from `main`'s `defer`
([`main.go:25`](../main.go#L25)) — i.e. at process exit.

Now follow a config reload. Both run modes call `container.Targets(cfg)`
([`internal/di/public_api.go:172`](../internal/di/public_api.go#L172)) again:

- headless: [`internal/cli/run_non_ineractive.go:96`](../internal/cli/run_non_ineractive.go#L96)
- interactive: [`internal/uncors_app/app.go` `restart`](../internal/uncors_app/app.go)

`Targets` rebuilds the whole router graph, which calls `HARMiddleware` again for
every mapping with `har.file` set. Each call:

1. Spawns **a new background goroutine** (`har.NewWriter` → `go writer.run()`,
   [`internal/handler/har/writer.go:47`](../internal/handler/har/writer.go#L47)).
2. Appends a **new** `Writer` to `c.closers`.
3. Leaves the **previous** `Writer` running, still holding its accumulated
   `all []Entry` slice, still owning the same output path.

So after N reloads there are N live writers, N goroutines, N in-memory copies of
the archive — and, worse, **N writers racing to rewrite the same file**. `flush`
([`internal/handler/har/writer.go:109`](../internal/handler/har/writer.go#L109))
serialises *its own* snapshot of the entire archive and `os.Rename`s it over the
target. A stale writer that flushes after a fresh one has just written will
clobber the new file with its older entry set. HAR recordings are silently
corrupted by a config save.

### The cache case is a different bug with the same root

`Cache` is memoised through `factory1`, whose `GetOrBuild` is guarded by
`sync.Once` **and ignores the argument on every call after the first**:

```go
// internal/di/factory.go:31
func (f *factory1[T, D]) GetOrBuild(arg D) T {
	f.once.Do(func() { f.cache = f.factory(arg) })
	return f.cache
}
```

`D comparable` is declared as if it were a keyed cache, but there is no map —
the type parameter constraint is vestigial. Practical effect: editing
`cache-config.max-size`, `expiration-time`, or `methods` in the config file and
saving has **no effect at all** until the process is restarted, even though the
server visibly restarts and prints "Server restarted". Additionally, stale
entries cached under the old glob rules survive a reload that was meant to stop
caching those paths.

## 2. Why it is an architectural problem

The container conflates three different lifetimes into one:

| Lifetime | Examples | Correct owner |
| --- | --- | --- |
| Process | `fs`, `stdout`, `version`, `args` | container |
| Application | `Server`, `RequestTracker`, `HostCertManager` | container |
| **Configuration generation** | HAR writers, cache instance, routers, HTTP clients, URL replacers | **a per-generation scope** |

Everything in the third row is rebuilt on reload but registered against the
first row's lifetime. That is the classic "singleton container in a
hot-reloadable app" failure: the container has no notion of a scope, so nothing
can ever be torn down before the process ends.

`ARCHITECTURE.md` describes behaviour the code does not have:

> **Lifecycle management** — `Writer` implements `io.Closer`. The app registers
> each writer via `registerCloser`; on shutdown **or config reload** it calls
> `Close()` …

There is no `registerCloser` symbol anywhere in the tree, and no reload path
closes anything. (See [D05](D05-har-docs-overstate-coverage-and-lifecycle.md).)

## 3. What the recommended approach is instead

**Introduce an explicit per-generation scope.** Concretely:

```go
// A Runtime is everything derived from one UncorsConfig.
type Runtime struct {
	Targets []server.Target
	closers []io.Closer
}

func (c *Container) BuildRuntime(cfg *config.UncorsConfig) (*Runtime, error)
func (rt *Runtime) Close() error
```

`BuildRuntime` collects the closers it creates into the `Runtime`, not into the
container. The reload sequence becomes:

```go
next, err := container.BuildRuntime(newCfg)
if err != nil { report(err); return }        // keep serving the old config
if err := srv.Restart(ctx, next.Targets); err != nil { next.Close(); return }
prev.Close()                                  // flush + stop the old generation
prev = next
```

Note the ordering: build the new generation *before* tearing down the old one,
so a config that fails to build leaves the running proxy untouched.

**Fix the cache lifetime explicitly.** Either:
- make the cache part of the generation scope (rebuild + `Close()` the old
  ristretto instance on reload — correct, and makes `cache-config` reloadable), or
- keep it process-scoped but **document** that `cache-config` requires a restart
  and delete the misleading `factory1[T, D comparable]` argument so the API stops
  implying the argument matters.

The first option is preferable: users editing `expiration-time` and seeing
"Server restarted" reasonably expect it to have applied.

**Delete `factory1` or make it a real keyed cache.** As written it is a footgun:
any future `GetOrBuild(x)` call site inherits "the first argument silently wins".

## 4. Why the proposed approach is better

- **Resources are released deterministically** at the moment their configuration
  stops being current — which is exactly what the docs already promise.
- **HAR files stop being corrupted by config saves**, because only one writer per
  path is ever live.
- **Goroutine and memory growth become bounded** regardless of how many times the
  user saves their config (a file watcher fires on *every* editor save; this is
  not an exotic scenario).
- **Failed reloads become non-destructive.** Building the new generation first
  means a typo in the config no longer risks a half-torn-down server.
- **`cache-config` becomes genuinely hot-reloadable**, closing the gap between
  what the UI says happened and what happened.

## 5. Trade-offs and migration considerations

- **Cache warmth is lost on every reload** if the cache moves into the generation
  scope. For a dev proxy this is usually desirable (stale entries are the more
  common complaint), but if warmth matters, key the cache on a hash of
  `CacheConfig` and only rebuild when that hash changes.
- **Close ordering matters.** `har.Writer.Close()` drains and flushes
  synchronously ([`internal/handler/har/writer.go:60`](../internal/handler/har/writer.go#L60));
  closing the old generation *after* the new server is live means a brief window
  where two writers exist. That window is safe as long as the old one is closed
  before the new one flushes for the first time — closing immediately after
  `Restart` returns is sufficient in practice, but the write-to-tmp path should
  be made unique per writer (`path + ".tmp." + pid + counter`) to remove the
  theoretical race entirely.
- The change is contained: `internal/di` (new `Runtime` type), the two reload
  call sites, and `main.go`. Handler packages are untouched.
- Add a test that reloads the config 50 times and asserts `runtime.NumGoroutine()`
  is stable — this class of bug is invisible to functional tests.

## 6. Code references

| What | Where |
| --- | --- |
| Append-only closers slice | [`internal/di/container.go:29`](../internal/di/container.go#L29) |
| `Close()` only at process exit | [`internal/di/container.go:79`](../internal/di/container.go#L79), [`main.go:25`](../main.go#L25) |
| HAR writer appended per call | [`internal/di/public_api.go:134`](../internal/di/public_api.go#L134) |
| Cache appended per call | [`internal/di/factories.go:37`](../internal/di/factories.go#L37) |
| `factory1` ignores its argument | [`internal/di/factory.go:31`](../internal/di/factory.go#L31) |
| Router graph rebuilt on reload | [`internal/di/public_api.go:172`](../internal/di/public_api.go#L172) |
| Headless reload | [`internal/cli/run_non_ineractive.go:88`](../internal/cli/run_non_ineractive.go#L88) |
| Interactive reload | [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) (`restart`) |
| Full-file rewrite in `flush` | [`internal/handler/har/writer.go:109`](../internal/handler/har/writer.go#L109) |
| Docs claiming close-on-reload | [`ARCHITECTURE.md`](../ARCHITECTURE.md), [`docs/HAR-Collector.md`](../docs/HAR-Collector.md) |
