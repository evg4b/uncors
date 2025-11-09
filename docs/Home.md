<!--suppress HtmlDeprecatedAttribute -->
<p align="center">
  <a href="https://github.com/evg4b/uncors" title="uncors">
    <img alt="UNCORS logo" width="60%" src="https://raw.githubusercontent.com/evg4b/uncors/main/.github/logo.png">
  </a>
  <br />
  <span>Version: 0.6.0</span>
</p>

## Introduction

UNCORS is a powerful development tool designed to simplify HTTP/HTTPS proxying and CORS header management during local development. It provides a comprehensive suite of features including HTTPS support, wildcard host mapping, request/response mocking, static file serving, response caching, and full proxy functionality. UNCORS streamlines your development workflow by eliminating common CORS-related obstacles without requiring backend modifications.

## Quick Start (5 minutes)

Get UNCORS running in 5 minutes:

**1. Install UNCORS:**

Choose your preferred installation method:

```bash
# macOS/Linux with Homebrew
brew install evg4b/tap/uncors

# or with NPM
npm install -g uncors
```

**2. Configure your hosts file:**

Add a local domain mapping to your system's hosts file:

**macOS/Linux:**

```bash
echo "127.0.0.1 api.local" | sudo tee -a /etc/hosts
```

**Windows (run as Administrator):**

Add this line to `C:\Windows\System32\drivers\etc\hosts`:

```
127.0.0.1 api.local
```

**3. Create a configuration file:**

Create `.uncors.yaml` in your project directory:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.github.com
```

**4. Start UNCORS:**

```bash
uncors --config .uncors.yaml
```

**5. Test it:**

```bash
curl http://api.local:3000/
# You should see GitHub's API response
```

That's it! UNCORS is now proxying requests from `api.local` to GitHub's API.

**Next steps:**

- Read [Configuration](./Configuration) for more options
- Explore [Response Mocking](./Response-Mocking) to add fake endpoints
- Learn about [Static File Serving](./Static-File-Serving) for local development

## Key Terminology

Understanding these terms will help you navigate the documentation more effectively:

| Term                                | Definition                                                                                                                     |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **Host Mapping**                    | A configuration that defines how requests from a source domain are routed to a target domain (defined by `from` and `to` URLs) |
| **Source Domain**                   | The local domain where UNCORS listens for requests (specified in the `from` URL, e.g., `http://api.local:3000`)                |
| **Target Domain** (Upstream Server) | The remote server where requests are proxied (specified in the `to` URL, e.g., `https://api.example.com`)                      |
| **Mapping Configuration**           | Settings specific to individual host mappings, including mocks, statics, scripts, cache, and rewrites                          |
| **Global Configuration**            | Settings that apply to all mappings, such as debug mode, proxy settings, and SSL certificates                                  |
| **Scheme**                          | The protocol prefix of a URL (`http://`, `https://`, or `//` for scheme-agnostic)                                              |
| **Port**                            | The network port number specified in the `from` URL (defaults: 80 for HTTP, 443 for HTTPS)                                     |
| **Mock**                            | A configuration that intercepts specific requests and returns pre-defined responses without contacting the upstream server     |
| **Static File**                     | Local files served directly by UNCORS instead of proxying to the upstream server                                               |
| **Cache**                           | A mechanism that stores responses from the upstream server to reduce latency on subsequent identical requests                  |
| **Rewrite**                         | A transformation applied to the request path or host before forwarding to the upstream server                                  |
| **Script Handler**                  | Custom Lua code that generates dynamic responses based on request properties                                                   |
| **OPTIONS Handling**                | Built-in processing of HTTP OPTIONS requests for CORS preflight checks                                                         |

## Documentation

- [Installation](./Installation)
- [Configuration](./Configuration)
- [Response mocking](./Response-Mocking)
- [Static file serving](./Static-File-Serving)
- [Response caching](./Response-Caching)
- [Request rewriting](./Request-Rewriting)
- [Migration guide](./Migration-Guide)
- [Script handler](./Script-Handler)
- [Troubleshooting](./Troubleshooting)
- [Real-world examples](./Real-World-Examples)

## List of core features

- CORS header replacement
- [HTTPS support](./Configuration#https-configuration)
- [Wildcard host mapping](./Configuration#wildcard-mapping)
- [HTTP/HTTPS proxy support](./Configuration#proxy-configuration)
- [Response mocking](./Response-Mocking)
- [Script handler](./Script-Handler) (Lua scripting with JSON support)
- [Static file serving](./Static-File-Serving)
- [Response caching](./Response-Caching)
- [Request rewriting](./Request-Rewriting)

## Overview

UNCORS enables developers to make browser requests to APIs that would typically be blocked by CORS (Cross-Origin Resource Sharing) policies. This tool is particularly valuable during application development and testing phases, as it eliminates the need to run backend services locally or modify server configurations. Key capabilities include support for local domain mapping and flexible wildcard-based domain matching.

> [!CAUTION]
> Please be aware that the modification or replacement of CORS headers may introduce potential security vulnerabilities.
> This tool is specifically engineered to optimize the development and testing workflow and is not intended for use in a
> production environment or as a remote proxy server. It has not undergone a thorough security review; therefore, caution
> should be exercised when utilizing it.
