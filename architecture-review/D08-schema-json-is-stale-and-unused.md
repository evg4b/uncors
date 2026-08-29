# D08 — `schema.json` is out of sync with the config structs and is never used at runtime

**Severity:** Medium
**Area:** Documentation vs implementation — config schema

---

## 1. What is wrong

`schema.json` (12 KB, repository root) is presented by both internal documents as
the config validation mechanism
([D07](D07-architecture-and-claude-md-are-stale.md)) and by the contribution
instructions as something to update when adding a config option:

> **New Config Option:** 1. Add field to `internal/config/` struct 2. **Update
> `schema.json` with validation rules** 3. Add parser/validator if complex
> — `CLAUDE.md`

Two problems.

### (a) It is not used by the program

`LoadConfiguration` reads YAML and runs hand-written validators
([`internal/config/config.go:26`](../internal/config/config.go#L26)); nothing loads
`schema.json`. Its only reader is [`tests/schema/schema_test.go`](../tests/schema/schema_test.go),
which validates fixture files against it. So the schema constrains fixtures, not
users' configs.

It is presumably intended as an editor-support artefact (YAML Language Server
`$schema` comments), but nothing in the docs tells users how to reference it, and
the project's own [`.uncors.yaml`](../.uncors.yaml) does not.

### (b) It has drifted from the config structs

Comparing the schema's properties against the Go types:

| Schema | Go struct | Status |
| --- | --- | --- |
| root `debug` | *(absent)* | **Extra** — the schema blesses a key the program ignores ([D01](D01-debug-flag-does-not-exist.md)) |
| root `proxy`, `mappings`, `cache-config` | present | ok |
| *(absent)* `interactive` | `Interactive` is `yaml:"-"` | consistent (CLI-only), but worth a note |
| `cache-config.clear-time` | *(absent)* | **Extra** — no such field in [`CacheConfig`](../internal/config/cache_config.go#L16) |
| *(absent)* `cache-config.max-size` | `MaxSize int64` | **Missing** — a real, documented option the schema rejects |
| `OptionsHandling.status` | field is `Code int` (`yaml:"code"`) | **Wrong name** — a valid `code:` fails schema validation, an invalid `status:` passes |
| `Mapping` properties | match | ok |
| `Script`, `Mock`, `StaticDirectory`, `Rewrite`, `HARConfig` | match | ok |

So a user who wires the schema into their editor gets red squiggles on
`max-size` and `code` (both correct) and no warning on `clear-time`, `status` or
`debug` (all meaningless). The schema is actively misleading in four places.

Note that `docs/Response-Caching.md` documents `max-size` with a default of
`104857600`, and `internal/config/cache_config.go:37` **rejects a config where
`max-size <= 0`** — so `max-size` is not merely supported, it is mandatory-by-default
and completely absent from the schema.

## 2. Why it matters

- **Contributors are instructed to maintain a file that has no effect**, which is
  why it drifted: there is no feedback when it is wrong.
- **The one audience it could serve — editor users — is not told it exists.** A
  `$schema` reference in the docs would make config authoring dramatically easier
  for a YAML format with this many nested optional keys, so the artefact is
  valuable; it is just disconnected.
- **Two validation definitions with no link between them** means every future
  config change can diverge again.

## 3. Recommended fix

Pick one of two coherent directions.

### Option A — make the schema real (recommended)

1. **Generate `schema.json` from the Go structs** rather than hand-writing it.
   The struct tags already carry the field names; a small `go generate` program
   using reflection over `config.UncorsConfig` (or a library such as
   `invopop/jsonschema`) can emit the schema, with enums/patterns supplied via
   struct tags or a small override table for the shorthand forms
   (`Mapping`, `HARConfig`, `StaticDirectories` all have custom `UnmarshalYAML`
   and need `oneOf` branches — that is the part worth hand-maintaining).
2. **Keep `tests/schema` as the guard**: it now verifies that the generated schema
   accepts the valid fixtures and rejects the invalid ones, which becomes a real
   round-trip test rather than a test of a parallel artefact.
3. **Document the `$schema` line** in `docs/Configuration.md`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/evg4b/uncors/main/schema.json
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
```

   and add it to the project's own [`.uncors.yaml`](../.uncors.yaml) as a worked
   example. (The repo already uses this pattern for GoReleaser —
   [`.goreleaser.yaml:1`](../.goreleaser.yaml#L1) — so the convention is familiar.)
4. **Optionally** validate against the schema at load time too, which would make
   `ARCHITECTURE.md`/`CLAUDE.md` true; but note this largely duplicates the
   hand-written validators, which produce better messages (field paths, TLS
   hints). Editor-time schema + runtime Go validation is the better split — just
   document it that way.

### Option B — delete it

If generating and publishing the schema is not wanted, delete `schema.json` and
`tests/schema/`, and remove the "update `schema.json`" step from `CLAUDE.md` and
`ARCHITECTURE.md`. A wrong schema is worse than no schema.

**In either case, fix the four drifted entries now** (`debug`, `clear-time`,
`max-size`, `status`) so the file is not misleading while the direction is
decided.

## 4. Why the recommended approach is better

- **Generation removes the drift mechanism entirely** — the schema cannot disagree
  with the structs because it is derived from them.
- **Editor completion and inline validation is genuinely high-value** for this
  config format: nested `mappings` with six optional sub-sections, three shorthand
  syntaxes, and duration/glob string formats. This is the cheapest large UX win in
  the project.
- **`tests/schema` gains a purpose**: it currently tests that a hand-written file
  matches hand-written fixtures, neither of which is the program.
- **`CLAUDE.md`'s contribution checklist becomes accurate** — "add the field, run
  `go generate`" instead of "hand-edit a JSON file nobody reads".

## 5. Trade-offs and migration considerations

- **Custom `UnmarshalYAML` implementations do not reflect.** `Mapping`'s
  `from: to` shorthand ([`internal/config/mapping.go:34`](../internal/config/mapping.go#L34)),
  `HARConfig`'s scalar form ([`internal/config/har.go:30`](../internal/config/har.go#L30)),
  and `StaticDirectories`' map form ([`internal/config/static.go:43`](../internal/config/static.go#L43))
  all accept shapes a naive generator will miss. Plan for a hand-maintained
  `oneOf` overlay for exactly these three types — the current schema already has
  those `oneOf` branches, so they can be lifted across.
- **Publishing the schema at a stable URL** creates a compatibility obligation:
  older uncors versions will validate configs against `main`'s schema if users
  reference the branch URL. Prefer a versioned path
  (`.../v0.6.1/schema.json`) or ship the schema in the release assets.
- **Adding runtime schema validation would add a dependency**
  (`xeipuuv/gojsonschema` is currently a test-only dependency) and duplicate error
  reporting. Recommend against it; keep runtime validation in Go.
- **`time.Duration` fields need a schema `pattern`**, and it must match whatever
  [D02](D02-durations-with-spaces-do-not-parse.md) settles on — do these two
  together.

## 6. Code and document references

| What | Where |
| --- | --- |
| Schema file | [`schema.json`](../schema.json) |
| Only consumer | [`tests/schema/schema_test.go`](../tests/schema/schema_test.go) |
| Runtime config loading (no schema) | [`internal/config/config.go:26`](../internal/config/config.go#L26) |
| `CacheConfig` (no `clear-time`, has `max-size`) | [`internal/config/cache_config.go:16`](../internal/config/cache_config.go#L16), [`:35`](../internal/config/cache_config.go#L35) |
| `OptionsHandling.Code` (schema says `status`) | [`internal/config/options_handling.go:5`](../internal/config/options_handling.go#L5) |
| Shorthand unmarshallers needing `oneOf` | [`internal/config/mapping.go:34`](../internal/config/mapping.go#L34), [`internal/config/har.go:30`](../internal/config/har.go#L30), [`internal/config/static.go:43`](../internal/config/static.go#L43) |
| `$schema` convention already used | [`.goreleaser.yaml:1`](../.goreleaser.yaml#L1) |
| Contribution instruction | [`CLAUDE.md`](../CLAUDE.md), [`ARCHITECTURE.md`](../ARCHITECTURE.md) |
