UNCORS provides multiple installation methods to suit different development
environments and preferences. Choose the method that best fits your workflow.

## Package Managers

### Homebrew (macOS | Linux)

For macOS or Linux users with [Homebrew](https://brew.sh/) installed:

```bash
brew install evg4b/tap/uncors
```

### Scoop (Windows)

Windows users with [Scoop](https://scoop.sh/):

```bash
scoop bucket add evg4b https://github.com/evg4b/scoop-bucket.git
scoop install evg4b/uncors
```

### NPM (Cross-platform)

UNCORS can be installed as a Node.js package using your preferred package
manager:

**npm:**

```bash
npm install uncors --save-dev
```

**yarn:**

```bash
yarn add uncors --dev
```

**pnpm:**

```bash
pnpm add -D uncors
```

### Docker (Cross-platform)

Docker images are available on [Docker
Hub](https://hub.docker.com/r/evg4b/uncors):

```bash
docker run -p 80:3000 evg4b/uncors --from 'http://local.github.com' --to 'https://github.com'
```

## Binary Installation

### Stew (Cross-platform)

For users of the [Stew](https://github.com/marwanhawari/stew) package manager:

```bash
stew install evg4b/uncors
```

### Direct Binary Download (Cross-platform)

Pre-compiled binaries are available for all major platforms on the [UNCORS
releases page](https://github.com/evg4b/uncors/releases/latest).

**Installation steps:**

 1. Download the appropriate binary for your operating system and architecture
 2. Extract the binary to your preferred location
 3. (Optional) Add the binary to your `PATH` for convenient access

**Recommended installation paths:**

 - **Linux|macOS:** `/usr/local/bin`
 - **Windows:** Add to a directory included in your system's `PATH` environment
   variable

The binary is self-contained and can be executed from any location without
additional dependencies.

## Build from Source

**Prerequisites:**

 - [Git](https://git-scm.com/)
 - [Go](https://go.dev/) 1.21 or later

**Build instructions:**

```bash
# Clone the repository
git clone https://github.com/evg4b/uncors.git
cd uncors

# Build and install
go install -tags release
```

## Post-Installation: Hosts File Setup

UNCORS works by mapping local domains to remote servers. To use UNCORS
effectively, you need to configure your system's hosts file to resolve custom
domain names to localhost.

### Understanding the Hosts File

The hosts file is a system file that maps hostnames to IP addresses. UNCORS
listens on localhost and requires entries in your hosts file to route traffic
through it.

### macOS and Linux

**1. Open the hosts file with root privileges:**

```bash
sudo nano /etc/hosts
```

Or use your preferred editor:

```bash
sudo vim /etc/hosts
```

**2. Add your domain mappings:**

```
127.0.0.1 api.local
127.0.0.1 app.local
127.0.0.1 admin.local
```

**3. Save and exit:**

 - In nano: Press `Ctrl+O` to save, then `Ctrl+X` to exit
 - In vim: Press `Esc`, type `:wq`, then press `Enter`

**4. Verify the configuration:**

```bash
ping api.local
# Should respond from 127.0.0.1
```

**Quick one-liner:**

```bash
echo "127.0.0.1 api.local" | sudo tee -a /etc/hosts
```

### Windows

**1. Run Notepad as Administrator:**

 - Press `Win` key, type "Notepad"
 - Right-click on "Notepad" and select "Run as administrator"

**2. Open the hosts file:**

 - In Notepad, click File → Open
 - Navigate to: `C:\Windows\System32\drivers\etc`
 - Change file filter from "Text Documents (_.txt)" to "All Files (_.\*)"
 - Select `hosts` and click Open

**3. Add your domain mappings:**

```
127.0.0.1 api.local
127.0.0.1 app.local
127.0.0.1 admin.local
```

**4. Save the file** (File → Save). If you get an access denied error, ensure
Notepad is running as Administrator.

**5. Verify the configuration:**

```cmd
ping api.local
```

Should respond from 127.0.0.1.

### Alternative: PowerShell (Windows)

Run PowerShell as Administrator:

```powershell
Add-Content -Path C:\Windows\System32\drivers\etc\hosts -Value "`n127.0.0.1 api.local"
```

### Common Domain Patterns

```
# API development
127.0.0.1 api.local
127.0.0.1 api.dev
127.0.0.1 local.api.com

# Frontend development
127.0.0.1 app.local
127.0.0.1 frontend.local

# Microservices
127.0.0.1 auth.local
127.0.0.1 users.local
127.0.0.1 payments.local

# Note: Hosts file does NOT support wildcards
# Each subdomain must be listed explicitly
127.0.0.1 sub1.local.com
127.0.0.1 sub2.local.com
```

> [!WARNING]
> The hosts file does not support wildcard entries like `*.local.com`. Each
> subdomain must be listed individually. However, UNCORS configuration supports
> named placeholder mappings (e.g., `http://{name}.local.com:8080` →
> `https://{name}.example.com`) for domains explicitly listed in your hosts file.

### Troubleshooting Hosts File Changes

**Changes not taking effect - flush DNS cache:**

```bash
# macOS
sudo dscacheutil -flushcache && sudo killall -HUP mDNSResponder

# Linux (systemd)
sudo systemctl restart systemd-resolved
```

```cmd
# Windows
ipconfig /flushdns
```

**Permission denied:** Ensure you are running your editor with
administrator/root privileges.

**Ping fails or wrong IP:** Check for typos, duplicate entries, extra
whitespace, and confirm you are using `127.0.0.1`.

**Browser not resolving domain:**

 - Chrome: Visit `chrome://net-internals/#dns` and click "Clear host cache"
 - Firefox: Restart the browser
 - Safari: Close all windows and restart

### Security Considerations

> [!CAUTION]
> Modifying the hosts file can affect your system's network behavior. Only add
> entries for domains you control or are using for local development. Never add
> entries for production domains you don't own.

**Best practices:**

 1. Use `.local` or `.dev` TLDs for development
 2. Document your hosts file changes
 3. Remove entries when a project is finished
 4. Never commit hosts file changes to version control

## Next Steps

Once your hosts file is configured:

 1. Create an UNCORS configuration file (see [Configuration](Configuration))
 2. Start UNCORS: `uncors --config .uncors.yaml`
 3. Access your mapped domains through your browser or API client
