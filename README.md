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
    <img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/evg4b/uncors">
  </a>
  <a href="https://github.com/evg4b/uncors/releases/latest">
    <img alt="GitHub version" src="https://img.shields.io/github/v/tag/evg4b/uncors?label=version">
  </a>
    <a href="https://github.com/evg4b/uncors/blob/main/LICENSE">
        <img alt="License" src="https://img.shields.io/github/license/evg4b/uncors?label=license">
    </a>
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
- [Wildcard host mapping](https://github.com/evg4b/uncors/wiki/2.-Configuration#wilcard-mapping)
- [HTTPS support](https://github.com/evg4b/uncors/wiki/2.-Configuration#https-configuration)
- [Response mocking](https://github.com/evg4b/uncors/wiki/3.-Response-mocksing)
- [HTTP/HTTPS proxy support](https://github.com/evg4b/uncors/wiki/2.-Configuration#proxy-configuration)
- [Static file serving](https://github.com/evg4b/uncors/wiki/4.-Static-file-serving)
- *Response caching ([coming soon...](./ROADMAP.md))*

Other new features you can find in [roadmap](https://github.com/evg4b/uncors/blob/main/ROADMAP.md).

Full documentation you can found on [wiki pages](https://github.com/evg4b/uncors/wiki).

# Quick Install

## Homebrew (macOS | Linux)

If you are on macOS or Linux and using [Homebrew](https://brew.sh/), you can install uncors with the following
one-liner:

```bash 
brew install evg4b/tap/uncors
```

## Scoop (Windows)

If you are on Windows and using [Scoop](https://scoop.sh/), you can install uncors with the following commands:

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

## NPM (Cross-platform)

To install uncors as a node package in your project, you can use the following commands:

Via npm:

```bash
npm install uncors --save-dev
```

Via yarn:

```bash
yarn add uncors --dev
```

## Docker

We currently offer images for [Docker](https://hub.docker.com/r/evg4b/uncors)

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

## Stew (Cross-platform)

Also, you can install binaries using [Stew](https://github.com/marwanhawari/) with the following commands:

```bash
stew install evg4b/uncors
```

## Binary (Cross-platform)

Download the appropriate version for your platform
from [UNCORS releases page](https://github.com/evg4b/uncors/releases/latest).
Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location.
This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your `PATH` for easy use. `/usr/local/bin` is the most probable location.

## Build from source

**Prerequisite Tools**

- Git
- Go (at least Go 1.11)

**Fetch from GitHub**

UNCORS uses the Go Modules support built into Go 1.11 to build. The easiest way to get started is to clone
UNCORS source code in a directory outside the GOPATH, as in the following example:

```
mkdir $HOME/src
cd $HOME/src
git clone https://github.com/evg4b/uncors.git
cd uncors
go install -tags release
```

If you are a Windows user, substitute the $HOME environment variable above with `%USERPROFILE%`.

# Usage

The following command can be used to start the UNCORS proxy server:

```
uncors --from 'http://localhost' --to 'https://github.com' --http-port 8080
```

More information about configuration and usage you can find on [UNCORS wiki](https://github.com/evg4b/uncors/wiki).

# ⚠️ Caution 

Please note that removing or replacing CORS headers can pose potential security vulnerabilities. This tool is specifically designed to streamline the development and testing workflow and should not be used in a production environment or as a remote proxy server. It has not undergone a thorough security review, so caution should be exercised when utilizing it.

## Stargazers over time

[![Stargazers over time](https://starchart.cc/evg4b/uncors.svg)](https://starchart.cc/evg4b/uncors)

