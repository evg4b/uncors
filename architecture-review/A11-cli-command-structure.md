# A11 — Subcommand dispatch is hand-rolled in `main`, split across two packages, with two flag sets

**Severity:** Medium
**Area:** CLI structure

---

## 1. What is wrong with the current approach

Command selection is a string comparison against `os.Args`:

```go
// main.go:29-39
if len(os.Args) >= 2 && os.Args[1] == cli.GenerateCertsCmd {
	container.Override(di.WithArgs(os.Args[2:]))
	err := cli.GenerateCerts(container)
	handleError(err)
	return
}

container.Override(di.WithArgs(os.Args[1:]))
err := cli.RunUncors(context.Background(), container)
```

Around this sit **two** packages that both claim the "commands" role:

- `internal/cli` — `RunUncors`, `GenerateCerts`, `runIneractive`, `runNonIneractive`
- `internal/commands` — `GenerateCertsCommand` with `DefineFlags` / `Execute`

`cli.GenerateCerts` exists only to build a `pflag.FlagSet`, hand it to
`commands.GenerateCertsCommand.DefineFlags`, parse, and call `Execute`
([`internal/cli/generate_certs.go:13`](../internal/cli/generate_certs.go#L13)). The
split adds a package boundary without adding a concept.

Concrete problems that follow:

**(a) Flags are defined twice, in two shapes.** The root flags live in
`internal/config/flags.go` and are parsed inside `LoadConfiguration`
([`internal/config/config.go:26`](../internal/config/config.go#L26)); the subcommand's
flags live in `internal/commands` and bind directly into struct fields. There is
no shared help, no shared usage banner logic (`flags.Usage` with `tui.PrintLogo`
is copy-pasted into both — [`internal/config/flags.go:12`](../internal/config/flags.go#L12)
and [`internal/commands/generate_certs.go:36`](../internal/commands/generate_certs.go#L36)),
and no way to discover that `generate-certs` exists: `uncors --help` lists only the
root flags and never mentions the subcommand.

**(b) `uncors generate-certs --help` and `uncors --help` return success by
swallowing `pflag.ErrHelp` in three different places** —
[`internal/cli/generate_certs.go:19`](../internal/cli/generate_certs.go#L19),
[`internal/cli/run_uncors.go:23`](../internal/cli/run_uncors.go#L23), and again at
[`internal/cli/run_uncors.go:38`](../internal/cli/run_uncors.go#L38). Each is a
slightly different `errors.Is` dance.

**(c) `--version` is implemented as a sentinel error.** `LoadConfiguration`
returns `ErrVersionRequested` ([`internal/config/config.go:13`](../internal/config/config.go#L13))
which the caller pattern-matches to print a string and exit
([`internal/cli/run_uncors.go:16`](../internal/cli/run_uncors.go#L16)). The flag is
then hidden from help ([`internal/config/flags.go:24`](../internal/config/flags.go#L24)),
so `-v`/`--version` is undiscoverable. Using the error channel for a
non-error control-flow outcome is the kind of thing a command framework exists to
avoid.

**(d) Config parsing and CLI parsing are the same function.**
`LoadConfiguration(fs, version, args)` parses flags, reads YAML, merges,
normalises and validates. That is why the interactive reload closure has to
re-pass `container.Args()` on every file change
([`internal/cli/run_ineractive.go:23`](../internal/cli/run_ineractive.go#L23)) — the
flags are re-parsed from scratch every time the config file is saved.

**(e) An empty, committed `cmd/diag/` directory** sits in the repo root
suggesting a third command location that was never built.

**(f) There is no `--listen`/`--host`, no `--quiet`, no `--log-level`, no
`--output`** — the flag surface is five flags, one of which is hidden, and the
mode-defining `--interactive` flag is undocumented ([D01](D01-debug-flag-does-not-exist.md)).

## 2. Why it is an architectural problem

- **Command dispatch, flag definition, and configuration loading are three
  concerns fused into one path.** Adding a second subcommand (`uncors validate`,
  `uncors print-config`, `uncors trust-ca` — all natural for this tool) means
  another `os.Args[1] ==` branch in `main`, another bespoke `FlagSet`, another
  `ErrHelp` special case.
- **Help output is not a coherent surface.** Users cannot discover subcommands or
  `--version` from `--help`, which is the primary discovery mechanism for a CLI.
- **`main` contains policy.** Argument slicing, container mutation, and
  error-to-exit-code translation ([`main.go:43`](../main.go#L43)) live in `package
  main` where nothing is testable except via the `osExit` variable seam that had
  to be introduced for exactly that reason.

## 3. What the recommended approach is instead

**Adopt a single command tree with one flag layer.** Two viable shapes:

**Option 1 — `spf13/cobra`** (natural given `pflag` is already a dependency):

```go
root := &cobra.Command{Use: "uncors", RunE: runProxy}
root.Flags().StringSliceP("from", "f", nil, "...")
root.AddCommand(generateCertsCmd, validateCmd)
root.Version = version          // gives --version + `uncors version` for free
```

Cobra provides: subcommand discovery in `--help`, per-command help, shell
completion, consistent `ErrHelp` handling, and `SilenceUsage`/`SilenceErrors` so
errors are reported once. Cost: one dependency (~1 MB binary growth).

**Option 2 — keep `pflag`, add a tiny dispatcher.** If the dependency is
unwanted:

```go
type Command struct {
	Name  string
	Short string
	Flags func(*pflag.FlagSet)
	Run   func(ctx context.Context, app *app.Container, args []string) error
}
var commands = []Command{proxyCmd, generateCertsCmd}
```

with one `Execute(args)` that resolves the name, builds the flag set, handles
`ErrHelp` once, and prints a command list when no name matches. ~80 lines,
no dependency, and it forces every future command through the same path.

Either way:

- **Separate flag parsing from config loading.** Parse once into a
  `config.Flags` struct; `LoadConfiguration(fs, flags)` then takes already-parsed
  values. Reload re-reads only the YAML file and re-applies the *same* flag
  overrides, instead of re-parsing `os.Args`.
- **Make `--version` a normal command outcome**, not a sentinel error, and unhide
  it.
- **Merge `internal/cli` and `internal/commands`** into one `internal/cli`
  package: one `Command` per file, each with its flags and its `Run`.
- **Delete `cmd/diag/`** or implement it.

## 4. Why the proposed approach is better

- **New commands become additive**, not another `main` branch. `uncors validate
  -c cfg.yaml` (validate and exit non-zero) is an obviously useful command that is
  currently awkward to add and would be trivial afterwards.
- **`--help` becomes the real contract**: subcommands and all flags discoverable
  in one place, which also gives the docs a single source to describe
  ([D01](D01-debug-flag-does-not-exist.md) exists partly because the flag surface
  is not self-describing).
- **`ErrHelp`/`--version` special cases collapse from four sites to one.**
- **Reload stops re-parsing `os.Args`**, which removes a class of surprise (flags
  are re-evaluated on every file save today) and makes the reload path
  side-effect-free.
- `main` becomes three lines, and `osExit` as a test seam is no longer needed.

## 5. Trade-offs and migration considerations

- **Cobra is a real dependency** with its own opinions (it will want a `Use`
  string, it prints its own error formatting unless silenced, and it changes help
  layout — which currently renders the ASCII logo via a custom `flags.Usage`).
  Preserving the logo means setting a custom help template; budget for that.
- **Changing `--version` from hidden to visible is user-facing**, as is any help
  reformatting. Both are improvements but belong in a release note.
- **Splitting flags from config loading touches `LoadConfiguration`'s signature**,
  which is called from three places
  ([`run_uncors.go`](../internal/cli/run_uncors.go), [`run_ineractive.go`](../internal/cli/run_ineractive.go),
  [`run_non_ineractive.go`](../internal/cli/run_non_ineractive.go)) plus tests. Small
  and mechanical.
- Do this **before** [A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md),
  because the presenter selection (`--interactive`, TTY detection, a future
  `--output`) naturally belongs to the command layer.

## 6. Code references

| What | Where |
| --- | --- |
| `os.Args[1]` dispatch | [`main.go:29`](../main.go#L29) |
| Exit-code translation in `main` | [`main.go:43`](../main.go#L43) |
| Subcommand wrapper | [`internal/cli/generate_certs.go:13`](../internal/cli/generate_certs.go#L13) |
| Subcommand implementation | [`internal/commands/generate_certs.go:35`](../internal/commands/generate_certs.go#L35) |
| Root flag definitions | [`internal/config/flags.go:10`](../internal/config/flags.go#L10) |
| Duplicated usage banner | [`internal/config/flags.go:12`](../internal/config/flags.go#L12), [`internal/commands/generate_certs.go:36`](../internal/commands/generate_certs.go#L36) |
| `--version` as sentinel error | [`internal/config/config.go:13`](../internal/config/config.go#L13), [`internal/cli/run_uncors.go:16`](../internal/cli/run_uncors.go#L16) |
| Hidden `--version` | [`internal/config/flags.go:24`](../internal/config/flags.go#L24) |
| Three `ErrHelp` special cases | [`internal/cli/generate_certs.go:19`](../internal/cli/generate_certs.go#L19), [`internal/cli/run_uncors.go:23`](../internal/cli/run_uncors.go#L23), [`:38`](../internal/cli/run_uncors.go#L38) |
| Flags re-parsed on every reload | [`internal/cli/run_ineractive.go:23`](../internal/cli/run_ineractive.go#L23), [`internal/cli/run_non_ineractive.go:91`](../internal/cli/run_non_ineractive.go#L91) |
| Empty command directory | `cmd/diag/` |
