# A19 — Lua scripts are re-parsed on every request, with no timeout and no cancellation

**Severity:** Medium
**Area:** Script handler

---

## 1. What is wrong with the current approach

Every request that matches a `scripts:` route builds a fresh Lua VM and compiles
the script from source:

```go
// internal/handler/script/handler.go:36-55
func (h *Handler) executeScript(writer contracts.ResponseWriter, request *contracts.Request) error {
	luaState := newLuaState()          // new LState + PreloadModule x5 + OpenLibs
	defer luaState.Close()
	...
	err := h.runScript(luaState)
}

// internal/handler/script/handler.go:57-69
func (h *Handler) runScript(luaState *lua.LState) error {
	if h.script.Script != "" {
		return luaState.DoString(h.script.Script)      // ← lex + parse + compile, per request
	}
	scriptContent, err := afero.ReadFile(h.fs, h.script.File)   // ← disk read, per request
	if err != nil { return fmt.Errorf("%w: %s", ErrScriptFileNotFound, err.Error()) }
	return luaState.DoString(string(scriptContent))             // ← and compile, per request
}
```

Per request, for a feature whose whole purpose is generating responses quickly:

- a new `lua.LState` (`lua.NewState()` allocates a call stack, registry, and
  string interning tables and runs `OpenLibs`)
