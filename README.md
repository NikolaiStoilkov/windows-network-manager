# windows-network-manager

A Go-based tool that lets you run **multiple browser instances simultaneously**, each bound to a **different network interface / IP address** and using its **own DNS server**. It achieves this by spinning up a lightweight HTTP/HTTPS proxy per interface and launching a browser with that proxy configured.

---

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Project Structure](#project-structure)
- [Dependencies](#dependencies)
- [Configuration](#configuration)
- [Running the Project](#running-the-project)
- [Building for Windows](#building-for-windows)
- [Extending the Code](#extending-the-code)
- [Limitations & Notes](#limitations--notes)

---

## Overview

In environments where a machine has multiple network adapters (e.g., wired LAN on `192.168.33.x`, a second LAN on `192.168.44.x`, and a third on `192.168.35.x`), traffic from regular browsers always exits through the OS default route. This tool solves that by:

1. Starting one per-interface **local proxy** (bound to a specific local IP).
2. Launching browsers with `--proxy-server` pointing to that local proxy.
3. Each proxy resolves DNS through the **interface-specific DNS server** and dials outbound connections from the **assigned local IP**.

---

## How It Works

```
Browser (Chrome / Edge / Opera)
        │  HTTP/HTTPS request
        ▼
 127.0.0.1:<ProxyPort>   ← goproxy HTTP proxy
        │
        ├── DNS lookup via <DNSServer>:53  (UDP, bound to <InterfaceIP>)
        │
        └── TCP dial from <InterfaceIP> to resolved destination
```

### Key components

| Component | Description |
|-----------|-------------|
| `NetworkConfig` | Struct that holds all settings for one network interface / browser pair |
| `startProxy()` | Creates a `goproxy` server with a custom dialer and DNS resolver tied to `InterfaceIP` |
| `launchBrowser()` | Starts a browser process with `--proxy-server` and an isolated `--user-data-dir` |
| `main()` | Wires everything together: starts proxies as goroutines, then launches browsers |

---

## Project Structure

```
windows-network-manager/
├── main.go                  # All application logic
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── network_manager.exe      # Pre-built Windows binary (no GUI)
├── network_manager_gui.exe  # Pre-built Windows binary (GUI variant)
└── README.md
```

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| [`github.com/elazarl/goproxy`](https://github.com/elazarl/goproxy) | v1.8.3 | HTTP/HTTPS proxy framework |
| `golang.org/x/net` | v0.43.0 | Extended networking support (transitive) |
| `golang.org/x/text` | v0.28.0 | Text utilities (transitive) |

Install dependencies:

```bash
go mod download
```

---

## Configuration

Configuration lives inside `main()` as a slice of `NetworkConfig` structs. Edit this slice to match your actual network adapters.

```go
type NetworkConfig struct {
    InterfaceIP string // Local IP of the network adapter to bind to
    DNSServer   string // DNS server reachable via that adapter
    ProxyPort   string // Local port the proxy will listen on (127.0.0.1)
    BrowserPath string // Absolute path to the browser executable (Windows)
    ProfileName string // Unique name used for the browser's user-data-dir
}
```

### Example

```go
configs := []NetworkConfig{
    {
        InterfaceIP: "192.168.33.200",          // Adapter on LAN 33
        DNSServer:   "1.1.1.1",                 // Cloudflare DNS
        ProxyPort:   "8081",                    // Proxy listens on 127.0.0.1:8081
        BrowserPath: `C:\Program Files\Google\Chrome\Application\chrome.exe`,
        ProfileName: "chrome_lan_33",           // Isolated Chrome profile
    },
    // Add more entries for additional interfaces...
}
```

> **Port conflicts** – make sure `ProxyPort` values are unique and not in use by other services.

---

## Running the Project

### Prerequisites

- Go 1.21+ (module requires `go 1.26` in go.mod – adjust if needed)
- Windows machine (or cross-compile from another OS — see below)
- At least one additional network adapter with a configured static IP

### Run directly

```bash
go run main.go
```

The application will:
1. Print a line for each started proxy, e.g. `[+] Proxy on :8081 bound to 192.168.33.200 (DNS: 1.1.1.1)`
2. Wait 2 seconds for proxies to become ready.
3. Launch each browser with the matching proxy settings.
4. Block indefinitely until you press **Ctrl+C**.

---

## Building for Windows

Two ready-made shell scripts are provided for convenience:

| Script | Output | Description |
|--------|--------|-------------|
| `build_console.sh` | `network_manager.exe` | Console binary — shows a terminal window when launched |
| `build_no_console.sh` | `network_manager_no_console.exe` | Silent binary — no console window, runs in the background |

### Using the build scripts (macOS / Linux)

```bash
# Make scripts executable (first time only)
chmod +x build_console.sh build_no_console.sh

# Build console binary
./build_console.sh

# Build no-console binary
./build_no_console.sh
```

### Manual build commands

#### Console binary (shows terminal window)
```bash
GOOS=windows GOARCH=amd64 go build -o network_manager.exe .
```

#### No-console binary (no console window)
```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o network_manager_no_console.exe .
```

#### On Windows directly
```bash
# Console
go build -o network_manager.exe .

# No-console (no console window)
go build -ldflags="-H windowsgui" -o network_manager_no_console.exe .
```

---

## Extending the Code

### Add a new interface / browser

Simply append another `NetworkConfig` entry to the `configs` slice in `main()`:

```go
{
    InterfaceIP: "10.0.0.50",
    DNSServer:   "10.0.0.1",
    ProxyPort:   "8084",
    BrowserPath: `C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
    ProfileName: "brave_lan_10",
},
```

### Changing DNS transport (TCP instead of UDP)

In `startProxy()`, change the dial network in the resolver from `"udp"` to `"tcp"`:

```go
return d.DialContext(ctx, "tcp", config.DNSServer+":53")
```

### Adding HTTPS (CONNECT) support

`goproxy` supports HTTPS tunneling out of the box. To intercept HTTPS traffic (e.g., for logging), set up a `HandleConnect` handler before starting the proxy:

```go
proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
```

---

## Limitations & Notes

- **Windows-only at runtime** – browser paths and `user-data-dir` flags use Windows-style paths. The proxy logic itself is cross-platform.
- **Static config** – there is currently no config file or CLI flags; all settings are hard-coded in `main()`. Consider adding a JSON/YAML config loader for production use.
- **No proxy authentication** – the local proxy listens only on `127.0.0.1` so it is not exposed to the network, but there is no authentication layer.
- **DNS caching** – there is no DNS cache in the current implementation; every connection performs a fresh lookup. Add a simple TTL-based cache if performance matters.
- **Browser must support `--proxy-server`** – Chrome, Edge, Opera, Brave, and most Chromium-based browsers do. Firefox uses a different flag scheme and would require adjustments.
