<!--suppress HtmlDeprecatedAttribute -->
<p align="center">
  <a href="https://github.com/evg4b/uncors" title="uncors">
    <img alt="UNCORS logo" width="80%" src="https://raw.githubusercontent.com/evg4b/uncors/main/.github/logo.png">
  </a>
</p>
<p align="center">
  A simple dev HTTP/HTTPS proxy for replacing CORS headers.
</p>
<p align="center">
  <a href="https://go.dev">
    <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/evg4b/uncors/main?label=go">
  </a>
  <a href="https://github.com/evg4b/uncors/releases/latest">
    <img alt="GitHub Release" src="https://img.shields.io/github/v/release/evg4b/uncors">
  </a>
  <a href="https://github.com/evg4b/uncors/blob/main/LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/evg4b/uncors?label=license&branch=main">
  </a>
  <a href="https://github.com/evg4b/uncors/actions/workflows/go.yml">
    <img alt="GitHub Actions Workflow Status" src="https://img.shields.io/github/actions/workflow/status/evg4b/uncors/go.yml">
  </a>
  <br/>
  <a href="https://www.npmjs.com/package/uncors">
    <img alt="NPM Downloads" src="https://img.shields.io/npm/dm/uncors?logo=npm">
  </a>
  <a href="https://hub.docker.com/r/evg4b/uncors">
    <img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/evg4b/uncors?logo=docker&logoColor=%23fff">
  </a>
  <a href="https://github.com/evg4b/uncors/releases/latest">
    <img alt="GitHub Downloads (all assets, all releases)" src="https://img.shields.io/github/downloads/evg4b/uncors/total?logo=github">
  </a>
  <br/>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Coverage" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=coverage">
  </a>
  <a href="https://goreportcard.com/report/github.com/evg4b/uncors">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/evg4b/uncors">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Reliability Rating" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=reliability_rating">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Security Rating" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=security_rating">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Lines of Code" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=ncloc">
  </a>
</p>

# Core features

- CORS header replacement
- [Host mapping with named placeholders](https://github.com/evg4b/uncors/wiki/Configuration#named-placeholder-mapping)
- [HTTPS support](https://github.com/evg4b/uncors/wiki/Configuration#https-configuration) with auto-generated certificates
- [Response mocking](https://github.com/evg4b/uncors/wiki/Response-Mocking)
- [Script handler](https://github.com/evg4b/uncors/wiki/Script-Handler) (Lua with a JSON module)
- [HTTP/HTTPS proxy support](https://github.com/evg4b/uncors/wiki/Configuration#proxy-configuration)
- [Static file serving](https://github.com/evg4b/uncors/wiki/Static-File-Serving)
- [Response caching](https://github.com/evg4b/uncors/wiki/Response-Caching)
- [Request rewriting](https://github.com/evg4b/uncors/wiki/Request-Rewriting)
- [HAR traffic recording](https://github.com/evg4b/uncors/wiki/HAR-Collector)

The full documentation is on the [wiki](https://github.com/evg4b/uncors/wiki).

# Quick install

#### [Homebrew](https://brew.sh/) (macOS, Linux)

```bash
brew install evg4b/tap/uncors
```

#### [Scoop](https://scoop.sh/) (Windows)

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

#### [npm](https://npmjs.com) (cross-platform)

```bash
# Run without installing
npx -y uncors --from 'http://localhost:8080' --to 'https://github.com'
# Or add it to your project
npm install uncors --save-dev
# yarn add uncors --dev
# pnpm add -D uncors
```

#### [Docker](https://www.docker.com/) (cross-platform)

```bash
docker run -p 80:80 evg4b/uncors --interactive=false --from 'http://local.github.com' --to 'https://github.com'
```

#### [Stew](https://github.com/marwanhawari/stew) (cross-platform)

```bash
stew install evg4b/uncors
```

Other options, including prebuilt binaries and building from source, are
described on the [Installation](https://github.com/evg4b/uncors/wiki/Installation)
wiki page.

# Usage

Start a proxy from `http://localhost:8080` to `https://github.com`:

```bash
uncors --from 'http://localhost:8080' --to 'https://github.com'
```

For larger setups, put the mappings in a YAML file and run
`uncors --config .uncors.yaml`. The
[Configuration](https://github.com/evg4b/uncors/wiki/Configuration) wiki page
describes every option.

> [!Caution]
>
> Rewriting CORS headers weakens browser security. UNCORS is meant for local
> development and testing. Do not run it in production or expose it as a remote
> proxy. It has not had a security review.

# Stargazers over time

[![Stargazers over time](https://starchart.cc/evg4b/uncors.svg?variant=adaptive&line=%232f81f7)](https://starchart.cc/evg4b/uncors)

# Support the project

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/X8X0SWTP3)