- five `PreloadModule` registrations ([`internal/handler/script/lua_state.go:15`](../internal/handler/script/lua_state.go#L15))
- a full **file read from disk** for file-backed scripts
- a full **lex/parse/compile** of the source
- construction of the request table, including `io.ReadAll` of the entire request
  body ([`internal/handler/script/lua_request.go:28`](../internal/handler/script/lua_request.go#L28))
- three metatables and ~8 closures for the response bindings
  ([`internal/handler/script/lua_response.go:19`](../internal/handler/script/lua_response.go#L19))

Beyond cost, three correctness/robustness gaps:

**(a) No context propagation.** `gopher-lua` supports `LState.SetContext(ctx)`,
which makes the VM abort on cancellation. It is never called. A script with an
infinite loop (`while true do end`) pins a goroutine and an OS thread **forever** —
the client disconnects, uncors keeps spinning, and only a process restart
recovers. There is also no execution deadline.

**(b) Full standard library, including `os` and `io`.** `lua.NewState()` runs
`OpenLibs`, which loads `base`, `io`, `os`, `debug` and more; `loadStandardLibraries`
then explicitly preloads `os` again. So `os.execute`, `os.remove` and `io.open`
are available to any script. For a tool whose config is authored by the user this
is defensible — a config file that can run shell commands is no worse than a
Makefile — but it should be a *documented, deliberate* decision, not a
side effect of using the default constructor. It matters more than it looks:
uncors configs get committed to repos and shared between teammates, and
`docs/Real-World-Examples.md` encourages copy-pasting them.

**(c) Script file changes are picked up per request, but script *errors* are
not surfaced usefully.** Because the file is read every time, editing it takes
effect immediately — an accidental feature that is arguably nice, but it means a
syntax error is discovered per request and reported through
`h.output.Errorf` ([`internal/handler/script/handler.go:28`](../internal/handler/script/handler.go#L28))
*and* returned, producing the stack-trace 500 page from
[A14](A14-http-error-page-leaks-stack-traces.md). Scripts referenced by file are
validated at config load ([`internal/config/script.go:96`](../internal/config/script.go#L96))
for *existence* only, never for syntax.

## 2. Why it is an architectural problem

- **Compilation is a construction-time concern being done at request time.** The
  script text is fixed for the lifetime of a config generation; the handler is
  already rebuilt on every reload
  ([A02](A02-di-container-leaks-resources-on-every-config-reload.md)), which is
  exactly the right moment to compile once.
- **There is no execution boundary.** A user-supplied program runs inside the
  request goroutine with no timeout, no cancellation and no resource limits. Any
  system that executes user code needs an explicit boundary; this one has none.
- **VM reuse is not modelled at all.** `gopher-lua`'s intended pattern for servers
  is to compile to a `*lua.FunctionProto` once and either create cheap states per
  request or pool them. Neither is done.

## 3. What the recommended approach is instead

**1. Compile once, at handler construction.**

```go
func NewHandler(opts ...HandlerOption) (*Handler, error) {
	h := helpers.ApplyOptions(&Handler{}, opts)
	src, name, err := h.source()                 // inline string or file read, once
	if err != nil { return nil, err }
	chunk, err := parse.Parse(strings.NewReader(src), name)
	if err != nil { return nil, fmt.Errorf("script %s: %w", name, err) }
	h.proto, err = lua.Compile(chunk, name)
	return h, err
}
```

A syntax error now fails **config load**, with the file name and line number,
instead of failing every request at runtime. That alone is a large UX improvement.

**2. Pool the VMs.**

```go
var statePool = sync.Pool{New: func() any { return newLuaState() }}

st := statePool.Get().(*lua.LState)
defer func() { resetGlobals(st); statePool.Put(st) }()
```

Reset only the globals the handler sets (`request`, `response`) between uses; do
**not** reuse a state that errored (close it instead) to avoid leaking a corrupted
VM. Even without pooling, hoisting compilation alone removes the dominant cost.

**3. Bind the context and add a deadline.**

```go
ctx, cancel := context.WithTimeout(request.Context(), h.timeout)  // default e.g. 5s
defer cancel()
st.SetContext(ctx)
```

Add a `timeout:` field to the script config (documented, with a sane default) so
long-running generators are possible but runaway loops are not.

**4. Make the sandbox explicit.** Replace `lua.NewState()` with
`lua.NewState(lua.Options{SkipOpenLibs: true})` and open exactly the libraries the
documentation promises (`base`, `string`, `table`, `math`, `os`, `json`). If `os`
and `io` are intended to be available, say so in `docs/Script-Handler.md` with a
security note; if they are not, do not open them. Today the answer is "whatever
`OpenLibs` happens to include", which is neither.

**5. Do not re-read script files per request** — but keep the developer-friendly
behaviour by recompiling on config reload (the file watcher already fires) or by
adding an explicit `watch: true` for script files.

## 4. Why the proposed approach is better

- **Per-request latency drops to VM setup plus execution**, removing a disk read
  and a full compile from the hot path. For a feature explicitly sold as "dynamic
  responses", this is the difference between a mock-speed and a
  noticeably-slow endpoint.
- **Syntax errors are reported once, at load, with a location** — instead of once
  per request, as a 500 page with a stack trace.
- **A runaway script can no longer wedge a goroutine permanently**, and client
  disconnects propagate.
- **The sandbox becomes a stated contract**, which is what a user needs before
  running a config file someone else wrote.
- Compilation-at-construction fits the generation-scoped lifecycle from
  [A02](A02-di-container-leaks-resources-on-every-config-reload.md) with no extra
  machinery.

## 5. Trade-offs and migration considerations

- **`NewHandler` gains an error return**, so `Container.ScriptHandler` and the
  router's script registration must propagate it
  ([`internal/di/public_api.go:139`](../internal/di/public_api.go#L139),
  [`internal/handler/router/router.go:77`](../internal/handler/router/router.go#L77)).
  `router.NewRouter` already returns an error, so the plumbing exists.
- **Losing per-request file re-reads is a behaviour change** for anyone editing
  script files live without touching the config. Mitigate by having the config
  watcher also watch referenced script files, or by documenting that a config
  touch reloads scripts.
- **State pooling must be conservative.** Lua globals set by a script persist in a
  pooled state; if `resetGlobals` misses something, one request can observe another
  request's leftovers. If that risk is unattractive, ship compilation-hoisting
  (large win, no risk) and skip pooling (smaller win, some risk).
- **`SetContext` changes error text** for aborted scripts; make sure the timeout
  case produces a clear message ("script exceeded 5s timeout") rather than a raw
  Lua error.
- **Restricting the standard library is potentially breaking** for existing user
  scripts that call `io.*`. If restricting, do it in a minor release with a note,
  or make it a config flag defaulting to the current permissive behaviour.

## 6. Code references

| What | Where |
| --- | --- |
| Per-request VM + compile | [`internal/handler/script/handler.go:36`](../internal/handler/script/handler.go#L36), [`:57`](../internal/handler/script/handler.go#L57) |
| Library loading | [`internal/handler/script/lua_state.go:8`](../internal/handler/script/lua_state.go#L8) |
| Request table (reads whole body) | [`internal/handler/script/lua_request.go:28`](../internal/handler/script/lua_request.go#L28) |
| Response bindings (metatables per request) | [`internal/handler/script/lua_response.go:19`](../internal/handler/script/lua_response.go#L19) |
| Existence-only validation | [`internal/config/script.go:96`](../internal/config/script.go#L96) |
| Handler construction | [`internal/di/public_api.go:139`](../internal/di/public_api.go#L139) |
| Route registration | [`internal/handler/router/router.go:77`](../internal/handler/router/router.go#L77) |
| Documented Lua surface | [`docs/Script-Handler.md`](../docs/Script-Handler.md) |
