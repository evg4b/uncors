# D03 — Documentation uses `~/…` paths throughout, but `~` is never expanded

**Severity:** Medium (documented examples fail validation at startup)
**Area:** Documentation vs implementation — filesystem paths

---

## 1. What is wrong

Home-relative paths appear in every path-taking config field across the
documentation:

| Document | Example |
| --- | --- |
| `docs/Configuration.md` L112 | `statics: { /another-path: ~/another-static-dir }` |
| `docs/Static-File-Serving.md` L16, L60, L86, L104 | `dir: ~/project/assets`, `dir: ~/project/dist`, `dir: ~/my-app/build` |
| `docs/Response-Mocking.md` L163 | `file: ~/mocks/users-response.json` |
| `docs/Migration-Guide.md` L39 | `cert-file: ~/certs/server.crt` |

Nothing in the codebase expands `~`. The only uses of the home directory are
`os.UserHomeDir()` for the CA directory
([`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20)) and for
`generate-certs`' output ([`internal/commands/generate_certs.go:48`](../internal/commands/generate_certs.go#L48)).
Config paths are used verbatim:

- static directories: `afero.NewBasePathFs(c.fs, dir.Dir)`
  ([`internal/di/public_api.go:75`](../internal/di/public_api.go#L75))
- mock files: `h.fs.OpenFile(fileName, …)`
  ([`internal/handler/mock/handler.go:96`](../internal/handler/mock/handler.go#L96))
- script files: `afero.ReadFile(h.fs, h.script.File)`
  ([`internal/handler/script/handler.go:64`](../internal/handler/script/handler.go#L64))
- HAR output: `os.WriteFile(tmp, …)`
  ([`internal/handler/har/writer.go:138`](../internal/handler/har/writer.go#L138))

Since config validation stats these paths at load time
(`ValidateDirectory`/`ValidateFile`,
[`internal/config/validate_primitives.go`](../internal/config/validate_primitives.go)),
a `dir: ~/project/dist` fails immediately with a "directory does not exist"
error — unless a literal directory named `~` happens to exist relative to the
working directory, in which case it silently resolves to the wrong place.

Note the shell hides this in the CLI case: `uncors --config ~/cfg.yaml` works
because the *shell* expands `~` before uncors sees it. It is only paths **inside
the YAML file** that break — which is exactly where the docs use them most.

## 2. Why it is worth fixing in code rather than only in docs

Home-relative paths in config files are a normal expectation for a developer
tool, and the documentation's consistent use of them suggests the author intended
them to work. Two further points make code the better fix:

- **The failure is confusing.** "directory `~/project/dist` does not exist" reads
  like a typo in the path, not like "this tool does not understand `~`".
- **Relative paths have the same ambiguity.** `dir: ./dist` resolves against the
  *process* working directory, not against the config file's directory. A user who
  runs `uncors -c ~/projects/app/.uncors.yaml` from elsewhere gets a different
  result than one who runs it from the project root. The docs never mention this,
  and every example uses either `~/…` (broken) or `./…` (position-dependent).

## 3. Recommended fix

**1. Resolve paths once, at config load, in one place.**

```go
// internal/config
func resolvePath(p string, base string, home string) (string, error) {
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		return filepath.Join(home, strings.TrimPrefix(p, "~")), nil
	case filepath.IsAbs(p):
		return p, nil
	default:
		return filepath.Join(base, p), nil    // base = dir of the config file
	}
}
```

Apply it during normalisation (next to `NormaliseMappings`,
[`internal/config/helpers.go:67`](../internal/config/helpers.go#L67)) to
`StaticDirectory.Dir`, `StaticDirectory.Index`, `Response.File`, `Script.File`
and `HARConfig.File`. After that step every path in the config is absolute, so
handlers, validators and the HAR writer all see the same value.

Resolving relative paths against the **config file's directory** is the behaviour
most tools converge on (ESLint, Prettier, Docker Compose) and is what makes a
config file portable. Where there is no config file (pure `--from`/`--to`), fall
back to the process working directory.

**2. Document the rule explicitly** in `docs/Configuration.md`: absolute paths are
used as-is, `~` expands to the user's home directory, and relative paths resolve
against the directory containing the config file.

**3. Fix the stale example.** `docs/Migration-Guide.md:39` shows `cert-file` /
`key-file`, which were removed in 0.6 — that block is intentional (it is the
"before" example), but it should be clearly labelled as legacy since it is the
only place those keys appear.

## 4. Why the recommended approach is better

- **Documented configurations start working**, without rewriting a dozen examples
  into absolute paths that nobody can copy.
- **Configs become portable** between machines and between working directories,
  which matters because `.uncors.yaml` is intended to be committed to a project
  repo (the project's own [`.uncors.yaml`](../.uncors.yaml) is committed).
- **One resolution point** means validation, the static middleware, the mock
  handler, the script handler and the HAR writer cannot disagree about what a path
  means — today each resolves independently against whatever `afero.Fs` it was
  given.

## 5. Trade-offs and migration considerations

- **Changing relative-path resolution from cwd to config-dir is a breaking
  change** for anyone who currently runs uncors from a specific directory and
  relies on that. It is the right behaviour, but it needs a Migration Guide entry;
  alternatively ship `~` expansion first (purely additive — no existing config can
  contain a working literal `~` path) and change relative resolution in the next
  minor release.
- **`os.UserHomeDir()` can fail** (notably in the `scratch` Docker image —
  [A18](A18-listen-address-is-hardcoded-to-loopback.md)). Expansion should return a
  clear error naming the path rather than panicking or silently leaving `~` in
  place.
- **`afero.NewBasePathFs` interacts with absolute paths**: the static middleware
  currently maps the request path onto a base-path FS rooted at `dir.Dir`
  ([`internal/di/public_api.go:75`](../internal/di/public_api.go#L75)); resolving
  `Dir` to an absolute path before that call is compatible, but the tests in
  [`internal/handler/router/router_test.go`](../internal/handler/router/router_test.go)
  that build a `MemMapFs` with absolute keys should be re-checked.
- **Windows**: `~` has no shell meaning there, so expansion in-process is a
  *gain* for Windows users, but `filepath.Join` must be used (not string
  concatenation) so separators stay correct.

## 6. Code and document references

| What | Where |
| --- | --- |
| Home dir used only for the CA | [`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20), [`internal/commands/generate_certs.go:48`](../internal/commands/generate_certs.go#L48) |
| Static dir used verbatim | [`internal/di/public_api.go:75`](../internal/di/public_api.go#L75) |
| Mock file used verbatim | [`internal/handler/mock/handler.go:96`](../internal/handler/mock/handler.go#L96) |
| Script file used verbatim | [`internal/handler/script/handler.go:64`](../internal/handler/script/handler.go#L64) |
| HAR file used verbatim | [`internal/handler/har/writer.go:138`](../internal/handler/har/writer.go#L138) |
| Path validation at load | [`internal/config/validate_primitives.go`](../internal/config/validate_primitives.go), [`internal/config/static.go:74`](../internal/config/static.go#L74) |
| Normalisation hook point | [`internal/config/helpers.go:67`](../internal/config/helpers.go#L67) |
| `~` in docs | [`docs/Static-File-Serving.md`](../docs/Static-File-Serving.md), [`docs/Response-Mocking.md`](../docs/Response-Mocking.md) L163, [`docs/Configuration.md`](../docs/Configuration.md) L112 |
