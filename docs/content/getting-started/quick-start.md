---
title: Quick Start
description: What speedtest-go measures, and how to install it, run a first test, and choose its servers and output
weight: 1
---

**speedtest-go** measures your internet connection against [speedtest.net](https://www.speedtest.net/) servers. It comes as a command-line tool and as a [Go library](/go-library/).

It measures:

- Latency and jitter, over HTTP, TCP, or ICMP
- Download and upload rates, over HTTP or TCP
- Packet loss

It is a fork of [showwin/speedtest-go](https://github.com/showwin/speedtest-go), updated to the latest Go version and dependencies, with a redesigned v2 library API.

## Install

On Linux and macOS, the install script downloads the latest release, verifies its checksum, and installs it:

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/scripts/install.sh -o "$tmp" && sh "$tmp"
```

[Installation](/installation/) covers the script's options, updating, and uninstalling, and the other methods: binaries for Linux, macOS, and Windows, Go, Docker, and Kubernetes.

## Run a test

```bash
speedtest-go
```

speedtest-go tests the server near you with the lowest latency. [How a Test Works](../how-it-works/) explains each step.

## Choose the output

| Output | Flag | Use it for |
| --- | --- | --- |
| Live progress | none | Watching a test in a terminal |
| Plain lines | `--unix` | Logs, and terminals that cannot redraw |
| One JSON document | `--json` | Scripts that read the whole result |
| One JSON line per server | `--jsonl` | Scripts that process servers one at a time |

```bash
speedtest-go --json > result.json
```

[Reading the Results](../results/) explains each output.

## Choose the servers

`list` shows the servers near you, with their IDs, distances, and latencies:

```bash
speedtest-go list
```

Pass one or more IDs to test them:

```bash
speedtest-go --server 6691 --server 6087
```

[Choosing servers](/usage/servers/) covers testing near another location, filtering by country, and spreading a test across several servers.

## Next steps

| To | See |
| --- | --- |
| Set defaults, so a plain `speedtest-go` runs the way you want | [Configuration](/usage/configuration/) |
| Look up every flag | [CLI Reference](/usage/cli-reference/) |
| Understand an error or an exit code | [Troubleshooting](/usage/troubleshooting/) |
