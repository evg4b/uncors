UNCORS can serve files from a local directory under a URL path. Typical uses
are running a local build of a single-page app against a remote API, or
replacing a few remote assets with local copies.

```yaml
mappings:
  - from: http://app.local:3000
    to: https://example.com
    statics:
      - path: /assets
        dir: ./assets
      - path: /app
        dir: ./dist
        index: index.html
```

## Options

| Option  | Type   | Required | Description                                                                   |
| ------- | ------ | -------- | ----------------------------------------------------------------------------- |
| `path`  | string | Yes      | URL path prefix, starting with `/`. Wildcards and placeholders are not supported. |
| `dir`   | string | Yes      | Local directory to serve. It must exist when UNCORS starts.                   |
| `index` | string | No       | Fallback file, relative to `dir`, returned when the requested file is missing. |

Relative `dir` paths are resolved against the directory you start UNCORS from.
`~` is not expanded.

`statics` can also be written as a map from path to directory. A value can be a
directory string or an object with `dir` and `index`:

```yaml
statics:
  /assets: ./assets
  /app:
    dir: ./dist
    index: index.html
```

## How requests are handled

A request whose path starts with `path` is looked up in `dir`, with the prefix
removed. With `path: /assets` and `dir: ./assets`, a request for
`/assets/css/site.css` reads `./assets/css/site.css`. A request for exactly
`/assets` is redirected to `/assets/`.

What happens next depends on whether the file exists and whether `index` is
set:

| Requested file   | `index` set                     | `index` not set                     |
| ---------------- | ------------------------------- | ----------------------------------- |
| Exists           | The file is returned            | The file is returned                |
| Is a directory   | The `index` file is returned    | The request goes to the upstream    |
| Does not exist   | The `index` file is returned    | The request goes to the upstream    |

"Goes to the upstream" means the request is proxied to `to`, with caching and
HAR recording if they are configured. It does not fall through to mocks or
scripts. Static paths are checked before mocks, scripts, and rewrites, so a
mock under a static prefix is never reached.

Files are served with a `Content-Type` based on their extension, and support
conditional and range requests. UNCORS does not add CORS headers to files
served from `dir`.

## Single-page apps

Client-side routers (React Router, Vue Router, Angular Router) use URLs that
don't match files on disk, such as `/app/users/123`. Set `index` so those URLs
return `index.html` and the router can take over:

```yaml
mappings:
  - from: http://app.local:3000
    to: https://example.com
    statics:
      - path: /app
        dir: ./dist
        index: index.html
```

- `/app/bundle.js` returns `./dist/bundle.js`.
- `/app/users/123` returns `./dist/index.html`.
- `/api/users` is outside `/app` and is proxied to `https://example.com`.

With `index` set, every request under `path` is answered locally. If you serve
the app from `/` with an index, nothing on that host reaches the upstream
server, including API calls, mocks, and scripts. To serve an app from `/` and
still proxy its API, put the API on a second host name:

```yaml
mappings:
  - from: http://app.local:3000
    to: https://example.com
    statics:
      - path: /
        dir: ./dist
        index: index.html
  - from: http://api.local:3000
    to: https://api.example.com
    mocks:
      - path: /health
        response:
          code: 200
          raw: ok
```

The app then calls `http://api.local:3000`. Because UNCORS adds CORS headers to
those responses, the cross-origin calls work.

## Overriding remote assets

Without `index`, local files take priority and everything else under `path`
still comes from the upstream server:

```yaml
mappings:
  - from: http://localhost
    to: https://www.example.com
    statics:
      - path: /assets
        dir: ./local-assets
```

If `./local-assets/logo.png` exists, `/assets/logo.png` is served locally.
`/assets/other.png` is fetched from `https://www.example.com/assets/other.png`.

## Several directories

```yaml
mappings:
  - from: http://localhost
    to: https://example.com
    statics:
      - path: /app
        dir: ./dist
        index: index.html
      - path: /docs
        dir: ./documentation
      - path: /images
        dir: ./assets/img
```
