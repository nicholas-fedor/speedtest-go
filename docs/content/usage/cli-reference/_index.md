---
title: "CLI Reference"
description: "Every speedtest-go command and flag, generated from the program itself"
weight: 7
type: docs
---

Complete command reference for speedtest-go, generated from the program itself.

A command-line tool to test internet download and upload speeds using speedtest.net servers.

## Usage

```bash
speedtest-go
```

Without a command, speedtest-go runs a speed test with the options below.

## Commands

| Command | Description |
| --- | --- |
| [cities](/usage/cli-reference/cities/) | List predefined city labels |
| [list](/usage/cli-reference/list/) | List available speedtest servers |
| [version](/usage/cli-reference/version/) | Print version information |

## Options

These flags apply to the speed test.

| Flag | Short | Default | Type | Description |
| --- | --- | --- | --- | --- |
| `--custom-url` |  | `""` | string | Specify the url of the server instead of fetching from speedtest.net. |
| `--dl-size` |  | `0` | int | Set the download block size, from 350 to 4000 (default 1000). |
| `--json` |  | `false` | bool | Output results in json format. |
| `--jsonl` |  | `false` | bool | Output results in jsonl format (one json object per line). |
| `--multi` | `-m` | `false` | bool | Enable multi-server mode. |
| `--no-download` |  | `false` | bool | Disable download test. |
| `--no-upload` |  | `false` | bool | Disable upload test. |
| `--ping-mode` |  | `http` | string | Select a method for Ping (support icmp/tcp/http). |
| `--protocol` |  | `http` | string | Select the download and upload protocol (http or tcp). |
| `--saving-mode` |  | `false` | bool | Test with few resources, though low accuracy (especially > 30Mbps). |
| `--server` | `-s` | `[]` | intSlice | Select server id to run speedtest. |
| `--thread` | `-t` | `0` | int | Set the number of concurrent connections. |
| `--ul-size` |  | `0` | int | Set the upload block size in kB, from 100 to 4000 (default 1000, or 800 with --multi). |
| `--unit` | `-u` | `""` | string | Set human-readable and auto-scaled rate units for output (options: decimal-bits/decimal-bytes/binary-bits/binary-bytes). |
| `--unix` |  | `false` | bool | Output results in unix like format. |

## Global options

Every command accepts these flags.

| Flag | Short | Default | Type | Description |
| --- | --- | --- | --- | --- |
| `--base-url` |  | `""` | string | Set the speedtest API base URL (default https://www.speedtest.net). |
| `--city` |  | `""` | string | Change the location with a predefined city label. |
| `--config` |  | `""` | string | settings file (default speedtest-go/config.yaml in the user config directory) |
| `--debug` |  | `false` | bool | Enable debug mode. |
| `--dns-bind-source` |  | `false` | bool | DNS request binding source (experimental). |
| `--dscp` |  | `0` | int | Mark connections with a DSCP code point from 0 to 63 (Linux, macOS and BSD). |
| `--filter-cc` |  | `[]` | stringSlice | Only use servers in these countries (two-letter codes, or auto for your own). |
| `--location` |  | `""` | string | Change the location with a precise coordinate (format: lat,lon). |
| `--mask-isp` |  | `false` | bool | Mask your IP address, ISP, coordinates, and distances to servers in all output. |
| `--proxy` |  | `""` | string | Set a proxy(http[s] or socks) for the speedtest. |
| `--search` |  | `""` | string | Fuzzy search servers by a keyword. |
| `--source` |  | `""` | string | Send from a source IP address or network interface name. |
| `--ua` |  | `""` | string | Set the user-agent header for the speedtest. |

Every flag except `--help` can also be set with an environment variable, and every flag except `--help` and `--config` in the settings file. See [Configuration](/usage/configuration/) for the names and the order in which they apply.
