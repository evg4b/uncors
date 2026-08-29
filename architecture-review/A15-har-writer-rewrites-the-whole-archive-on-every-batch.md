# A15 — The HAR writer re-serialises and rewrites the entire archive after every batch

**Severity:** Medium-High (performance, degrades quadratically)
**Area:** HAR collector

---

## 1. What is wrong with the current approach

`har.Writer` keeps **every entry ever recorded** in memory and rewrites the whole
file each time it flushes:

```go
// internal/handler/har/writer.go:30-38
type Writer struct {
	path    string
	entries chan Entry
	...
	all     []Entry        // ← grows forever
}

// internal/handler/har/writer.go:71-88
for {
	select {
	case entry := <-w.entries:
		w.append(entry)
		w.drain()          // take everything else queued
		w.flush()          // ← rewrite the entire archive
	case <-w.done:
		w.drain(); w.flush(); return
	}
}

// internal/handler/har/writer.go:109-148
func (w *Writer) flush() {
	w.mu.Lock()
	snapshot := make([]Entry, len(w.all))
	copy(snapshot, w.all)                       // ← full copy of the slice header set
	w.mu.Unlock()
	archive := HAR{Log: Log{... Entries: snapshot}}
	data, err := json.MarshalIndent(archive, "", "  ")   // ← full re-serialisation, indented
	...
	os.WriteFile(tmp, data, harFileMode)                 // ← full write
	os.Rename(tmp, w.path)
}
```

Cost model: with *N* entries recorded, the total bytes marshalled and written is
`O(N²)` in the number of flushes. Concretely, browsing a modern SPA through a
mapping with `har:` enabled easily produces 2 000+ entries; if the traffic is
bursty enough that batching does not coalesce them, that is 2 000 full
serialisations of a growing document — the last few of which each marshal and
write tens of megabytes.

