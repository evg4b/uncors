# D05 — HAR documentation overstates what is recorded and when the file is closed

**Severity:** Medium-High
**Area:** Documentation vs implementation — HAR collector

---

## 1. What is wrong

`docs/HAR-Collector.md` and `ARCHITECTURE.md` make four claims that the
implementation does not support.

### Claim 1 — "records **every** request and response that passes through a mapping"

> The HAR Collector records every request and response that passes through a
> mapping to an HTTP Archive (HAR 1.2) file.
> — `docs/HAR-Collector.md`, opening paragraph

> The HAR collector records **every** proxied request/response pair
> — `ARCHITECTURE.md`

The HAR middleware is attached only to the mapping's *proxy* chain
([`internal/handler/router/router.go:104`](../internal/handler/router/router.go#L104)),
which is used by: the mapping default route, static-file misses, and rewrites.
It is **not** attached to mock routes
([`internal/handler/router/router.go:70`](../internal/handler/router/router.go#L70)) or
script routes ([`:77`](../internal/handler/router/router.go#L77)). See
[A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md).

So a mapping that mocks `/api/feature-flags` records nothing for it — and the
page's own "Combine with Other Features" example is precisely such a mapping:

```yaml
har: ./recordings/app.har
cache: [/api/config]
mocks:
  - path: /api/feature-flags
    ...
statics:
  - path: /
    dir: ./dist
    index: index.html
```

In this example the static mount at `/` additionally shadows everything
([D04](D04-spa-with-mocks-example-is-broken.md)), so the produced HAR would contain
static-file responses and nothing else.

### Claim 2 — "Flushed and closed on shutdown **or when the configuration is reloaded**"

> - **Flushed and closed** on shutdown or when the configuration is reloaded. All
>   buffered entries are written before the file handle is released.
> — `docs/HAR-Collector.md`, "File Lifecycle"

> **Lifecycle management** — `Writer` implements `io.Closer`. The app registers
> each writer via `registerCloser`; on shutdown or config reload it calls
> `Close()`…
> — `ARCHITECTURE.md`

No reload path closes anything. Writers are appended to a container-wide slice
([`internal/di/public_api.go:136`](../internal/di/public_api.go#L136)) that is drained
only by `Container.Close()` at process exit
([`internal/di/container.go:79`](../internal/di/container.go#L79)). A reload creates an
*additional* writer for the same file while the previous one keeps running — see
[A02](A02-di-container-leaks-resources-on-every-config-reload.md), where this is shown
to corrupt the output file. There is no `registerCloser` symbol in the repository.

### Claim 3 — "Updated atomically **after every request**"

> - **Updated atomically** after every request — UNCORS writes to a temporary file
>   then renames it…

The writer batches: it flushes after draining whatever is queued
([`internal/handler/har/writer.go:71-80`](../internal/handler/har/writer.go#L72)), so
under load one flush covers many requests. The atomicity claim is correct; the
"after every request" part is not, and the file can lag behind. (It should lag
further still — see [A15](A15-har-writer-rewrites-the-whole-archive-on-every-batch.md).)

### Claim 4 — "Created on the first captured request (**parent directory must
exist**)"

The writer calls `os.MkdirAll(filepath.Dir(w.path), 0o755)` before writing
([`internal/handler/har/writer.go:129`](../internal/handler/har/writer.go#L129)), so the
parent directory is created automatically. This claim is wrong in the *safe*
direction, but it will make users create directories they do not need to.

### Minor: the `creator` block is misleading

Recorded archives always report `"creator": {"name": "uncors", "version": "dev"}`
([`internal/handler/har/writer.go:15`](../internal/handler/har/writer.go#L15)) regardless
of the real build version, so a shared HAR file cannot be attributed to a release.

## 2. Why it matters

HAR recording is a debugging and evidence-gathering feature. Its value depends
entirely on the user trusting that what is in the file is what happened. Two of
the claims above break that trust in the worst way:

- **Silently missing entries** (mocks, scripts) mean a developer comparing "what
  the frontend sent" against a HAR will conclude requests were never made.
- **Silently corrupted files after a config save** mean the recording can lose
  entries with no error anywhere (HAR write failures are logged to a discarded
  logger — [A13](A13-logging-and-output-are-three-parallel-systems.md)).

A user cannot detect either condition from the artefact itself.

## 3. Recommended fix

**Code (preferred), then docs:**

1. **Move the HAR middleware to wrap the whole mapping subrouter** so mocks,
   scripts and statics are recorded — [A06](A06-cross-cutting-middleware-attached-to-one-branch-only.md).
   This makes Claim 1 true rather than deleting it.
2. **Close the previous generation's writers on reload** —
   [A02](A02-di-container-leaks-resources-on-every-config-reload.md) — making Claim 2 true.
3. **Set `creator.version` from the build version** (`di.Container.Version()` is
   already available at the `har.NewWriter` call site).

**Docs, in the same change:**

4. Restate Claim 3 accurately: "flushed shortly after each request (batched under
   load); the file on disk is always a complete, valid HAR."
5. Delete the "parent directory must exist" parenthetical.
6. Remove the `registerCloser` reference from `ARCHITECTURE.md`
   ([D07](D07-architecture-and-claude-md-are-stale.md) covers the rest of that file).
7. Document what is **not** recorded, if anything remains excluded after (1) —
   for example, if OPTIONS preflights answered locally are deliberately skipped,
   say so.
8. Fix the "Combine with Other Features" example so it actually works (narrow the
   static mount), or it will keep demonstrating [D04](D04-spa-with-mocks-example-is-broken.md).

**Add a test that proves Claim 1.** An integration test in
`tests/integration/har` asserting that a mocked path and a scripted path appear in
the produced archive would lock the behaviour in.

## 4. Why this is better

- Users can trust the artefact, which is the entire point of the feature.
- Making the code match the docs (rather than the reverse) preserves a genuinely
  useful capability — "record everything this mapping did" is more valuable than
  "record everything this mapping proxied", especially when debugging why a mock
  did or did not fire.
- Version attribution makes shared HARs actionable in bug reports.

## 5. Trade-offs and migration considerations

- **Recording mocks and scripts increases HAR size and body capture volume**, and
  interacts with the unbounded capture problem in
  [A07](A07-response-writer-drops-streaming-capabilities.md) — pair the changes.
- **If mocks are recorded, users may want to exclude them.** Consider a
  `har.include: [proxy, mock, script, static]` option rather than an all-or-nothing
  switch; default to everything.
- **Closing writers on reload changes file timing**: a reload now forces a flush,
  which is what the docs promise but is a visible behaviour change for anyone
  tailing the file.
- Documentation lives in `docs/` and is published as a GitHub wiki; make sure the
  wiki sync runs after these edits, and update `ARCHITECTURE.md` in the same PR so
  the two do not diverge again.

## 6. Code and document references

| What | Where |
| --- | --- |
| HAR attached to proxy chain only | [`internal/handler/router/router.go:104`](../internal/handler/router/router.go#L104) |
| Mocks/scripts registered unwrapped | [`internal/handler/router/router.go:70`](../internal/handler/router/router.go#L70), [`:77`](../internal/handler/router/router.go#L77) |
| Writers appended, closed only at exit | [`internal/di/public_api.go:136`](../internal/di/public_api.go#L136), [`internal/di/container.go:79`](../internal/di/container.go#L79) |
| Batched flush | [`internal/handler/har/writer.go:72`](../internal/handler/har/writer.go#L72) |
| `MkdirAll` on flush | [`internal/handler/har/writer.go:129`](../internal/handler/har/writer.go#L129) |
| Hard-coded creator version | [`internal/handler/har/writer.go:15`](../internal/handler/har/writer.go#L15) |
| Claims | [`docs/HAR-Collector.md`](../docs/HAR-Collector.md) (opening, "File Lifecycle", "Combine with Other Features"), [`ARCHITECTURE.md`](../ARCHITECTURE.md) ("HAR Collector") |
