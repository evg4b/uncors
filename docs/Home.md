<!--suppress HtmlDeprecatedAttribute -->
<p align="center">
  <a href="https://github.com/evg4b/uncors" title="uncors">
    <img alt="UNCORS logo" width="60%" src="https://raw.githubusercontent.com/evg4b/uncors/main/.github/logo.png">
  </a>
  <br />
  <span>Version: 0.5.0</span>
</p>

## Introduction

UNCORS is a powerful development tool designed to simplify HTTP/HTTPS proxying and CORS header management during local development. It provides a comprehensive suite of features including HTTPS support, wildcard host mapping, request/response mocking, static file serving, response caching, and full proxy functionality. UNCORS streamlines your development workflow by eliminating common CORS-related obstacles without requiring backend modifications.

## Documentation

- [Installation](./1.-Installation)
- [Configuration](./2.-Configuration)
- [Response mocking](./3.-Response-mocking)
- [Static file serving](./4.-Static-file-serving)
- [Response caching](./5.-Response-caching)
- [Request rewriting](./6.-Request-rewriting)
- [Request flow](./7.-Request-flow)
- [Migration guide](./8.-Migration-Guide)

## List of core features

- CORS header replacement
- [HTTPS support](./2.-Configuration#https-configuration)
- [Wildcard host mapping](./2.-Configuration#wildcard-mapping)
- [HTTP/HTTPS proxy support](./2.-Configuration#proxy-configuration)
- [Response mocking](./3.-Response-mocking)
- [Static file serving](./4.-Static-file-serving)
- [Response caching](./5.-Response-caching)
- [Request rewriting](./6.-Request-rewriting)

## Overview

UNCORS enables developers to make browser requests to APIs that would typically be blocked by CORS (Cross-Origin Resource Sharing) policies. This tool is particularly valuable during application development and testing phases, as it eliminates the need to run backend services locally or modify server configurations. Key capabilities include support for local domain mapping and flexible wildcard-based domain matching.

> [!CAUTION]
> Please be aware that the modification or replacement of CORS headers may introduce potential security vulnerabilities.
> This tool is specifically engineered to optimize the development and testing workflow and is not intended for use in a
> production environment or as a remote proxy server. It has not undergone a thorough security review; therefore, caution
> should be exercised when utilizing it.
