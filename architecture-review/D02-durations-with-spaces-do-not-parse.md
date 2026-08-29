# D02 — Every documented multi-unit duration (`1m 30s`, `1h 30m`, `2s 500ms`) fails to parse

**Severity:** High (documented examples abort startup)
**Area:** Documentation vs implementation — configuration format

---

## 1. What is wrong

`Response.Delay` and `CacheConfig.ExpirationTime` are plain `time.Duration`
fields decoded by `gopkg.in/yaml.v3`:

```go
// internal/config/response.go:15
Delay   time.Duration `yaml:"delay"`
// internal/config/cache_config.go:17
ExpirationTime time.Duration `yaml:"expiration-time,omitempty"`
```

`yaml.v3` decodes a string into `time.Duration` via `time.ParseDuration`, which
does **not** accept spaces between components. Verified:

```
delay: 1m 30s   → yaml: unmarshal errors: line 1: cannot unmarshal !!str `1m 30s` into time.Duration
delay: 1m30s    → 1m30s  (ok)
```

Because the decode error propagates out of `readYAMLFile`
([`internal/config/config.go:71`](../internal/config/config.go#L71)) and out of
`LoadConfiguration`, a config using a documented spaced duration **does not start
the proxy at all** — it exits with
`failed to read config file '…': While parsing config: …`.

The spaced form is documented as the canonical example in four places:

| Document | Line | Value |
| --- | --- | --- |
| `docs/Configuration.md` | 120 | `delay: 1m 30s` |
| `docs/Response-Mocking.md` | 144 | `delay: 1m 30s   # 1 minute 30 seconds` |
| `docs/Response-Mocking.md` | 145 | `delay: 2s 500ms  # 2.5 seconds` |
| `docs/Response-Caching.md` | 79 | `1h 30m - 1 hour 30 minutes` |

`docs/Response-Mocking.md` goes further and states the format explicitly:

> Format: `<number><unit> [<number><unit> ...]`

which describes a space-separated grammar the parser does not implement.

Single-component values used elsewhere in the docs (`delay: 10s`, `delay: 500ms`,
`expiration-time: 10m`, `delay: 3s`, `delay: 100ms`) all work correctly, so the
defect is specific to multi-unit values with spaces.

## 2. Why this happened, and why it matters structurally

This is almost certainly a regression from the migration off `spf13/viper`. Viper
uses `mapstructure` with a `StringToTimeDurationHookFunc`, and projects commonly
add a hook that strips whitespace before calling `time.ParseDuration`. The
project's own test file is named
[`internal/config/time_decode_hook_test.go`](../internal/config/time_decode_hook_test.go) —
"decode hook" is viper/mapstructure terminology — but the file it was written for
no longer exists, and its only remaining duration test case is deliberately
labelled `"duration without spaces"`. The test suite therefore *documents* the
gap rather than covering it.

The structural point: **the config format is defined by whatever `yaml.v3` happens
to do with each Go type**, and nothing checks that against the documented format.
The same class of drift produced [D01](D01-debug-flag-does-not-exist.md) and
[D08](D08-schema-json-is-stale-and-unused.md).

## 3. Recommended fix

**Preferred — support the documented syntax.** Give the durations an explicit
unmarshaller so the format is owned by the project rather than inherited:

```go
// internal/config
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {           // also accept plain integers as ns? decide.
		return err
	}
	parsed, err := time.ParseDuration(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
	if err != nil {
		return fmt.Errorf("invalid duration %q: expected e.g. 500ms, 30s, 1m30s: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}
```

Use it for `Response.Delay` and `CacheConfig.ExpirationTime`. Add the matching
`MarshalYAML` so the round-trip used by the integration harness
([`testing/integration/proxy.go`](../testing/integration/proxy.go) marshals the config
back to YAML) stays lossless.

**Alternative — fix the docs.** Replace every spaced example with the compact
form (`1m30s`, `2s500ms`, `1h30m`) and correct the stated grammar to
`<number><unit>[<number><unit>…]` with no spaces, linking to
`time.ParseDuration`'s documentation.

**Either way — add tests.** Extend
[`internal/config/time_decode_hook_test.go`](../internal/config/time_decode_hook_test.go)
with the exact strings that appear in the documentation, and add a test that
parses every YAML block in `docs/` (see §5).

## 4. Why the recommended approach is better

- **Users copy examples verbatim.** The `1m 30s` form appears in the two most-read
  reference pages; anyone following them gets a startup failure whose message
  (`cannot unmarshal !!str '1m 30s' into time.Duration`) does not obviously point
  at "remove the space".
- **Supporting the space is trivial and strictly more permissive** — no existing
  config breaks, and the friendlier form the docs promise starts working.
- **A custom type puts the format under the project's control**, which is where a
  user-facing config grammar belongs; it also lets the error message name the
  field and suggest valid values, which `yaml.v3`'s generic message cannot.

## 5. Trade-offs and migration considerations

- **Changing the field type to a named `Duration`** ripples through
  [`internal/config/response.go`](../internal/config/response.go),
  [`internal/config/cache_config.go`](../internal/config/cache_config.go), their
  validators (`ValidateDuration`,
  [`internal/config/validate_primitives.go`](../internal/config/validate_primitives.go)),
  the mock handler's `waitDelay`
  ([`internal/handler/mock/handler.go:110`](../internal/handler/mock/handler.go#L110)) and
  the cache TTL. All conversions are `time.Duration(d)`; mechanical but touches
  several files. Keeping `time.Duration` and adding the unmarshaller on a wrapper
  used only for decoding is a lighter alternative.
- **Marshalling must round-trip.** The integration harness and
  [`internal/cli/run_uncors_test.go`](../internal/cli/run_uncors_test.go) marshal
  `UncorsConfig` back to YAML; without a `MarshalYAML`, a named type will encode as
  a bare integer (nanoseconds), which still parses but makes generated configs
  unreadable.
- **Add a docs-example test.** A test that walks `docs/*.md`, extracts every
  ```yaml fenced block that contains a `mappings:` or `cache-config:` key, and runs
  it through `LoadConfiguration` against a stub filesystem would have caught this,
  [D01](D01-debug-flag-does-not-exist.md) and [D10](D10-mapping-and-cors-behaviour-does-not-match-docs.md)
  automatically. That is the highest-value durable fix in this whole documentation
  section.
- Note the fenced examples reference files and directories that do not exist
  (`./dist`, `~/mocks/...`), so such a test needs a permissive stub `afero.Fs` that
  reports every path as present, or a validation-skipping parse mode.

## 6. Code and document references

| What | Where |
| --- | --- |
| `Delay` field | [`internal/config/response.go:15`](../internal/config/response.go#L15) |
| `ExpirationTime` field | [`internal/config/cache_config.go:17`](../internal/config/cache_config.go#L17) |
| YAML decode error path | [`internal/config/config.go:71`](../internal/config/config.go#L71) |
| Test that documents the gap | [`internal/config/time_decode_hook_test.go`](../internal/config/time_decode_hook_test.go) |
| Documented spaced durations | [`docs/Configuration.md`](../docs/Configuration.md) L120, [`docs/Response-Mocking.md`](../docs/Response-Mocking.md) L138–145, [`docs/Response-Caching.md`](../docs/Response-Caching.md) L79 |
