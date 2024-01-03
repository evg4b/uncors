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
    <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/evg4b/uncors/develop?label=go">
  </a>
  <a href="#">
    <img alt="GitHub version" src="https://img.shields.io/badge/version-%40next-blue">
  </a>
  <a href="https://github.com/evg4b/uncors/blob/main/LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/evg4b/uncors?label=license&branch=develop">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Coverage" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=coverage&branch=develop">
  </a>
  <a href="https://goreportcard.com/report/github.com/evg4b/uncors">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/evg4b/uncors">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Reliability Rating" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=reliability_rating&branch=develop">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Security Rating" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=security_rating&branch=develop">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors">
    <img alt="Lines of Code" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=ncloc&branch=develop">
  </a>
</p>

# Core features

- CORS header replacement
- [Wildcard host mapping](https://github.com/evg4b/uncors/wiki/2.-Configuration#wilcard-mapping)
- [HTTPS support](https://github.com/evg4b/uncors/wiki/2.-Configuration#https-configuration)
- [Response mocking](https://github.com/evg4b/uncors/wiki/3.-Response-mocksing)
- [HTTP/HTTPS proxy support](https://github.com/evg4b/uncors/wiki/2.-Configuration#proxy-configuration)
- [Static file serving](https://github.com/evg4b/uncors/wiki/4.-Static-file-serving)
- [Response caching](https://github.com/evg4b/uncors/wiki/5.-Response-caching)

Other new features you can find in [roadmap](https://github.com/evg4b/uncors/blob/main/ROADMAP.md).

Full documentation you can found on [wiki pages](https://github.com/evg4b/uncors/wiki).

# Quick Install

You can install the application in one of the following ways:

#### [Homebrew](https://brew.sh/) (macOS | Linux)

```bash 
brew install evg4b/tap/uncors
```

#### [Scoop](https://scoop.sh/) (Windows)

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

#### [NPM](https://npmjs.com) (Cross-platform)

```bash
npm install uncors --save-dev
# OR
yarn add uncors --dev
# OR
pnpm add -D uncors
```

#### [Docker](https://www.docker.com/) (Cross-platform)

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

#### [Stew](https://github.com/marwanhawari/) (Cross-platform)

```bash
stew install evg4b/uncors
```

Or find more installation methods in [uncors wiki](https://github.com/evg4b/uncors/wiki/1.-Installation).

# Usage

The following command can be used to start the UNCORS proxy server:

```
uncors --from 'http://localhost' --to 'https://github.com' --http-port 8080
```

More information about configuration and usage you can find on [UNCORS wiki](https://github.com/evg4b/uncors/wiki).

# ⚠️ Caution

Please note that removing or replacing CORS headers can pose potential security vulnerabilities. This tool is
specifically designed to streamline the development and testing workflow and should not be used in a production
environment or as a remote proxy server. It has not undergone a thorough security review, so caution should be exercised
when utilizing it.

# Stargazers over time

[![Stargazers over time](https://starchart.cc/evg4b/uncors.svg?variant=adaptive)](https://starchart.cc/evg4b/uncors)
