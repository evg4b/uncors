# A06 — Cross-cutting middleware wraps only the proxy branch, so mocks, scripts and statics silently bypass HAR, cache and OPTIONS

**Severity:** High
**Area:** Routing / middleware composition
**Status:** Behaviour verified with a targeted test

---

## 1. What is wrong with the current approach

`Router.registerMapping` builds one wrapped handler and then attaches it to some
routes but not others:

```go
// internal/handler/router/router.go:62-92
func (r *Router) registerMapping(mapping config.Mapping) {
	router := r.Router.Host(mapping.From.Hostname).Subrouter()

	defaultHandler := r.prepareDefaultHandler(mapping)   // HAR( cache( options( proxy ) ) )

	for _, staticDir := range mapping.Statics {          // static → defaultHandler
		registerPrefixHandler(router, staticDir.Path, infra.Mddleware(middleware, defaultHandler))
	}

	registerMatchedRoutes(mapping.Mocks, ..., func(def *config.Mock) {
		registerRoute(createRoute(router, def.Matcher), r.container.MockHandler(&def.Response))
	})                                                    // ← bare handler, no wrapping

	registerMatchedRoutes(mapping.Scripts, ..., func(def *config.Script) {
		registerRoute(createRoute(router, def.Matcher), r.container.ScriptHandler(def))
	})                                                    // ← bare handler, no wrapping

	for _, rewrite := range mapping.Rewrites {            // rewrite → defaultHandler
		registerPathHandler(router, rewrite.From, infra.Mddleware(r.container.RewriteMiddleware(&rewrite), defaultHandler))
	}

	setDefaultHandler(router, defaultHandler)
}
```

and

```go
// internal/handler/router/router.go:94-108
func (r *Router) prepareDefaultHandler(mapping config.Mapping) contracts.Handler {
	defaultHandler := r.defaultHandler                    // the proxy
	if !mapping.OptionsHandling.Disabled { defaultHandler = Mddleware(optionsMw, defaultHandler) }
	if len(mapping.Cache) > 0            { defaultHandler = Mddleware(cacheMw,   defaultHandler) }
	if mapping.HAR.Enabled()             { defaultHandler = Mddleware(harMw,     defaultHandler) }
	return defaultHandler
}
```

Three concrete consequences:

**(a) Mocks and scripts are invisible to HAR and to the cache, and get no OPTIONS
handling.** A request served by a mock never passes through `harMw`, so
`docs/HAR-Collector.md`'s "records **every** request and response that passes
through a mapping" is false, and the mixed-feature example at the end of that
page does not do what it shows. A CORS preflight to a mocked path that also
matches the mock's matcher is answered by the mock body instead of the OPTIONS
middleware.

**(b) Route order silently shadows later routes.** `gorilla/mux` matches routes in
registration order, and statics are registered first. A static entry mounted at
`/` with an `index` therefore swallows every path in the mapping, including
mocks. This is exactly the configuration `docs/Static-File-Serving.md` presents as
"SPA with API Proxying". Verified:

```go
Statics: {Dir: "/dist", Path: "/", Index: "index.html"}
Mocks:   {Path: "/api/health", Response: {Code: 200, Raw: "MOCK-HEALTH"}}
```
```
GET /api/health -> 200 "SPA-INDEX"     // the mock is unreachable
```

**(c) A rewrite's output never re-enters routing.** `rewrite` middleware mutates
`request.URL` and calls `next`, where `next` is the *proxy chain*, not the
router. So `docs/Request-Rewriting.md`'s "Combining Rewrites with Other Features"
example — rewrite `/old-api/{r}` → `/v2/api/{r}` and mock `/v2/api/health` —
cannot work: the rewritten request goes straight upstream.

