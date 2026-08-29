# D09 — The documented Docker command cannot work: wrong port mapping, loopback bind, TUI without a TTY, and no `$HOME`

**Severity:** High (a shipped, documented distribution channel is non-functional)
**Area:** Documentation vs implementation — packaging

---

## 1. What is wrong

Both `README.md` and `docs/Installation.md:52` give this command:

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

Four independent reasons it does not work.

### (a) The port mapping does not match the port uncors listens on

`--from http://local.github.com` has no port, so `normalizeHost` leaves it empty
and `GroupByPort` defaults an `http` mapping to **port 80**
([`internal/config/mappings.go:63`](../internal/config/mappings.go#L63),
[`internal/config/default.go:8`](../internal/config/default.go#L8)). The command publishes
container port **3000**, which nothing is listening on. Host port 80 forwards to a
closed port.

### (b) uncors binds `127.0.0.1`, so published ports can never reach it

```go
// internal/di/public_api.go:186
Address: net.JoinHostPort("127.0.0.1", strconv.Itoa(group.Port)),
```

Docker's port publishing forwards to the container's network interface, not to its
loopback. Even with the port numbers corrected, the process is unreachable from
outside the container. There is no flag or config key to change the bind address
— [A18](A18-listen-address-is-hardcoded-to-loopback.md).

### (c) The image starts the interactive TUI against a non-TTY

`--interactive` defaults to `true` ([`internal/config/flags.go:20`](../internal/config/flags.go#L20)),
and the `Dockerfile`'s `ENTRYPOINT ["/bin/uncors"]` passes no override. `docker
run` without `-it` gives the process no TTY, so a full-screen BubbleTea program
with `AltScreen = true` ([`internal/uncors_app/app.go` `View`](../internal/uncors_app/app.go))
starts against a pipe. Nothing in the image sets `--interactive=false`.

### (d) HTTPS mappings cannot work: there is no home directory

The image is `FROM scratch` with `USER 65532:65532` and no `/etc/passwd`, no
`$HOME`. `GetCAPath` calls `os.UserHomeDir()`
([`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20)), which fails
without `$HOME`, so:

- `uncors generate-certs` fails with "failed to get user home directory"
  ([`internal/commands/generate_certs.go:48`](../internal/commands/generate_certs.go#L48))
- `CAExists` returns `false`, so every `https://` mapping fails validation with a
  `TLSError` ([`internal/config/mapping.go:88`](../internal/config/mapping.go#L88))

The `Dockerfile` nonetheless declares `EXPOSE 443`.

### (e) The example itself is odd

`--from http://local.github.com` maps a hostname that must resolve to the
container — inside a container, `local.github.com` resolves to nothing unless the
user adds `--add-host`. The docs do not mention this, while the hosts-file
requirement is prominently documented for native installs
(`docs/Home.md`, `docs/Installation.md`).

## 2. Why it matters

The Docker image is published on Docker Hub, badged in `README.md` with a pull
count, exposed in the `Dockerfile`, built by `.goreleaser.yaml`, and documented in
two places. It is presented as a first-class installation method. A user following
the documented command gets a container that either exits or hangs, with no
diagnostic (logging is discarded by default —
[A13](A13-logging-and-output-are-three-parallel-systems.md)).

This is a packaging/architecture mismatch rather than a typo: three of the four
causes are code-level constraints, so the documentation cannot be fixed on its
own.

## 3. Recommended fix

**Code / image changes (required):**

1. **Add a `--listen` flag / `listen:` config key** defaulting to `127.0.0.1`
   ([A18](A18-listen-address-is-hardcoded-to-loopback.md)).
2. **In the `Dockerfile`, set container-appropriate defaults** so the published
   image works out of the box:

```dockerfile
FROM scratch
USER 65532:65532
COPY uncors /bin/uncors
ENV HOME=/home/nonroot
ENV UNCORS_LISTEN=0.0.0.0
ENV UNCORS_INTERACTIVE=false
EXPOSE 80 443
ENTRYPOINT ["/bin/uncors"]
```

   (This requires reading `UNCORS_*` environment variables as flag defaults — a
   small addition to the flag layer and a conventional one for containerised CLIs.
   Alternatively bake the flags into the entrypoint, at the cost of making them
   non-overridable.)

3. **Create the home directory in the image** (`COPY --chown` an empty dir, or
   switch from `scratch` to `gcr.io/distroless/static:nonroot`, which provides
   `/home/nonroot` and `/etc/passwd`), so `generate-certs` and HTTPS mappings work.
4. **Make the CA directory configurable** (`--ca-dir`, defaulting to
   `$XDG_CONFIG_HOME/uncors` then `~/.config/uncors`) so container users can mount
   a volume for it. This also fixes `generate-certs` on systems where
   `os.UserHomeDir()` is unusable.
5. **Fall back to the plain presenter when stdout is not a TTY**, regardless of
   `--interactive` ([A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md)) —
   a defence in depth that fixes this for every containerised or piped invocation,
   not just the official image.

**Documentation changes:**

6. Correct the run command so the ports agree and the hostname resolves, e.g.:

```bash
docker run --rm -p 3000:3000 --add-host local.github.com:127.0.0.1 \
  evg4b/uncors --from 'http://local.github.com:3000' --to 'https://github.com'
```

7. Add a short "Running in Docker" section covering: mounting a config file
   (`-v $PWD/.uncors.yaml:/config.yaml -c /config.yaml`), mounting the CA
   directory for HTTPS, and the fact that hostnames must resolve **inside** the
   container.

**Add a smoke test.** A CI job that builds the image, runs it, and curls a mapped
endpoint would have caught all of this; it is the only durable guard for a
packaging path that no unit test covers.

## 4. Why this is better

- A documented, badged installation method starts working.
- The `--listen` and `--ca-dir` flags are independently useful (device testing,
  CI, VMs) and are what makes the container fixable rather than special-cased.
- The TTY fallback fixes the whole class — `uncors | tee`, `npx uncors` in a CI
  script, systemd units — not just Docker.
- A container smoke test in CI keeps it working.

## 5. Trade-offs and migration considerations

- **`ENV UNCORS_LISTEN=0.0.0.0` in the image exposes the proxy to the container
  network by default.** That is correct for a container (the isolation boundary is
  the container, not loopback), but the startup warning from
  [A18](A18-listen-address-is-hardcoded-to-loopback.md) should still be printed, and
  the docs must repeat the "never expose uncors to an untrusted network" caution
  in the Docker section.
- **Reading `UNCORS_*` env vars as flag defaults is a new configuration
  precedence layer** (env < config file < CLI flags, or env < flags?). Pick and
  document the order; the conventional choice is CLI > env > config file >
  defaults.
- **Switching the base image from `scratch` to `distroless/static:nonroot`** adds
  a few hundred KB and a base-image dependency, in exchange for a working
  `$HOME`, `/etc/passwd`, CA certificates for upstream TLS verification (which
  `scratch` also lacks — worth checking whether HTTPS *upstreams* currently work
  in this image at all) and `/tmp`. This is very likely a net win and should be
  evaluated on its own.
- The `.goreleaser.yaml` docker build definition must be updated in step with the
  `Dockerfile`.

## 6. Code and document references

| What | Where |
| --- | --- |
| Documented command | [`README.md`](../README.md), [`docs/Installation.md`](../docs/Installation.md) L52 |
| Image definition | [`Dockerfile`](../Dockerfile) |
| Loopback bind | [`internal/di/public_api.go:186`](../internal/di/public_api.go#L186) |
| Default port for schemeless `from` | [`internal/config/mappings.go:63`](../internal/config/mappings.go#L63), [`internal/config/default.go:8`](../internal/config/default.go#L8) |
| Interactive default | [`internal/config/flags.go:20`](../internal/config/flags.go#L20) |
| Alt-screen TUI | [`internal/uncors_app/app.go`](../internal/uncors_app/app.go) (`View`) |
| CA path requires `$HOME` | [`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20) |
| `generate-certs` requires `$HOME` | [`internal/commands/generate_certs.go:48`](../internal/commands/generate_certs.go#L48) |
| HTTPS mapping validation needs the CA | [`internal/config/mapping.go:88`](../internal/config/mapping.go#L88) |
| Release/image build | [`.goreleaser.yaml`](../.goreleaser.yaml) |
