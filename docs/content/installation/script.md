---
title: Install Script
description: Install, update, or remove speedtest-go on Linux and macOS with the install script
weight: 1
---

The install script downloads a release from [GitHub Releases](https://github.com/nicholas-fedor/speedtest-go/releases), verifies its checksum, and installs it:

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/scripts/install.sh -o "$tmp" && sh "$tmp"
```

How it installs:

- On Debian, Fedora, Alpine, Arch Linux, and their derivatives, it installs the native package when it runs as root or can use `sudo`.
- Otherwise, it extracts the release archive into `$HOME/go/bin`.
- Either way, it verifies the signature of the checksums when [cosign](https://github.com/sigstore/cosign) is installed.

## Options

These environment variables change what it installs:

| Variable | Description | Default |
| --- | --- | --- |
| `VERSION` | Release tag to install, such as `v2.0.0` | The latest release |
| `PREFIX` | Directory for archive installs | `$HOME/go/bin`, or the directory of the previous archive install |
| `METHOD` | `auto`, `package`, or `archive` | `auto` |

For example, to install the archive into `~/.local/bin`:

```bash
METHOD=archive PREFIX="$HOME/.local/bin" sh "$tmp"
```

Releases before v2.0.0 publish their packages under different names, so the script installs them from the archive. `METHOD=auto` falls back to the archive, and `METHOD=package` fails.

## Update and uninstall

| Action | Command |
| --- | --- |
| Update to the latest release | `sh "$tmp" update` |
| Remove speedtest-go | `sh "$tmp" uninstall` |