The memory side is just as bad. `Entry.Response.Content.Text` holds the **entire
decoded response body as a string** ([`internal/handler/har/content.go:44`](../internal/handler/har/content.go#L44)),
and `w.all` retains all of them for the process lifetime. Recording a session that
downloads 500 MB of assets pins 500 MB (plus the marshalling buffer, plus the
`snapshot` copy) in the heap — see [A07](A07-response-writer-drops-streaming-capabilities.md)
for the uncapped capture that feeds it.

Additional issues:

- **`json.MarshalIndent`** doubles-to-triples the output size and costs
  meaningfully more than `Marshal`. For a machine-readable artefact opened in
  DevTools, indentation is not required.
- **The temp path is `w.path + ".tmp"`**, a fixed name
  ([`internal/handler/har/writer.go:137`](../internal/handler/har/writer.go#L137)). Two
  writers on the same path — which is exactly what a config reload creates today
  ([A02](A02-di-container-leaks-resources-on-every-config-reload.md)) — will interleave
  writes to the same temp file and rename each other's partial output over the
  target.
- **The writer bypasses `afero.Fs`** and calls `os.MkdirAll`/`os.WriteFile`
  directly ([`internal/handler/har/writer.go:129`](../internal/handler/har/writer.go#L129)),
  unlike every other filesystem consumer in the project. That makes it untestable
  with the in-memory filesystem the rest of the test suite uses, and it means the
  `--config`-relative filesystem abstraction does not apply to HAR output.
- **`creatorVersion` is hard-coded to `"dev"`**
  ([`internal/handler/har/writer.go:15`](../internal/handler/har/writer.go#L15)) instead of
  the real build version, so recorded archives cannot be attributed to a release.

## 2. Why it is an architectural problem

- **The storage model does not match the access pattern.** HAR is an
  append-oriented log; it is being persisted with a whole-document rewrite. The
  mismatch is what produces the quadratic behaviour, and no amount of
  micro-optimising `flush` fixes it.
- **Retaining the full archive in memory couples recording duration to memory
  usage** with no bound and no configuration. A long-running dev session with HAR
  on is a slow memory leak by design.
- **"Atomic file update" was chosen as the durability strategy**, and it is a
  reasonable one — but it forces the full rewrite. The design goal ("the file is
  always valid if you open it mid-session") can be met far more cheaply.

## 3. What the recommended approach is instead

**Option A (recommended): append-only journal + assemble on demand.**

Write each entry as one line of JSON to a `.har.jsonl` sidecar as it arrives
(`O(1)` per entry, no retained state), and materialise the real `.har` document:

- on `Close()`,
- and on a slow timer (e.g. every 2 s, only if dirty) so the file is usable
  mid-session.

```go
func (w *Writer) run() {
	tick := time.NewTicker(materialiseInterval)
	for {
		select {
		case e := <-w.entries: w.appendJournal(e); w.dirty = true
		case <-tick.C:         if w.dirty { w.materialise(); w.dirty = false }
		case <-w.done:         w.drainJournal(); w.materialise(); return
		}
	}
}
```

Entries no longer need to be retained in memory at all — `materialise` streams the
journal into the HAR envelope with `json.Encoder`.

**Option B (simpler): stream the HAR envelope directly.**

Keep a single open file, write the fixed prefix
(`{"log":{"version":"1.2","creator":{…},"entries":[`) once, append each entry
followed by a comma, and write the suffix (`]}}`) on `Close`. To keep the file
readable mid-session, periodically write the suffix at the current offset without
advancing it (or maintain a small `.har.partial` copy). This is `O(1)` per entry
and needs no journal, at the cost of a slightly more delicate file format
handling.

**In both options:**

- Drop `MarshalIndent` for `Marshal` (or make indentation an option).
- Route all file I/O through the injected `afero.Fs` so HAR is testable with
  `MemMapFs` and consistent with the rest of the project.
- Make the temp path unique per writer (`fmt.Sprintf("%s.%d.tmp", path, id)`).
- Use the real build version for `creator.version`.
- Add a `max-entries` / `max-size` option to bound a long session, with clearly
  documented behaviour when the bound is hit (stop recording vs. rotate).

## 4. Why the proposed approach is better

- **Per-entry cost becomes constant** instead of proportional to session length;
  long recording sessions stop degrading.
- **Memory stops growing with session length.** Entries are written and released.
- **The concurrent-writer hazard disappears** (unique temp files), and combined
  with [A02](A02-di-container-leaks-resources-on-every-config-reload.md) the
  reload-corruption bug is fully closed.
- **The collector becomes testable in-memory**, matching the rest of the suite.
- The stated design goal from `ARCHITECTURE.md` — "the file is always in a valid
  state" — is preserved, just paid for at a much lower rate.

## 5. Trade-offs and migration considerations

- **Option A adds a sidecar file** that must be cleaned up on `Close`. Users who
  kill the process with `SIGKILL` will find a `.har.jsonl` next to their `.har`;
  recovering it is a feature (a crash-resistant recording), but it needs
  documenting.
- **The "always valid" guarantee weakens slightly** if materialisation is on a
  timer: opening the file mid-session may show entries up to 2 s stale. That is a
  much better trade than quadratic I/O, but it changes the documented promise in
  `docs/HAR-Collector.md` ("Updated atomically **after every request**") and the
  doc must be corrected either way
  ([D05](D05-har-docs-overstate-coverage-and-lifecycle.md)).
- **Bounding memory requires the body cap from [A07](A07-response-writer-drops-streaming-capabilities.md)**;
  without it, a single 2 GB response still blows up before the writer sees it.
- Switching to `afero.Fs` means `har.NewWriter` gains an `fs` parameter — a
  one-line change at its only call site
  ([`internal/di/public_api.go:134`](../internal/di/public_api.go#L134)).

## 6. Code references

| What | Where |
| --- | --- |
| Unbounded in-memory archive | [`internal/handler/har/writer.go:32`](../internal/handler/har/writer.go#L32), [`:106`](../internal/handler/har/writer.go#L106) |
| Full rewrite per batch | [`internal/handler/har/writer.go:109`](../internal/handler/har/writer.go#L109) |
| `MarshalIndent` | [`internal/handler/har/writer.go:120`](../internal/handler/har/writer.go#L120) |
| Fixed temp path | [`internal/handler/har/writer.go:137`](../internal/handler/har/writer.go#L137) |
| Direct `os` I/O, bypassing afero | [`internal/handler/har/writer.go:129`](../internal/handler/har/writer.go#L129), [`:138`](../internal/handler/har/writer.go#L138) |
| Hard-coded creator version | [`internal/handler/har/writer.go:15`](../internal/handler/har/writer.go#L15) |
| Whole body retained as a string | [`internal/handler/har/content.go:44`](../internal/handler/har/content.go#L44) |
| Writer construction | [`internal/di/public_api.go:134`](../internal/di/public_api.go#L134) |
