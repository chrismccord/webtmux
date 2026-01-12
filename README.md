# webtmux

A web-based terminal with tmux-specific features. Access your tmux sessions from any browser with a visual pane layout, touch-friendly controls, and automatic scroll-to-copy-mode.

## Quick Start (Sprite)

Deploy webtmux as a service on [Sprite](https://sprites.app):

```bash
curl -fsSL https://raw.githubusercontent.com/chrismccord/webtmux/main/builds/webtmux-linux-amd64 \
  -o /usr/local/bin/webtmux && \
  chmod +x /usr/local/bin/webtmux && \
  sprite-env services create webtmux \
    --cmd /usr/local/bin/webtmux \
    --args '-w,--no-auth,tmux,new-session,-A,-s,main' \
    --http-port 8080
```

## Features

- **Visual Pane Layout**: Sidebar minimap shows your tmux pane arrangement - click to switch panes
- **Window Tabs**: Quick window switching via clickable tabs
- **Touch-Friendly**: Mobile controls for split, new window, and pane switching
- **Scroll-to-Copy-Mode**: Scroll up automatically enters tmux copy mode
- **Secure by Default**: HTTP Basic Auth with auto-generated credentials
- **Single Binary**: All assets embedded - just download and run
- **Real-time Updates**: Layout changes sync automatically

## Installation

### Prebuilt Binaries

Prebuilt binaries are available in the `builds/` directory for all major platforms:

| Platform | Binary |
|----------|--------|
| Linux (x64) | `builds/webtmux-linux-amd64` |
| Linux (ARM64) | `builds/webtmux-linux-arm64` |
| Linux (ARM) | `builds/webtmux-linux-arm` |
| macOS (Intel) | `builds/webtmux-darwin-amd64` |
| macOS (Apple Silicon) | `builds/webtmux-darwin-arm64` |
| FreeBSD (x64) | `builds/webtmux-freebsd-amd64` |

```bash
# Clone and use prebuilt binary (example for Linux x64)
git clone https://github.com/chrismccord/webtmux.git
cd webtmux
chmod +x builds/webtmux-linux-amd64
./builds/webtmux-linux-amd64 -w tmux new-session -A -s main

# Or copy to your PATH
sudo cp builds/webtmux-linux-amd64 /usr/local/bin/webtmux
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/chrismccord/webtmux.git
cd webtmux

# Build for current platform
make build

# Or cross-compile for all platforms
make cross-compile
```

## Usage

### Basic Usage

```bash
# Start with tmux (auto-generates credentials)
webtmux -w tmux new-session -A -s main

# Output:
# ========================================
#   Authentication Required (default)
#   Username: admin
#   Password: <random-32-char-password>
# ========================================
```

### Custom Credentials

```bash
webtmux -w -c user:password tmux new-session -A -s main
```

### Disable Authentication (not recommended)

```bash
webtmux -w --no-auth tmux new-session -A -s main
```

### Common Options

| Flag | Description |
|------|-------------|
| `-w, --permit-write` | Allow input to the terminal (required for interactive use) |
| `-p, --port PORT` | Port to listen on (default: 8080) |
| `-a, --address ADDR` | Address to bind to (default: 0.0.0.0) |
| `-c, --credential USER:PASS` | Set custom credentials for HTTP Basic Auth |
| `--no-auth` | Disable authentication (NOT RECOMMENDED) |
| `--ws-origin REGEX` | Regex for allowed WebSocket origins |
| `-t, --tls` | Enable TLS/SSL |
| `--tls-crt FILE` | TLS certificate file |
| `--tls-key FILE` | TLS key file |
| `-r, --random-url` | Add random string to URL path |
| `--reconnect` | Enable automatic reconnection |
| `--once` | Accept only one client, then exit |

Run `webtmux --help` for all available options.

## Architecture

```
Browser                              Go Backend
+------------------+                +------------------+
| xterm.js         |<--WebSocket-->| webtty core      |<--PTY--> tmux
| Lit.js Sidebar   |   (extended)  | tmux controller  |
| Touch Controls   |               |                  |
+------------------+                +------------------+
```

### Extended WebSocket Protocol

WebTmux extends the gotty protocol with tmux-specific message types:

**Client -> Server:**
- `5` TmuxSelectPane - Switch to pane by ID
- `6` TmuxSelectWindow - Switch to window by ID
- `7` TmuxSplitPane - Split current pane (h/v)
- `8` TmuxClosePane - Close pane by ID
- `9` TmuxCopyMode - Enter/exit copy mode
- `B` TmuxScrollUp - Scroll up in copy mode
- `C` TmuxScrollDown - Scroll down in copy mode
- `D` TmuxNewWindow - Create new window

**Server -> Client:**
- `7` TmuxLayoutUpdate - Full layout JSON
- `9` TmuxModeUpdate - Copy mode state

## Development

### Project Structure

```
webtmux/
├── main.go                 # CLI entry point
├── server/                 # HTTP server & WebSocket handlers
├── webtty/                 # WebTTY protocol implementation
├── pkg/tmux/               # Tmux controller
├── backend/localcommand/   # PTY backend
├── bindata/static/         # Embedded web assets
│   ├── js/
│   │   ├── webtmux.js      # Main frontend
│   │   └── components/     # Lit.js web components
│   └── index.html
└── resources/              # Source assets (for development)
```

### Building

```bash
# Development build (copies fresh assets)
make dev

# Production build
make build

# Cross-compile all platforms
make cross-compile

# Create release archives
make release
```

### Tech Stack

- **Backend**: Go, gorilla/websocket
- **Frontend**: xterm.js, Lit.js, Tailwind CSS (CDN)
- **Embedded Assets**: Go 1.16+ embed directive

## Credits

WebTmux is a fork of [gotty](https://github.com/yudai/gotty) by Iwasaki Yudai.

## License

MIT License - See [LICENSE](LICENSE) file for details.
