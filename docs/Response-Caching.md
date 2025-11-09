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
  clear-time: 15m
```

## Configuration Properties

| Property          | Type     | Default | Description                                        |
| ----------------- | -------- | ------- | -------------------------------------------------- |
| `methods`         | array    | `[GET]` | HTTP methods to cache (e.g., `GET`, `POST`, `PUT`) |
| `expiration-time` | duration | -       | Time until cached response is considered stale     |
| `clear-time`      | duration | -       | Time until cached response is permanently removed  |

**Duration format:** `<number><unit>` where unit is `s` (seconds), `m` (minutes), or `h` (hours)

**Examples:**

- `30s` - 30 seconds
- `5m` - 5 minutes
- `2h` - 2 hours
- `1h 30m` - 1 hour 30 minutes

## Cache Lifecycle

1. **Fresh**: Response is returned immediately from cache
2. **Stale** (after `expiration-time`): Response is revalidated with upstream server
3. **Removed** (after `clear-time`): Cache entry is deleted, next request fetches fresh data

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
  clear-time: 1h
```

### Cache Multiple Methods

```yaml
cache-config:
  methods: [GET, POST, PUT]
  expiration-time: 2m
  clear-time: 30m

mappings:
  - from: http://localhost
    to: https://api.example.com
    cache:
      - /api/search
      - /api/query/**
```
