<!--suppress HtmlDeprecatedAttribute -->

<p align="center">
  <a href="https://github.com/evg4b/uncors" title="uncors">
    <img alt="UNCORS logo" width="60%" src="https://raw.githubusercontent.com/evg4b/uncors/main/.github/logo.png">
  </a>
  <br />
  <span>Version: 0.6.1</span>
</p>

UNCORS is a local HTTP/HTTPS proxy for development. You point a local domain
at it, it forwards requests to a remote server, and it replaces the CORS headers
in every response so the browser accepts them. You don't have to change the
backend.

On top of plain proxying, each mapping can mock endpoints, serve local files,
run Lua scripts, cache responses, rewrite paths, and record traffic to a HAR
file.

> [!CAUTION]
> Rewriting CORS headers weakens browser security. UNCORS is meant for local
> development and testing. Do not run it in production or expose it as a remote
> proxy. It has not had a security review.

## Quick start

Install UNCORS:

```bash
# macOS/Linux with Homebrew
brew install evg4b/tap/uncors

# or with npm
npm install -g uncors
```

Point a local domain at your machine. On macOS and Linux:

```bash
echo "127.0.0.1 api.local" | sudo tee -a /etc/hosts
```

On Windows, open `C:\Windows\System32\drivers\etc\hosts` as Administrator and
add the line `127.0.0.1 api.local`.

Create `.uncors.yaml` in your project directory:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.github.com
```

Start UNCORS and send a request:

```bash
uncors --config .uncors.yaml
curl http://api.local:3000/
```

The response comes from `https://api.github.com`, with CORS headers added by
UNCORS.

From here, [Configuration](Configuration) covers every option,
[Response Mocking](Response-Mocking) shows how to add fake endpoints, and
[Static File Serving](Static-File-Serving) shows how to serve a local build.

## Terms used in these docs

| Term           | Meaning                                                                                                   |
| -------------- | --------------------------------------------------------------------------------------------------------- |
| Mapping        | One entry under `mappings`: a `from` URL where UNCORS listens and a `to` URL it forwards to.              |
| Source         | The `from` URL, for example `http://api.local:3000`. Its port is the port UNCORS listens on.              |
| Target         | The `to` URL, the upstream server, for example `https://api.example.com`.                                 |
| Global options | Top-level keys that apply to all mappings: `proxy` and `cache-config`.                                    |
| Mock           | A fixed response returned for matching requests without contacting the target.                           |
| Static         | A local directory served under a URL path prefix.                                                         |
| Script         | A Lua script that builds the response for matching requests.                                              |
| Cache          | Stored upstream responses for paths that match a glob pattern.                                            |
| Rewrite        | A rule that changes the request path (and optionally the upstream host) before the request is proxied.    |
| OPTIONS handling | UNCORS answering CORS preflight `OPTIONS` requests itself instead of forwarding them.                   |

## Documentation

Getting started:

- [Installation](Installation): package managers, binaries, Docker, building
  from source, and hosts file setup.
- [Configuration](Configuration): CLI flags, the YAML file, host mappings,
  HTTPS, and upstream proxy settings.

Features:

- [Response Mocking](Response-Mocking): return predefined responses for
  matching requests.
- [Static File Serving](Static-File-Serving): serve local files and single-page
  apps.
- [Response Caching](Response-Caching): cache upstream responses by path glob.
- [Request Rewriting](Request-Rewriting): change paths and upstream hosts
  before proxying.
- [Script Handler](Script-Handler): build responses with Lua.
- [HAR Recording](HAR-Collector): record proxied traffic to a HAR file.

Reference:

- [Real-World Examples](Real-World-Examples): complete configurations for
  common setups.
- [Migration Guide](Migration-Guide): breaking changes between versions.
- [Troubleshooting](Troubleshooting): common problems and how to fix them.
