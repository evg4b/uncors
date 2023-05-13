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
    <a href="https://sonarcloud.io/summary/new_code?id=evg4b_uncors&branch=develop">
        <img alt="Coverage" src="https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=coverage&branch=develop">
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
- HTTPS support
- Wildcard URL request mapping
- Simple request/response mocking
- HTTP/HTTPS proxy support
- Static file serving
- *Response caching ([coming soon...](./ROADMAP.md))*

Other new features you can find in [UNCORS roadmap](https://github.com/evg4b/uncors/blob/main/ROADMAP.md)

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

## Binary (Cross-platform)

Download the appropriate version for your platform
from [UNCORS releases page](https://github.com/evg4b/uncors/releases/latest).
Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location.
This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your `PATH` for easy use. `/usr/local/bin` is the most probable location.

## Docker

We currently offer images for [Docker](https://hub.docker.com/r/evg4b/uncors)

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

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

The following command can be used to start the Uncors proxy server:

```
uncors --http-port 8080 --to 'https://github.com' --from 'http://localhost'
```

## CLI Parameters

The following command-line parameters can be used to configure the Uncors proxy server:

* `--from` - Specifies the local host with protocol for the resource from which proxying will take place.
* `--to` - Specifies the target host with protocol for the resource to be proxied.
* `--http-port` or `-p` - Specifies the local HTTP listening port.
* `--https-port` or `-s` - Specifies the local HTTPS listening port.
* `--cert-file` - Specifies the path to the HTTPS certificate file.
* `--key-file` - Specifies the path to the matching certificate private key.
* `--proxy` - Specifies the HTTP/HTTPS proxy to provide requests to the real server (system default is used by default).
* `--config` - Specifies the path to the [configuration file](#configuration-file).
* `--debug` - Enables debug output.

Any configuration parameters passed via CLI (except for `--from` and `--to`) will override the corresponding
parameters specified in the configuration file. The `--from` and `--to` parameters will add an additional mapping
to the configuration.

## Configuration file

Uncors supports a YAML file configuration with the following options:

```yaml
# Base configuration
http-port: 8080 # Local HTTP listened port.
mappings:
  - http://localhost: https://githib.com
  - from: http://other.domain.com
    to: https//example.com
    statics:
      /path: ./public
      /another-path: ~/another-static-dir
debug: false # Show debug output.
proxy: localhost:8080

# HTTPS configuration
https-port: 8081 # Local HTTPS listened port.
cert-file: ~/server.crt # Path to HTTPS certificate file.
key-file: ~/server.key # Path to matching for certificate private key.

# Mock definitions are used to generate fake responses for certain endpoints.
mocks:
  - path: /hello-word
    response:
      code: 200
      raw-content: 'Hello word'
```

#### Mocks configuration

The mocks configuration section in Uncors allows you to define specific endpoints to be mocked, including the response
data and other parameters. Currently, mocks are defined globally for all mappings. Available path, method, queries, and headers filters,
which utilize the [gorilla/mux route matching system](https://github.com/gorilla/mux#matching-routes).

Each endpoint mock requires a path parameter, which defines the URL path for the endpoint. You can also use the method
parameter to define a specific HTTP method for the endpoint.

The queries and headers parameters can be used to specify more detailed URLs that will be mocked. The queries parameter
allows you to define specific query parameters for the URL, while the headers parameter allows you to define specific
HTTP headers.

Here is the structure of the mock configuration:

```yaml
mocks:
  - path: <string>
    method: <string>
    queries:
      <string>: <string>
      # ...
    headers:
      <string>: <string>
      # ...
    response:
      code: <int>
      headers:
        <string>: <string>
        # ...
      delay: <string>
      raw-content: <string>
      file: <string>
```

- `path` (required) - This property is used to define the URL path that should be mocked. The value should be a string,
  such as `/example`. The path can include variables, such as `/users/{id}`, which will match any URL that starts
  with `/users/` and has a variable `id` in it.
- `method` (optional) - This property is used to define the HTTP method that should be mocked.
  The value should be a string. If this property is not specified, the mock will match any HTTP method.
- `queries` (optional) - This property is used to define specific query parameters that should be matched against the
  request URL. The value should be a mapping of query parameters and their values, such as `{"param1": "value1", "
  param2": "value2"}`. If this property is not specified, the mock will match any query parameter.
- `headers` (optional): This property is used to define specific HTTP headers that should be matched against the request
  headers. The value should be a mapping of header names and their values, such as `{"Content-Type": "application/json"}`.
  If this property is not specified, the mock will match any HTTP header.
- `response` (required): This property is used to define the mock response. It should be a mapping that contains the
  following properties:
  - `code` (optional): This property is used to define the HTTP status code that should be returned in the mock
    response. The value should be an integer, such as 200 or 404. If this property is not specified, the mock will use
    200 OK status code.
  - `headers` (optional): This property is used to define specific HTTP headers that should be returned in the mock
    response. The value should be a mapping of header names and their values, such as `{"Content-Type": "
    application/json"}`. If this property is not specified, the mock response will have no extra headers.
  - `delay` (optional): This property is used to define a delay before sending the mock response. The value should be a
    string in the format `<number><unit> <nunmber><units> ...`, where `<number>` is a positive integer and `<unit>` is time units.
    Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. For example, `1m 30s` would delay the response by 1 minute and 30
    seconds. If this property is not specified, the mock response will be sent immediately.
  - `raw-content` (optional): This property is used to define the raw content that should be returned in the mock
    response. The value should be a string, such as `Hello, world!`. If this property is not specified, the mock
    response will be empty.
  - `file` (optional): This property is used to define the path to a file that contains the mock response content. The
    file content will be used as the response content. The value should be a string that specifies the file path, such
    as `~/mocks/example.json`. If this property is not specified, the mock response will be empty.

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
      Server->>Uncors: Real response
      Note over Uncors: Replacing url with source<br/> in headers and cookies
      Uncors-->>Client: Data response
    end
```
