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
- the request method is listed in `cache-config.methods` (`GET` by default);
- the request path matches one of the mapping's `cache` globs;
- the upstream status code is in the 2xx range.

The cache key is the method, host name, path, and query string. Query
parameters are sorted first, so `?a=1&b=2` and `?b=2&a=1` share an entry.
Request headers are not part of the key.

A cache hit returns the stored status code, headers, and body without
contacting the upstream server. The cache lives in memory only. It is emptied
when UNCORS stops or reloads its configuration.

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

| Option            | Type     | Default     | Description                                        |
| ----------------- | -------- | ----------- | -------------------------------------------------- |
| `methods`         | array    | `[GET]`     | HTTP methods whose responses can be cached.        |
| `expiration-time` | duration | `30m`       | How long an entry is kept.                         |
| `max-size`        | integer  | `104857600` | Total cache size in bytes (100 MB by default).     |

`expiration-time` uses Go duration syntax without spaces, for example `30s`,
`5m`, `2h`, or `1h30m`. The cache is built on
[ristretto](https://github.com/dgraph-io/ristretto). When it reaches
`max-size`, ristretto evicts entries it considers least useful, and it may
also decline to store a new entry.

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
