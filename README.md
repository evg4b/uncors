<p align="center">
  <a href="https://github.com/evg4b/uncors" title="uncors">
    <img width="80%" src="https://raw.githubusercontent.com/evg4b/uncors/main/.github/logo.png">
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

## Usage
```
./uncors --port 8080 --target 'https://github.com' --source 'http://localhost'
```

## Parameters

* `--port` - Local HTTP linthing port. 
* `--target` - Url with protocol for to the resource to be proxyed.
* `--source` - Url with protocol for to the resource from which proxying will take place.