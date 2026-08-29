# D01 — The documented `--debug` flag and `debug:` config key do not exist, and `--interactive` is undocumented

**Severity:** High (documented instructions fail outright)
**Area:** Documentation vs implementation — CLI surface

---

## 1. What is wrong

### `--debug` does not exist

The complete flag set is defined in one place:

```go
// internal/config/flags.go:9-26
flags.StringSliceP("to", "t", ...)
flags.StringSliceP("from", "f", ...)
flags.String("proxy", "", ...)
flags.StringP("config", "c", ...)
flags.Bool("interactive", true, ...)
flags.BoolP("version", "v", false, ...)   // then MarkHidden("version")
```

There is no `--debug`. Because the flag set is created with
`pflag.ContinueOnError` and the parse error is returned as
`"failed parsing flags: unknown flag: --debug"`
([`internal/config/config.go:31`](../internal/config/config.go#L31)), running any
documented `--debug` command **exits with an error and starts nothing**.

`--debug` is documented six times:

| Document | Line |
| --- | --- |
| `docs/Configuration.md` | 93 (Global Configuration table) |
| `docs/Migration-Guide.md` | 326 |
| `docs/Troubleshooting.md` | 159, 209, 446, 496 |

Troubleshooting is the worst placement: a user who is already stuck is told four
separate times to run a command that fails.

### `debug:` in YAML does not exist either

`docs/Configuration.md:105` and `:144`, `docs/Real-World-Examples.md:631` and
`docs/Troubleshooting.md:365` all show `debug: false` as a global config
property. `UncorsConfig` has four fields and none of them is `Debug`:

```go
// internal/config/config.go:16-21
type UncorsConfig struct {
	Mappings    Mappings    `yaml:"mappings"`
	Proxy       string      `yaml:"proxy"`
	CacheConfig CacheConfig `yaml:"cache-config"`
	Interactive bool        `yaml:"-"`
}
```

The YAML decoder does not use `KnownFields`
([`internal/config/config.go:70`](../internal/config/config.go#L70)), so `debug: false`
is **silently ignored** — arguably worse than an error, because the user believes
debug output is enabled and concludes the tool is broken when nothing appears.

`docs/Home.md:97` compounds this by describing "debug mode" as a core global
configuration concept.

### The real diagnostic mechanism is documented nowhere

Log output is controlled exclusively by the `UNCORS_LOGGING` environment
variable ([`internal/infra/loggings.go:16`](../internal/infra/loggings.go#L16)), and
by default all internal logging is discarded
([A13](A13-logging-and-output-are-three-parallel-systems.md)). `UNCORS_LOGGING`
appears in `CLAUDE.md` and nowhere in `docs/`.

### `--interactive` is undocumented

`--interactive` (default `true`) selects between a full-screen BubbleTea TUI and
plain output ([`internal/config/flags.go:20`](../internal/config/flags.go#L20)). It is
the single most behaviour-defining flag in the tool — it decides whether requests
are printed at all ([A01](A01-request-tracker-deadlocks-headless-mode.md)) — and it
appears in no user-facing document. `docs/Configuration.md`'s "Global
Configuration" table lists `--proxy`, `--config` and the nonexistent `--debug`,
but not `--interactive`.

### `--version` is hidden

`flags.MarkHidden("version")` ([`internal/config/flags.go:24`](../internal/config/flags.go#L24))
removes `-v`/`--version` from `--help`, and it is not documented either. A user
has no supported way to discover the installed version.

## 2. Why it is a documentation problem worth fixing structurally

The flag surface has no single source of truth. `docs/Configuration.md` is a
hand-maintained table that has drifted from `internal/config/flags.go`, and
because `--help` hides one flag and omits the `generate-certs` subcommand
entirely ([A11](A11-cli-command-structure.md)), a reader cannot cross-check the
docs against the tool. Every future flag change will drift the same way.

## 3. Recommended fix

**Immediate (documentation):**

1. Remove every `--debug` occurrence from `docs/Configuration.md`,
   `docs/Migration-Guide.md` and `docs/Troubleshooting.md`.
2. Remove `debug:` from all YAML examples and from the Global Configuration
   Properties table; remove the "debug mode" phrasing in `docs/Home.md`.
3. Document the diagnostics that do exist: `UNCORS_LOGGING=/path/to/file`, and
   what actually lands in it.
4. Document `--interactive` (and `interactive: false` if it is ever supported in
   YAML — currently it is `yaml:"-"`, i.e. CLI-only, which should also be stated).
5. Un-hide and document `--version`.

**Durable (implementation):**

6. Implement the flags the docs describe — `--log-level`/`--debug` and
   `--quiet` — as part of [A13](A13-logging-and-output-are-three-parallel-systems.md).
   The docs are describing a feature users clearly want; the cheapest way to make
   the docs true is to build it.
7. Make the config decoder reject unknown keys
   (`decoder.KnownFields(true)`) so `debug: false` produces
   `field debug not found in type config.UncorsConfig` instead of silence.
8. Generate the CLI reference table in `docs/Configuration.md` from the flag set
   (a small `go generate` step emitting markdown from `pflag.FlagUsages`), so it
   cannot drift again.

## 4. Why this is better

- Documented commands work, which is the minimum contract of a troubleshooting
  guide.
- `KnownFields(true)` converts an entire class of silent config typos —
  `debug`, `cert-file`, `http-port`, misspelled keys — into immediate, precise
  errors. Given the Migration Guide describes removed keys
  (`cert-file`, `http-port`), users upgrading are *especially* likely to leave
  stale keys behind and get no feedback today.
- A generated flag table removes the maintenance burden that caused the drift.

## 5. Trade-offs and migration considerations

- **`KnownFields(true)` is a breaking change** for any config that currently
  carries stale or extra keys — which, per the Migration Guide, is likely
  widespread. Options: ship it as a warning first (parse into a `map[string]any`,
  diff the key set, warn), then promote to an error in the next minor release.
- **Adding `--debug` as an alias for `--log-level=debug`** keeps the documented
  spelling working, which is friendlier than deleting it from the docs; prefer
  this over pure documentation removal if the logging work in
  [A13](A13-logging-and-output-are-three-parallel-systems.md) is planned.
- Generating the flag table requires the docs and the code to live in one repo —
  they do, but note that `docs/` is published as a GitHub wiki, so the generation
  step must run before the wiki sync.

## 6. Code and document references

| What | Where |
| --- | --- |
| Complete flag definitions | [`internal/config/flags.go:10`](../internal/config/flags.go#L10) |
| Unknown-flag error path | [`internal/config/config.go:31`](../internal/config/config.go#L31) |
| Config struct (no `Debug`) | [`internal/config/config.go:16`](../internal/config/config.go#L16) |
| Decoder ignores unknown keys | [`internal/config/config.go:70`](../internal/config/config.go#L70) |
| `--version` hidden | [`internal/config/flags.go:24`](../internal/config/flags.go#L24) |
| Real logging switch | [`internal/infra/loggings.go:16`](../internal/infra/loggings.go#L16) |
| `--debug` in docs | [`docs/Configuration.md`](../docs/Configuration.md) L93, [`docs/Troubleshooting.md`](../docs/Troubleshooting.md) L159/209/446/496, [`docs/Migration-Guide.md`](../docs/Migration-Guide.md) L326 |
| `debug:` in docs | [`docs/Configuration.md`](../docs/Configuration.md) L105/L144, [`docs/Real-World-Examples.md`](../docs/Real-World-Examples.md) L631, [`docs/Troubleshooting.md`](../docs/Troubleshooting.md) L365, [`docs/Home.md`](../docs/Home.md) L97 |
