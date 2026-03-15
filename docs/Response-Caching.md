UNCORS provides response caching to optimize development workflows by reducing latency for expensive or frequently repeated requests. Cache entries are matched using URL glob patterns.

**Benefits:**

- Faster response times for repeated requests
- Reduced load on upstream servers during development
- Improved performance when working with slow APIs
- Useful for caching heavy computations or large datasets

**Configuration:**

Specify URL patterns to cache for each mapping:

```yaml
mappings:
  - from: ...
    to: ...
    cache:
      - /api/info
      - /api/users/**
```

# Pattern Syntax

Cache patterns use glob syntax to match URL paths. The following special characters are supported:

| Special Terms | Meaning                                                                                                   |
| ------------- | --------------------------------------------------------------------------------------------------------- |
| `*`           | matches any sequence of non-path-separators                                                               |
| `/**/`        | matches zero or more directories                                                                          |
| `?`           | matches any single non-path-separator character                                                           |
| `[class]`     | matches any single non-path-separator character against a class of characters ([see "character classes"]) |
| `{alt1,...}`  | matches a sequence of characters if one of the comma-separated alternatives matches                       |

**Important notes:**

- Escape special characters with backslash: `\*`, `\?`, `\[`
- Double star `**` must be surrounded by path separators: `/**/`
- Incorrect: `path/to/**.txt` (acts like `path/to/*.txt`)
- Correct: `path/to/**/*.txt` (matches files in subdirectories)

## Character Classes

Character classes match single characters against a set or range:

| Class      | Meaning                                                       |
| ---------- | ------------------------------------------------------------- |
| `[abc]`    | matches any single character within the set                   |
| `[a-z]`    | matches any single character in the range                     |
| `[^class]` | matches any single character which does _not_ match the class |
| `[!class]` | same as `^`: negates the class                                |

# Global Cache Configuration

Configure caching behavior globally using the `cache-config` section:

```yaml
cache-config:
  methods: [GET]
  expiration-time: 10m
  max-size: 104857600
```

## Configuration Properties

| Property          | Type     | Default      | Description                                        |
| ----------------- | -------- | ------------ | -------------------------------------------------- |
| `methods`         | array    | `[GET]`      | HTTP methods to cache (e.g., `GET`, `POST`, `PUT`) |
| `expiration-time` | duration | `30m`        | Time until a cached response is evicted            |
| `max-size`        | integer  | `104857600`  | Maximum total cache size in bytes (default 100 MB) |

**Duration format:** `<number><unit>` where unit is `s` (seconds), `m` (minutes), or `h` (hours)

**Examples:**

- `30s` - 30 seconds
- `5m` - 5 minutes
- `2h` - 2 hours
- `1h 30m` - 1 hour 30 minutes

## Cache Lifecycle

1. **Hit**: Response is returned immediately from cache
2. **Miss**: Request is forwarded to the upstream server and the response is stored in cache
3. **Evicted** (after `expiration-time` or when `max-size` is reached): Cache entry is removed; the next request fetches fresh data from upstream

## Examples

### Cache API Responses

```yaml
mappings:
  - from: http://localhost
    to: https://api.example.com
    cache:
      - /api/users
      - /api/posts/*
      - /api/data/**/*.json

cache-config:
  methods: [GET]
  expiration-time: 5m
  max-size: 52428800
```

### Cache Multiple Methods

```yaml
cache-config:
  methods: [GET, POST, PUT]
  expiration-time: 2m
  max-size: 104857600

mappings:
  - from: http://localhost
    to: https://api.example.com
    cache:
      - /api/search
      - /api/query/**
```
