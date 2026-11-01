UNCORS can keep upstream responses in memory and answer repeated requests from
that cache. This helps when an API is slow, rate-limited, or returns large data
that doesn't change while you work.

Turn caching on per mapping by listing path globs under `cache`:

```yaml
mappings:
  - from: http://api.local:3000
    to: https://api.example.com
    cache:
      - /api/info
      - /api/users/**
```

## What gets cached

A response is cached when the request was proxied (mock, script, and static
responses never are), its method is in `cache-config.methods`, and its path
matches one of the globs. The upstream response must also:

- have a 2xx status other than `206 Partial Content`;
- have no `Set-Cookie` header;
- have no `no-store` or `private` in `Cache-Control`, and no `Vary: *`;
- have a body of at most 10 MiB.

A cached entry lives for `expiration-time`, whatever the upstream
`Cache-Control` or `Expires` says. A hit returns the stored status, headers,
and body with an `Age` header, and gets fresh CORS headers for its own
`Origin`.

The cache lives in memory only and is emptied when UNCORS stops or reloads its
configuration.

> [!NOTE]
> The cache key is the method, scheme, host, port, path, and query string
> (parameter names sorted), plus `Accept-Encoding` and the request headers
> named in the upstream `Vary`. Other headers and the request body are not
> part of the key, so two `POST` requests to the same URL with different bodies
> share one entry.

## Glob syntax

Globs are matched against the request path with
[doublestar](https://github.com/bmatcuk/doublestar):

| Pattern      | Matches                                           |
| ------------ | ------------------------------------------------- |
| `*`          | Any sequence of characters except `/`             |
| `**`         | Zero or more path segments, as a whole segment    |
| `?`          | Any single character except `/`                   |
| `[abc]`      | One of the listed characters; `[^abc]` negates it |
| `{alt1,...}` | Any one of the comma-separated alternatives       |

`**` only works as a complete segment: `/data/**.json` behaves like
`/data/*.json`, so write `/data/**/*.json` to include subdirectories.

## Global cache settings

`cache-config` is a top-level section shared by all mappings:

```yaml
cache-config:
  methods: [GET, POST]
  expiration-time: 10m
  max-size: 52428800
```

| Option            | Type     | Default     | Description                                                           |
| ----------------- | -------- | ----------- | --------------------------------------------------------------------- |
| `methods`         | array    | `[GET]`     | HTTP methods whose responses can be cached. `CONNECT` is not allowed. |
| `expiration-time` | duration | `30m`       | How long an entry is kept, for example `30s`, `5m`, or `1h30m`.       |
| `max-size`        | integer  | `104857600` | Total cache size in bytes (100 MiB).                                  |

> [!NOTE]
> The cache is built on [ristretto](https://github.com/dgraph-io/ristretto).
> When it is full, ristretto evicts the entries it considers least useful and
> may decline to store a new one.
