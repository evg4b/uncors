UNCORS can keep upstream responses in memory and answer repeated requests from
that cache. This helps when an API is slow, rate-limited, or returns large data
that doesn't change while you work.

Caching is turned on per mapping by listing path globs under `cache`:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/info
      - /api/users/**
```

## What gets cached

A response is cached when all of these hold:

- the request is proxied to the upstream server (mock, script, and static
  responses are never cached);
- the request is not a WebSocket handshake or another protocol upgrade;
- the request method is listed in `cache-config.methods` (`GET` by default);
- the request path matches one of the mapping's `cache` globs;
- the upstream status code is in the 2xx range;
- the response has no `Set-Cookie` header;
- the upstream `Cache-Control` header has neither `no-store` nor `private`;
- the upstream `Vary` header is not `*`;
- the response body is at most 10 MiB. Larger responses, such as long event
  streams, are passed through without being stored.

## Cache key

The cache key is made of:

- the method, scheme, host name and port, and path, so mappings on different
  ports never share entries;
- the query string. Parameter names are sorted, so `?a=1&b=2` and `?b=2&a=1`
  share an entry. The values of a repeated parameter keep their order:
  `?a=1&a=2` and `?a=2&a=1` are cached separately;
- the request's `Accept-Encoding`, compared without case, spaces, or order, so
  `gzip, br` and `br,gzip` share an entry. UNCORS forwards this header, and
  the upstream server may compress the body for it. A client that sends no
  `Accept-Encoding`, such as `curl` without `--compressed`, never gets a body
  compressed for a browser;
- the request headers listed in the upstream `Vary` header. With
  `Vary: Accept-Language`, requests in different languages are cached
  separately. `Origin` and `Accept-Encoding` in `Vary` add nothing: CORS headers
  are written for each request, and `Accept-Encoding` is always in the key.

Other request headers are not part of the key.

## Cache hits

A cache hit returns the stored status code, headers, and body without
contacting the upstream server, with an `Age` header: the seconds since the
response was stored, plus the upstream `Age` if it sent one. The stored entry
lives for `expiration-time` whatever the upstream `Cache-Control: max-age` or
`Expires` says. `Access-Control-*` headers are not stored.
Each hit gets the CORS headers written for its own `Origin`, as a proxied
response would, so a response cached for one page can be served to another.
When the mapping has [`allowed-origins`](Configuration#allowed-origins) and the
origin does not match, the hit has no CORS headers, as with any other response.

The cache lives in memory only. It is emptied when UNCORS stops or reloads its
configuration.

## Glob syntax

Globs are matched against the request path with
[doublestar](https://github.com/bmatcuk/doublestar):

| Pattern      | Matches                                                                              |
| ------------ | ------------------------------------------------------------------------------------ |
| `*`          | Any sequence of characters except `/`                                                |
| `**`         | Zero or more path segments, when it is a whole segment (`/api/**`, `/api/**/x.json`) |
| `?`          | Any single character except `/`                                                      |
| `[class]`    | One character from a class (see below)                                               |
| `{alt1,...}` | Any one of the comma-separated alternatives                                          |

`**` only works as a complete path segment. `/data/**.json` behaves like
`/data/*.json`; write `/data/**/*.json` to include subdirectories. Escape a
special character with a backslash, for example `\*`.

Character classes:

| Class      | Matches                                    |
| ---------- | ------------------------------------------ |
| `[abc]`    | One of the listed characters               |
| `[a-z]`    | One character in the range                 |
| `[^class]` | One character that is not in the class     |
| `[!class]` | Same as `[^class]`                         |

## Global cache settings

`cache-config` is a top-level section shared by all mappings:

```yaml
cache-config:
  methods: [GET]
  expiration-time: 10m
  max-size: 104857600
```

| Option            | Type     | Default     | Description                                                           |
| ----------------- | -------- | ----------- | --------------------------------------------------------------------- |
| `methods`         | array    | `[GET]`     | HTTP methods whose responses can be cached. `CONNECT` is not allowed. |
| `expiration-time` | duration | `30m`       | How long an entry is kept.                                            |
| `max-size`        | integer  | `104857600` | Total cache size in bytes (100 MB by default).                        |

`expiration-time` uses Go duration syntax without spaces, for example `30s`,
`5m`, `2h`, or `1h30m`. The cache is built on
[ristretto](https://github.com/dgraph-io/ristretto). When it reaches
`max-size`, ristretto evicts entries it considers least useful, and it may
also decline to store a new entry. A response larger than `max-size` is never
stored. With [`UNCORS_TRACE`](Configuration#tracing-and-diagnostics) on, a
declined response adds the `uncors.cache.rejected` event to its `cache.store`
span and is counted by the `uncors.cache.rejections` metric.

## Examples

Cache a few API paths for five minutes:

```yaml
cache-config:
  expiration-time: 5m
  max-size: 52428800

mappings:
  - from: http://localhost
    to: https://api.example.com
    cache:
      - /api/users
      - /api/posts/*
      - /api/data/**/*.json
```

Cache search requests sent with `POST` as well:

```yaml
cache-config:
  methods: [GET, POST]
  expiration-time: 2m

mappings:
  - from: http://localhost
    to: https://api.example.com
    cache:
      - /api/search
      - /api/query/**
```

The request body is not part of the cache key, so two `POST` requests to the
same URL with different bodies share one cache entry.
