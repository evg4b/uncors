![UNCORS](./.github/logo.png)

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/evg4b/uncors)](https://go.dev)
[![GitHub tag](https://img.shields.io/github/v/tag/evg4b/uncors?label=version)](https://github.com/evg4b/uncors/releases)
[![GitHub](https://img.shields.io/github/license/evg4b/uncors)](https://github.com/evg4b/uncors)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=coverage)](https://sonarcloud.io/summary/new_code?id=evg4b_uncors)
[![Go Report Card](https://goreportcard.com/badge/github.com/evg4b/fisherman)](https://goreportcard.com/report/github.com/evg4b/fisherman)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=evg4b_uncors)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=evg4b_uncors)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=evg4b_uncors&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=evg4b_uncors)

A simple dev HTTP/HTTPS proxy for replacing CORS headers.

## Usage
```
./uncors --port 8080 --target 'https://github.com' --source 'http://localhost'
```

## Parameters

* `--port` - Local HTTP linthing port. 
* `--target` - Url with protocol for to the resource to be proxyed.
* `--source` - Url with protocol for to the resource from which proxying will take place.