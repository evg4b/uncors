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
    <a href="https://github.com/evg4b/uncors/releases">
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

# Quick Install 

## Homebrew (macOS | Linux)

If you are on macOS or Linux and using [Homebrew](https://brew.sh/), you can install uncors with the following one-liner:

```bash 
brew install evg4b/tap/uncors
```

## Scoop (Windows) 

If you are on Windows and using [Scoop](https://scoop.sh/), you can install uncors with the following commands:

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

## Binary (Cross-platform) 

Download the appropriate version for your platform from [UNCORS releases page](https://github.com/evg4b/uncors/releases).
Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location.
This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your `PATH` for easy use. `/usr/local/bin` is the most probable location.

## Docker 

We currently offer images for Docker https://hub.docker.com/r/evg4b/uncors

```bash
docker run -p 3000:3000 evg4b/uncors --source http://local.github.com --target //github.com
```

## Source 

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
go install
```

If you are a Windows user, substitute the $HOME environment variable above with `%USERPROFILE%`.

# Usage
```
./uncors --port 8080 --target 'https://github.com' --source 'http://localhost'
```

## Parameters

* `--source` - Local host with protocol for to the resource from which proxying will take place.
* `--target` - Target host with protocol for to the resource to be proxy.
* `--http-port` - Local HTTP listened port.
* `--https-port` - Local HTTPS listened port. 
* `--cert-file` - Path to HTTPS certificate file.
* `--key-file` - Path to matching for certificate private key.
* `--proxy` - HTTP/HTTPS proxy to provide requests to real server (used system by default).
* `--mocks` - File with defined [mocks](#mocks)

## Mocks

Uncors has endpoint mocks mechanism.
All mocks should be defined in yaml file and passed as parameter `--mocks`.
Currently available path, method, queries and headers filters
(for more information see [gorilla/mux](https://github.com/gorilla/mux#matching-routes) route matching) . 

**Mocks file example:**

```yaml
- path: /hello
  response:
    code: 200
    raw-content: '
      Hello word
    '
- path: /word
  method: POST
  queries:
    param1: param 1 value
    param2: param 1 value
  headers:
    header1: header 1 value
    header2: header 2 value
  response:
    code: 200
    headers:
      header1: header 1 value
      header2: header 2 value
    raw-content: '
      { "status": "ok" }
    '
```

At the moment supported only raw response content.
Content should be defined as multiline string (see more [here](https://yaml-multiline.info/)). 
`Content-Type` will be set automatically, but you can specify custom content type via headers section.

## How it works 

```mermaid
sequenceDiagram
    participant Client
    participant Uncors
    participant Server


    alt Handling OPTIONS queries 
        Client ->> Uncors: Access-Control-Request
        Uncors ->> Client: Allow-Control-Request
    end
    
    alt Handling Data queries 
      Client ->> Uncors: GET, POST, PUT... query
      Note over Uncors: Replacing url with target<br/> in headers and cookies
      Uncors-->>Server: Real GET, POST, PUT... query
      Server->>Uncors: Real responce
      Note over Uncors: Replacing url with source<br/> in headers and cookies
      Uncors-->>Client: Data responce
    end
```
