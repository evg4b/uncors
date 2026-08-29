# A18 — The listen address is hard-coded to `127.0.0.1`, which makes Docker and LAN/device testing impossible

**Severity:** Medium
**Area:** Server / configuration surface

---

## 1. What is wrong with the current approach

The bind address is a string literal inside the DI container:

```go
// internal/di/public_api.go:184-188
targets = append(targets, server.Target{
	Address:   net.JoinHostPort("127.0.0.1", strconv.Itoa(group.Port)),
	Handler:   muxRouter,
	EnableTLS: group.Scheme == "https",
})
```

There is no flag, no config key, and no environment variable to change it. Two
direct consequences:

**(a) The published Docker image cannot work.** `Dockerfile` exposes 80 and 443
and the README/Installation docs instruct:

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

A process bound to `127.0.0.1` inside a container is unreachable from the host's
published port, which requires binding `0.0.0.0` (or at least the container's
external interface). The command is additionally inconsistent — `--from
http://local.github.com` has no port, so uncors listens on **80**, while
`-p 80:3000` publishes container port **3000**. See
[D09](D09-docker-instructions-cannot-work.md).

**(b) Testing from another device is impossible.** A very common dev-proxy use
case — running uncors on a laptop and pointing a phone, a VM, or a container at
it — cannot be done at all.

Related gaps in the same area:

- **`Server.Target` has no concept of an interface**, only `Address`
  ([`internal/server/server.go:23`](../internal/server/server.go#L23)), so the
  limitation is baked into the type that the whole server layer uses.
- **`PortListener.Listen` uses `"tcp"`** ([`internal/server/port_listener.go:22`](../internal/server/port_listener.go#L22))
  while the test helper occupying a port in
  [`internal/cli/run_uncors_test.go`](../internal/cli/run_uncors_test.go) uses `"tcp4"` —
  a small inconsistency that matters on dual-stack hosts.
- **The TLS certificate manager falls back to the connection's local address when
  there is no SNI** and explicitly rejects `0.0.0.0` / `::`
  ([`internal/server/host_cert_manager.go:131`](../internal/server/host_cert_manager.go#L131)),
  so binding a wildcard address would need that path to be reconsidered (SNI-less
  clients would get `ErrNoSNIProvided`). That is a solvable problem but shows the
  loopback assumption has propagated.
- **`GetCAPath` requires `os.UserHomeDir()`** ([`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20)),
  which fails in the `scratch`-based image (`USER 65532`, no `/etc/passwd`, no
  `$HOME`), so HTTPS mappings cannot work in Docker even if the bind address is
  fixed.

## 2. Why it is an architectural problem

- **A deployment-shaped decision is embedded in a construction helper.** The
  address a server binds to is configuration; putting it in `di.Targets` means
  neither the server layer nor the config layer owns it, and nothing can override
  it.
- **It silently invalidates a shipped distribution channel.** The project builds,
  publishes and documents a Docker image whose primary invocation cannot function.
  That is a design/packaging mismatch, not a typo.
- **Loopback-only is a reasonable *default*, not a reasonable *constraint*.** The
  security argument for it is real (uncors disables CORS and must not be exposed),
  but the correct expression of that is a safe default plus an explicit,
  warned-about opt-in.

## 3. What the recommended approach is instead

**Make the bind address configurable, defaulting to loopback.**

1. Add a global config key and flag:

```yaml
listen: 127.0.0.1        # default; use 0.0.0.0 to expose on the network
```
```
--listen ADDRESS         (default: 127.0.0.1)
```

2. Optionally allow it per mapping (`from` already carries a hostname; a
   `listen:` override per mapping covers multi-interface setups). Start global —
   it covers every reported use case.

3. **Warn loudly on a non-loopback bind:**

```
WARNING  uncors is listening on 0.0.0.0:3000 and disables CORS protections.
         Anyone on your network can use this proxy. Do not run this on an
         untrusted network.
```

4. **Move address construction into the server layer.** `server.Target` gains the
   full address from a `TargetsFor(cfg)` function that lives with the server (see
   [A04](A04-service-locator-di-container.md)), not in the DI container.

5. **Handle the SNI-less TLS case for wildcard binds.** When the listener is
   wildcard-bound and a client sends no SNI, fall back to a certificate for the
   first configured HTTPS hostname of that port group rather than erroring —
   or return a clear error message that names the cause.

6. **Fix the container image**:
   - Ship a default that binds `0.0.0.0` *inside the image only* (e.g. an
     `ENV UNCORS_LISTEN=0.0.0.0` in the `Dockerfile`, or document
     `--listen 0.0.0.0` in the run command).
   - Set `--interactive=false` by default in the image (no TTY —
     [A10](A10-interactive-and-headless-modes-are-two-parallel-implementations.md)).
   - Set `ENV HOME=/home/nonroot` and create that directory, or make the CA path
     configurable, so `generate-certs` and HTTPS mappings work.
   - Correct the documented `-p` mapping so the published port matches the port in
     `--from`.

## 4. Why the proposed approach is better

- **Docker, VMs, containers, and mobile-device testing become possible**, which
  unblocks a documented installation method and a common workflow.
- **The safe default is preserved** — nothing changes for existing users — while
  the escape hatch is explicit and accompanied by a warning, which is a better
  security posture than "impossible, therefore safe" (users currently work around
  it with `socat`/`ssh -L`, which is strictly worse).
- **Address policy moves next to the listener**, so the server layer becomes
  self-contained and the DI container stops carrying deployment decisions.
- **The IPv4/IPv6 story becomes explicit** rather than depending on what `"tcp"`
  resolves to for the literal `127.0.0.1`.

## 5. Trade-offs and migration considerations

- **Exposing the proxy is genuinely dangerous** — it strips CORS and can hold a
  trusted local CA. The warning text matters, and it may be worth requiring an
  extra confirmation flag (`--listen 0.0.0.0 --i-know-this-is-unsafe`) or refusing
  non-loopback binds unless a config file (not just a CLI flag) requests it. Pick
  one deliberately.
- **Wildcard binds interact with the on-the-fly certificate manager**; do not ship
  the bind change for HTTPS mappings until the SNI fallback is handled, or HTTPS
  users will get confusing handshake failures.
- **The Docker image changes are user-visible** and should land together with the
  documentation fix, otherwise the docs will describe a third, also-wrong
  invocation.
- The core change itself is small: one config field, one flag, one line in
  `Targets`, plus the warning.

## 6. Code references

| What | Where |
| --- | --- |
| Hard-coded loopback | [`internal/di/public_api.go:186`](../internal/di/public_api.go#L186) |
| `Target` type | [`internal/server/server.go:23`](../internal/server/server.go#L23) |
| Listener creation | [`internal/server/port_listener.go:22`](../internal/server/port_listener.go#L22) |
| SNI fallback rejects wildcard addresses | [`internal/server/host_cert_manager.go:131`](../internal/server/host_cert_manager.go#L131) |
| CA path requires `$HOME` | [`internal/server/ca_path.go:20`](../internal/server/ca_path.go#L20) |
| Container image | [`Dockerfile`](../Dockerfile) |
| Documented run command | [`README.md`](../README.md), [`docs/Installation.md`](../docs/Installation.md) |
| Flag definitions (no `--listen`) | [`internal/config/flags.go:10`](../internal/config/flags.go#L10) |
