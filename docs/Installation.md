UNCORS is a single binary. Install it with a package manager, download a
release, run it in Docker, or build it from source. After that, add your local
domains to the hosts file.

## Package managers

### Homebrew (macOS, Linux)

```bash
brew install evg4b/tap/uncors
```

### Scoop (Windows)

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

### npm (cross-platform)

Run it once without installing:

```bash
npx -y uncors --from 'http://localhost:8080' --to 'https://github.com'
```

Or add it to your project as a dev dependency:

```bash
npm install uncors --save-dev
# or
yarn add uncors --dev
# or
pnpm add -D uncors
```

### Stew (cross-platform)

With the [Stew](https://github.com/marwanhawari/stew) package manager:

```bash
stew install evg4b/uncors
```

## Docker

Images are published on
[Docker Hub](https://hub.docker.com/r/evg4b/uncors). Inside the container,
UNCORS listens on the port from the `from` URL, which is 80 when none is given:

```bash
docker run -p 80:80 evg4b/uncors --interactive=false --from 'http://local.github.com' --to 'https://github.com'
```

`--interactive=false` turns off the terminal UI, which needs an interactive
terminal, and prints requests as plain log lines. To use a configuration file,
mount it into the container and pass its path with `--config`.

## Prebuilt binaries

Binaries for all supported platforms are attached to each
[GitHub release](https://github.com/evg4b/uncors/releases/latest):

1. Download the archive for your operating system and architecture.
2. Extract the `uncors` binary.
3. Optionally, move it to a directory on your `PATH`, such as `/usr/local/bin`
   on Linux and macOS.

The binary has no runtime dependencies.

## Build from source

You need [Git](https://git-scm.com/) and the [Go](https://go.dev/) version
listed in `go.mod` (currently 1.26.4) or newer.

```bash
git clone https://github.com/evg4b/uncors.git
cd uncors
go install -tags release .
```

With the `release` build tag, an unexpected crash prints a short error message
instead of re-panicking with a full stack trace. `make install` builds the same
way and also embeds the current commit as the version.

## Hosts file setup

A browser only sends a request to UNCORS if the host name resolves to your
machine. Add each local domain you use in `from` to the hosts file, pointing at
`127.0.0.1`.

### macOS and Linux

Edit `/etc/hosts` as root, for example with `sudo nano /etc/hosts`, and add
lines like these:

```
127.0.0.1 api.local
127.0.0.1 app.local
127.0.0.1 admin.local
```

Or append a single entry from the shell:

```bash
echo "127.0.0.1 api.local" | sudo tee -a /etc/hosts
```

Check the result with `ping api.local`; replies should come from `127.0.0.1`.

### Windows

1. Run Notepad as administrator: press the Windows key, type "Notepad",
   right-click it, and choose "Run as administrator".
2. Open `C:\Windows\System32\drivers\etc\hosts`. In the Open dialog, switch the
   file filter from "Text Documents (\*.txt)" to "All Files (\*.\*)" to see
   it.
3. Add your entries, for example `127.0.0.1 api.local`, and save.
4. Check the result with `ping api.local`.

From an administrator PowerShell window you can append an entry directly:

```powershell
Add-Content -Path C:\Windows\System32\drivers\etc\hosts -Value "`n127.0.0.1 api.local"
```

### No wildcards

The hosts file does not support wildcards such as `*.local.com`. List every
subdomain on its own line:

```
127.0.0.1 sub1.local.com
127.0.0.1 sub2.local.com
```

A [placeholder mapping](Configuration#named-placeholder-mapping) such as
`http://{name}.local.com:8080` then handles all of them with one mapping
entry.

### When changes don't take effect

Flush the DNS cache:

```bash
# macOS
sudo dscacheutil -flushcache && sudo killall -HUP mDNSResponder

# Linux with systemd-resolved
sudo systemctl restart systemd-resolved
```

```cmd
:: Windows
ipconfig /flushdns
```

Browsers keep their own DNS cache. In Chrome, open
`chrome://net-internals/#dns` and click "Clear host cache". In Firefox and
Safari, restart the browser.

If `ping` resolves to the wrong address, look for typos and duplicate entries
for the same name.

> [!CAUTION]
> Entries in the hosts file affect every program on your machine. Add only
> names you use for local development. If you add a real production domain, you
> won't reach the real site until you remove the entry.

The `.test` top-level domain is reserved for testing, so names under it never
collide with real sites. Remove entries when a project is finished.

## Next steps

1. Write a configuration file; see [Configuration](Configuration).
2. Start UNCORS with `uncors --config .uncors.yaml`.
3. Open your local domain in a browser or API client.
