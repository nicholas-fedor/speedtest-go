---
title: Configuration
description: Set speedtest-go's defaults with a settings file or environment variables, and how they combine with flags
weight: 1
---

Every flag except `--help` can also be set with an environment variable, and every flag except `--help` and `--config` in a settings file.

## Precedence

Each setting comes from the first of these that sets it:

1. A flag on the command line
2. Its `SPEEDTEST_GO_` environment variable
3. The settings file
4. The flag's default

## Settings file

The settings file is YAML. Its default location depends on the platform:

| Platform | Path |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/speedtest-go/config.yaml`, which is `~/.config/speedtest-go/config.yaml` by default |
| macOS | `~/Library/Application Support/speedtest-go/config.yaml` |
| Windows | `%LOCALAPPDATA%\speedtest-go\config.yaml` |

`--config` or `SPEEDTEST_GO_CONFIG` names another file.

| File | When it is missing |
| --- | --- |
| The default file | The defaults apply |
| A file named by `--config` or `SPEEDTEST_GO_CONFIG` | An error |

Its keys are the flag names in camelCase, such as `pingMode` for `--ping-mode`:

```yaml
pingMode: tcp
unit: binary-bits
server: [6691, 6087]
filterCC: [gb, ie]
maskISP: true
```

The file is read strictly, so a typo never passes silently. These are errors:

- An unknown key
- A repeated key
- A value of the wrong type
- More than one YAML document

## Environment variables

A setting's environment variable is `SPEEDTEST_GO_` followed by the flag name in upper case, with hyphens as underscores.

| Value | Format |
| --- | --- |
| Boolean | `true` or `false` |
| List | Entries separated by commas |

```bash
SPEEDTEST_GO_PING_MODE=tcp SPEEDTEST_GO_DNS_BIND_SOURCE=true speedtest-go
SPEEDTEST_GO_SERVER=6691,6087 speedtest-go
```

## Settings

Settings that apply to the speed test are ignored by `list`, `cities`, and `version`.

| Flag | Environment variable | Settings file key | Applies to |
| --- | --- | --- | --- |
| `--base-url` | `SPEEDTEST_GO_BASE_URL` | `baseURL` | Every command |
| `--city` | `SPEEDTEST_GO_CITY` | `city` | Every command |
| `--custom-url` | `SPEEDTEST_GO_CUSTOM_URL` | `customURL` | Speed test |
| `--debug` | `SPEEDTEST_GO_DEBUG` | `debug` | Every command |
| `--dl-size` | `SPEEDTEST_GO_DL_SIZE` | `dlSize` | Speed test |
| `--dns-bind-source` | `SPEEDTEST_GO_DNS_BIND_SOURCE` | `dnsBindSource` | Every command |
| `--dscp` | `SPEEDTEST_GO_DSCP` | `dscp` | Every command |
| `--filter-cc` | `SPEEDTEST_GO_FILTER_CC` | `filterCC` | Every command |
| `--json` | `SPEEDTEST_GO_JSON` | `json` | Speed test |
| `--jsonl` | `SPEEDTEST_GO_JSONL` | `jsonl` | Speed test |
| `--location` | `SPEEDTEST_GO_LOCATION` | `location` | Every command |
| `--mask-isp` | `SPEEDTEST_GO_MASK_ISP` | `maskISP` | Every command |
| `--multi` | `SPEEDTEST_GO_MULTI` | `multi` | Speed test |
| `--no-download` | `SPEEDTEST_GO_NO_DOWNLOAD` | `noDownload` | Speed test |
| `--no-upload` | `SPEEDTEST_GO_NO_UPLOAD` | `noUpload` | Speed test |
| `--ping-mode` | `SPEEDTEST_GO_PING_MODE` | `pingMode` | Speed test |
| `--protocol` | `SPEEDTEST_GO_PROTOCOL` | `protocol` | Speed test |
| `--proxy` | `SPEEDTEST_GO_PROXY` | `proxy` | Every command |
| `--saving-mode` | `SPEEDTEST_GO_SAVING_MODE` | `savingMode` | Speed test |
| `--search` | `SPEEDTEST_GO_SEARCH` | `search` | Every command |
| `--server` | `SPEEDTEST_GO_SERVER` | `server` | Speed test |
| `--source` | `SPEEDTEST_GO_SOURCE` | `source` | Every command |
| `--thread` | `SPEEDTEST_GO_THREAD` | `thread` | Speed test |
| `--ua` | `SPEEDTEST_GO_UA` | `ua` | Every command |
| `--ul-size` | `SPEEDTEST_GO_UL_SIZE` | `ulSize` | Speed test |
| `--unit` | `SPEEDTEST_GO_UNIT` | `unit` | Speed test |
| `--unix` | `SPEEDTEST_GO_UNIX` | `unix` | Speed test |

`--config` and its variable `SPEEDTEST_GO_CONFIG` name the settings file, so the file itself has no key for them.

## Invalid settings

An invalid setting stops the command before it runs, and it exits with code 2. The error names where the value came from:

| Source | Named as |
| --- | --- |
| Flag | `--thread` |
| Environment variable | `SPEEDTEST_GO_THREAD` |
| Settings file | The file's path and the key, such as `config.yaml: thread` |

Every invalid setting is reported at once, not just the first.
