# A12 — Package boundaries: config depends on the server and the TUI, and `helpers` is a junk drawer

**Severity:** Medium
**Area:** Package structure / separation of responsibilities

---

## 1. What is wrong with the current approach

### (a) `internal/config` imports `internal/server`

```go
// internal/config/mapping.go:11, 82-92
import "github.com/evg4b/uncors/internal/server"

func ValidateTLS(_ string, mapping Mapping, fs afero.Fs) error {
	if mapping.From.Scheme != httpsScheme { return nil }
	if !server.CAExists(fs) { return &TLSError{mapping.From.HostPort()} }
	return nil
}
```

Configuration — the lowest layer, which everything else consumes — reaches up
into the server layer to ask whether a CA file exists on disk. `server.CAExists`
itself calls `os.UserHomeDir()` ([`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20)),
so config validation is also coupled to the process environment in a way that is
invisible from the config package's API. (This is why
[`internal/cli/generate_certs_test.go`](../internal/cli/generate_certs_test.go) has
to `t.Setenv("HOME", …)`.)

### (b) `internal/config` imports `internal/tui`

```go
// internal/config/flags.go:6, 12
import "github.com/evg4b/uncors/internal/tui"
flags.Usage = func() { tui.PrintLogo(flags.Output(), version) ... }
```

Config now depends on the presentation layer to render an ASCII logo. A package
that parses and validates YAML should not transitively pull in lipgloss.

### (c) `internal/helpers` is an unstructured catch-all

Eleven unrelated files: HTTP status predicates, request normalisation, a
generic options applier, map cloning, a log sanitiser, a panic interceptor with
build-tag variants, a signal-handling helper, a closer that panics, and an
`unsafe.Pointer` nil check. It is imported by nearly every package, which makes
it a de-facto second `contracts` — a dependency hub with no cohesion.

Two of its members are actively hazardous:

```go
// internal/helpers/asset.go:8  — reads the interface data word directly
func AssertIsDefined(value any, message ...string) {
	if (*[2]uintptr)(unsafe.Pointer(&value))[1] == 0 { panic(message) }
}

// internal/helpers/closer.go:6  — turns any Close error into a panic
func CloseSafe(resource io.Closer) {
	if err := resource.Close(); err != nil { panic(err) }
}
```

`AssertIsDefined` depends on the internal layout of Go's `iface`/`eface` and
exists only to compensate for the runtime-wired DI of
[A04](A04-service-locator-di-container.md). `CloseSafe` is used in the proxy hot
path ([`internal/handler/proxy/handler.go:69`](../internal/handler/proxy/handler.go#L69)),
where an upstream body close error becomes a panic — the name says the opposite
of what it does.

### (d) `internal/contracts` holds an interface nobody implements or uses

`contracts.Logger` ([`internal/contracts/logger.go:3`](../internal/contracts/logger.go#L3))
has 10 methods and zero non-test references; the only thing that mentions it is a
generated minimock file. Meanwhile the interface that *is* used, `contracts.Output`,
bundles seven responsibilities (`io.Writer`, info/warn/error, `Print`,
`Request(*RequestData)`, and a `NewPrefixOutput` factory) into one type that every
handler must accept in full.

### (e) `pkg/` vs `internal/` is inconsistent

`pkg/urlt` is a public package (a fork of `net/url` —
[A17](A17-pkg-urlt-is-a-fork-of-net-url.md)) that exists purely to serve
`internal/urlreplacer`. Publishing it as `pkg/` commits the project to its API
surface for external consumers who have no reason to want it.

### (f) Two packages named for the same role

`internal/cli` and `internal/commands` ([A11](A11-cli-command-structure.md)), and
`internal/tui` (rendering primitives) vs `internal/uncors_app` (BubbleTea app,
package name `uncorsapp`, directory name `uncors_app` — a snake_case directory in
a Go tree).

## 2. Why it is an architectural problem

The intended layering is clear from the package names — `contracts` → `config` →
`handler/*` → `server` → `cli`/`tui` — but three edges run the wrong way
(config→server, config→tui, handler/router→di). Once the lowest layer depends on
the highest:

- **Compile-time coupling explodes.** Touching `tui` rebuilds `config`, which
  rebuilds everything.
- **Testing the low layer requires the high layer's environment** (a HOME
  directory, a terminal-capable renderer).
- **The layering stops being a design tool.** New code has no signal about where
  something belongs, which is how `Targets` ended up in the DI container and
  `NormaliseRequest` in `helpers`.

A `helpers` package with no theme is the standard symptom: it is where things go
when the layering does not say where they belong.

## 3. What the recommended approach is instead

**Invert the two bad edges out of `config`:**

1. **CA presence becomes an injected predicate, not an import.**

```go
// internal/config
type Environment struct {
	Fs      afero.Fs
	CAExists func() bool     // supplied by the caller
}
func (cfg *UncorsConfig) Validate(env Environment) error
```

   Or, cleaner still: move TLS-readiness out of config validation entirely and
   check it in the server when a TLS listener is actually created — that is where
   the requirement lives, and it removes an environment dependency from a pure
   validation function.

2. **The usage banner moves to the CLI layer.** `config` exposes
   `RegisterFlags(*pflag.FlagSet)`; whoever owns the command tree
   ([A11](A11-cli-command-structure.md)) sets `flags.Usage` and prints the logo.

**Dissolve `helpers` into cohesive homes:**

| Current | Suggested home |
| --- | --- |
| `Is1xxCode`…`NormaliseStatusCode`, `NormaliseRequest`, `ToRequestData` | `internal/infra` (HTTP utilities) |
| `ApplyOptions`, `CloneMap` | keep as a small `internal/xslices`-style package, or inline |
| `SanitizeLogValue` | `internal/infra` (logging) |
| `PanicInterceptor` | `internal/runtime` (only the runtime should be recovering) |
| `GracefulShutdown` | delete — dead ([A20](A20-dead-code-and-vestigial-abstractions.md)) |
| `CloseSafe` | delete; use `defer x.Close()` or log the error |
| `AssertIsDefined` | delete; constructors take required deps as parameters |

**Split `contracts.Output`** into the roles that are actually consumed:

```go
type Printer interface { Print(string); Printf(string, ...any) }
type Reporter interface { Info(any); Warn(any); Error(any) }   // + Boxes
type RequestReporter interface { Request(*RequestData) }
```

Handlers take `Reporter`; the request pipeline takes `RequestReporter`. Delete
`contracts.Logger` and its generated mock.

**Move `pkg/urlt` to `internal/urlt`** unless there is a deliberate decision to
support external users of it.

**Rename `internal/uncors_app` → `internal/tui/app`** (package `app`), so the
directory name matches Go convention and the presentation layer is one subtree.

## 4. Why the proposed approach is better

- **The dependency graph becomes acyclic and directional**, which is the point of
  having layers at all: you can reason about, test, and replace a layer without
  the ones above it.
- **`config` becomes pure**: YAML in, validated struct or errors out, no
  filesystem-of-record, no terminal. That makes it trivially testable and makes
  a future `uncors validate` command ([A11](A11-cli-command-structure.md)) a
  one-liner.
- **Deleting `unsafe` from the tree** removes a genuine portability and maintenance hazard
  for a benefit (a nicer panic message) that disappears once dependencies are
  passed as constructor parameters.
- **Narrow interfaces mean narrow mocks.** `testing/mocks/output_mock.go` is
  generated for a seven-method interface used by handlers that need one method.
- Smaller, focused packages reduce rebuild time and make `go doc` useful.

## 5. Trade-offs and migration considerations

- **Removing `ValidateTLS` from config changes when the user sees the error.**
  Today a missing CA is reported at startup with a nice `TLSError`; if the check
  moves to listener creation it is reported slightly later. Preserve the UX by
  having the server perform the check *before* binding and surface the same
  message — do not simply drop the check.
- **Dissolving `helpers` touches many import blocks.** It is mechanical and can be
  done file-by-file with `gopls` rename; do it in one pass to avoid a long period
  with both locations live.
- **Splitting `contracts.Output` is a wide but shallow change**; it can be staged
  by defining the narrow interfaces first, having `Output` embed them, and then
  narrowing consumers one at a time.
- **Moving `pkg/urlt` to `internal/` is a breaking change for any external
  importer.** Check the module's usage before deciding; given it is a fork of the
  standard library with an uncors-specific API, external usage is unlikely.
- Note `.golangci.yml` excludes `pkg/urlt` from linting entirely
  ([`.golangci.yml`](../.golangci.yml)) — a 1 100-line unlinted package in the tree
  is itself worth revisiting.

## 6. Code references

| What | Where |
| --- | --- |
| config → server | [`internal/config/mapping.go:11`](../internal/config/mapping.go#L11), [`:83`](../internal/config/mapping.go#L83) |
| config → tui | [`internal/config/flags.go:6`](../internal/config/flags.go#L6), [`:12`](../internal/config/flags.go#L12) |
| `CAExists` reads `$HOME` | [`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20) |
| `unsafe` nil check | [`internal/helpers/asset.go:8`](../internal/helpers/asset.go#L8) |
| Panicking closer, used in proxy | [`internal/helpers/closer.go:6`](../internal/helpers/closer.go#L6), [`internal/handler/proxy/handler.go:69`](../internal/handler/proxy/handler.go#L69) |
| Unused `Logger` interface | [`internal/contracts/logger.go:3`](../internal/contracts/logger.go#L3) |
| Wide `Output` interface | [`internal/contracts/output.go:25`](../internal/contracts/output.go#L25) |
| Router → DI (see A04) | [`internal/handler/router/router.go:23`](../internal/handler/router/router.go#L23) |
| Public fork package | [`pkg/urlt/url.go`](../pkg/urlt/url.go) |
| Lint exclusion for `pkg/urlt` | [`.golangci.yml`](../.golangci.yml) |
