# Installation

Moonbase is a single self-contained binary (the 14 agents are embedded via `go:embed`) for **macOS and Linux**. It works in any project with no repository checkout — `moonbase init` / `moonbase setup` just work from anywhere.

## Install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/rk-senne/moonbase/main/install.sh | sh
```

Detects your OS/arch, downloads the latest release, verifies its checksum, installs the binary to `~/.local/bin`, and runs `moonbase setup`. Override the target dir with `MOONBASE_INSTALL_DIR` or pin a version with `MOONBASE_VERSION=v1.16.0`.

## With Go

```bash
go install github.com/rk-senne/moonbase/cmd/moonbase@latest
moonbase setup   # install the embedded agents to ~/.moonbase/agents
```

Requires **Go 1.26.5+** (the toolchain auto-downloads it if needed).

## Download a release

Grab `moonbase_<os>_<arch>.tar.gz` from the [Releases page](https://github.com/rk-senne/moonbase/releases), extract it, put `moonbase` on your `PATH`, then run `moonbase setup`.

## Build from source

```bash
git clone https://github.com/rk-senne/moonbase.git
cd moonbase && make build && cp bin/moonbase ~/.local/bin/
```

## Private / internal use (current)

The repository is currently private, so the install script and a plain `go install` need access. Until it's public, install within the org by either:

```bash
# Build from source (you have repo access)
git clone git@github.com:rk-senne/moonbase.git
cd moonbase && make build && cp bin/moonbase ~/.local/bin/

# …or go install with private-module + git auth configured
GOPRIVATE=github.com/rk-senne/* go install github.com/rk-senne/moonbase/cmd/moonbase@latest
```

No engineering change is needed to go public later: the release owner is derived from the binary's module path, goreleaser auto-detects the repo from the git remote, and the install script targets public release assets.

## Verify

```bash
moonbase version   # prints version + commit + build time
moonbase status    # environment health check
moonbase lint      # validates all 14 agent files
```