There is also dead diagnostics: the "host not mapped" fallback carries two
commented-out logging lines and returns a bare error
([`internal/handler/router/router.go:53-58`](../internal/handler/router/router.go#L53)),
so an unmapped host produces the giant 500 page from
[A14](A14-http-error-page-leaks-stack-traces.md) rather than a useful message.

## 2. Why it is an architectural problem

The mapping's middleware chain is a **per-mapping cross-cutting concern**, but it
is modelled as "a decorator on one particular terminal handler". That places the
composition at the wrong level of the tree:

```
current:                              intended:
host subrouter                        host subrouter
├── static  → [static]→PROXY-CHAIN    │  (wrapped once: HAR → cache → options)
├── mock    → MOCK                    ├── static → next
├── script  → SCRIPT                  ├── mock
├── rewrite → [rewrite]→PROXY-CHAIN   ├── script
└── default → PROXY-CHAIN             ├── rewrite → next
                                      └── default → proxy
```

Because the chain is built per-terminal-handler rather than per-mapping:

- Adding any new cross-cutting feature (metrics, request logging, auth stubbing,
  latency injection) requires remembering to attach it to five places, and
  forgetting one produces a silent partial feature — which is what already
  happened to HAR.
- The *semantics of a mapping* ("everything for `api.local` is recorded to
  `api.har`") is not expressible; only "everything proxied for `api.local`" is.
- Route precedence is implicit in the order of `for` loops rather than being a
  designed policy. Nothing in the config file tells the user that statics beat
  mocks, and nothing in the code documents it.

## 3. What the recommended approach is instead

**Wrap the subrouter, not the terminal handler.**

```go
func (r *Router) registerMapping(m config.Mapping) http.Handler {
	sub := mux.NewRouter()

	// terminal routes, in an explicit, documented precedence order
	registerMocks(sub, m.Mocks)        // most specific first
	registerScripts(sub, m.Scripts)
	registerRewrites(sub, m.Rewrites)  // rewrite → re-dispatch (see below)
	registerStatics(sub, m.Statics)    // path-prefix catch-alls last
	sub.NotFoundHandler = r.proxy
	sub.MethodNotAllowedHandler = r.proxy

	// one chain for the whole mapping
	return chain(sub, harMW(m), cacheMW(m), optionsMW(m))
}
```

Three design decisions fall out of this:

1. **Precedence becomes explicit.** Register the most specific matchers first and
   catch-all prefixes (statics, especially at `/`) last. `registerMatchedRoutes`
   ([`internal/handler/router/routes.go:5`](../internal/handler/router/routes.go#L5))
   already implements a two-pass "specific before path-only" scheme for mocks and
   scripts; generalise that policy across *all* route kinds instead of applying it
   within each kind. Document the resulting order in `docs/Configuration.md`.

2. **OPTIONS should be outermost, before route matching.** A CORS preflight is a
   transport-level concern; it should be answered before the router decides
   whether the path is a mock, a static file or a proxy target. Wrapping the
   subrouter achieves that for free.

3. **Rewrites should re-dispatch, not short-circuit.** After mutating the URL,
   call the *subrouter* again (with a re-entrancy guard to prevent loops, e.g. a
   context counter capped at N rewrites). That makes "rewrite then mock" work and
   matches the documented behaviour. If re-dispatch is judged too risky, then the
   docs must be corrected instead (see [D06](D06-rewrite-docs-do-not-match-behaviour.md)) —
   but silently doing neither is the worst option.

## 4. Why the proposed approach is better

- **HAR and cache become truthful.** "Everything for this mapping" is what users
  expect and what the docs already promise; today they get "everything except the
  interesting locally-served responses".
- **A new cross-cutting middleware is one line**, attached at one place, with no
  way to forget a branch.
- **SPA + mocks works**, which is the single most common real-world uncors
  configuration and is currently broken in the documented form.
- **Preflight handling stops depending on route shape.** Today whether OPTIONS is
  handled depends on whether the path happens to hit a mock — a surprising
  coupling.
- The subrouter-per-mapping structure also makes per-mapping teardown natural,
  which supports [A02](A02-di-container-leaks-resources-on-every-config-reload.md)'s
  generation scope.

## 5. Trade-offs and migration considerations

- **This is a behaviour change and needs a release note.** Mock and script
  responses will start appearing in HAR files and can start being cached. Caching
  a mock is almost certainly *not* wanted — exclude terminal handlers that are
  already local from the cache middleware, or (simpler) keep `cache` inside the
  proxy branch and move only HAR and OPTIONS outward. Deciding this explicitly is
  the point; today it is decided by accident.
- **Changing route precedence can break existing user configs** that (perhaps
  unknowingly) rely on statics winning. Mitigate with a clear entry in
  `docs/Migration-Guide.md` and, if desired, a startup warning when a static path
  of `/` with an `index` shadows a configured mock — that warning is worth adding
  regardless of whether precedence changes.
- **Rewrite re-dispatch needs a loop guard.** A config with `/a → /b` and
  `/b → /a` must terminate; a context-carried depth counter with a small limit
  (say 8) and a 508-style error is sufficient.
- Body-capturing middleware moving outward increases the number of responses whose
  bodies are buffered; pair this change with the body-size cap from
  [A07](A07-response-writer-drops-streaming-capabilities.md).

## 6. Code references

| What | Where |
| --- | --- |
| Per-mapping registration | [`internal/handler/router/router.go:62`](../internal/handler/router/router.go#L62) |
| Chain built around the proxy only | [`internal/handler/router/router.go:94`](../internal/handler/router/router.go#L94) |
| Mocks registered bare | [`internal/handler/router/router.go:70`](../internal/handler/router/router.go#L70) |
| Scripts registered bare | [`internal/handler/router/router.go:77`](../internal/handler/router/router.go#L77) |
| Statics registered first (shadowing) | [`internal/handler/router/router.go:66`](../internal/handler/router/router.go#L66), [`internal/handler/router/router_helpres.go:45`](../internal/handler/router/router_helpres.go#L45) |
| Rewrite short-circuits to proxy | [`internal/handler/router/router.go:85`](../internal/handler/router/router.go#L85), [`internal/handler/rewrite/handler.go:23`](../internal/handler/rewrite/handler.go#L23) |
| Two-pass specificity (mocks/scripts only) | [`internal/handler/router/routes.go:5`](../internal/handler/router/routes.go#L5) |
| Commented-out unmapped-host logging | [`internal/handler/router/router.go:53`](../internal/handler/router/router.go#L53) |
| Docs contradicted | [`docs/HAR-Collector.md`](../docs/HAR-Collector.md), [`docs/Static-File-Serving.md`](../docs/Static-File-Serving.md), [`docs/Request-Rewriting.md`](../docs/Request-Rewriting.md) |
