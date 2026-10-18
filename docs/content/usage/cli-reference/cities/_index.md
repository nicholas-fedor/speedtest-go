---
title: "Cities"
description: "Display a list of predefined city labels that can be used for location-based server selection."
type: docs
---

Display a list of predefined city labels that can be used for location-based server selection.

## Usage

```bash
speedtest-go cities
```

## Global options

Every command accepts these flags. See [Configuration](/usage/configuration/) for how they combine with environment variables and the settings file.

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
