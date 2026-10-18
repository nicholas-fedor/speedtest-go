---
title: Troubleshooting
description: What speedtest-go's exit codes mean, and how to fix the errors it reports
weight: 6
---

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The command succeeded |
| `1` | The command failed, such as a network error or a `--server` ID that speedtest.net does not list. Errors go to standard error |
| `2` | A setting is invalid, or a settings file cannot be read or decoded |
| `130` | The run was interrupted with Ctrl-C, and no report was written |

A second Ctrl-C ends a run that does not stop after the first.

## Invalid settings

An invalid setting stops the command before it runs, exits with code 2, and names where the value came from: a flag, an environment variable, or a settings file key. Every invalid setting is reported at once.

| Error | Fix |
| --- | --- |
| An unknown `--ping-mode`, `--protocol`, or `--unit` value | Use one of the values the [CLI Reference](/usage/cli-reference/) lists |
| A negative `--thread` | Use 0 for the default, or a positive number |
| More than one of `--json`, `--jsonl`, and `--unix` | Choose one output format |
| `--city` and `--location` both set | Use one of them |
| `--city`, `--location`, or `--search` with `--server` or `--custom-url` | Leave the location settings out, since the servers are already chosen |
| `--filter-cc` with `--server` or `--custom-url` | Leave the filter out, since the servers are already chosen |
| `--proxy` with `--protocol tcp` | Use HTTP transfers, since TCP connections cannot go through an HTTP proxy |
| A `--source` that is neither an IP address nor a network interface | Use an address of this host or an interface name |
| A `--dl-size` or `--ul-size` outside its table | Use a size from [Transfer sizes](/usage/protocols/#transfer-sizes) |
| An unknown key, a repeated key, or a value of the wrong type in the settings file | Fix the named key. See [Configuration](/usage/configuration/#settings) for the keys |
| A settings file named by `--config` or `SPEEDTEST_GO_CONFIG` that does not exist | Fix the path, or leave the setting out to use the default file |

## Server errors

| Error | Cause |
| --- | --- |
| A `--server` ID is not listed | speedtest.net does not list that server near you. Run `speedtest-go list` to see the available IDs |
| No servers in the selected countries | `--filter-cc` left no servers. Check the codes, or remove the filter |
| Cannot determine your country for `--filter-cc auto` | speedtest.net did not report your country. Name the countries instead, or set `--city` |

## Network errors

| Symptom | Cause |
| --- | --- |
| `--ping-mode icmp` fails | ICMP needs raw sockets, so it needs root or `CAP_NET_RAW` on Linux. In a container, use `http` or `tcp` |
| Binding to an interface fails on Linux | Kernels before 5.7 allow it only with the `CAP_NET_RAW` capability |
| A nonzero `--dscp` is rejected | DSCP marking works only on Linux, macOS, and the BSDs |
| Packet loss shows N/A | The server received none of the test packets. Packet loss is measured over UDP, which does not go through a proxy, and which a firewall may block |

## Debug output

`--debug` writes the client's debug messages to standard error, and shows plain lines instead of the live progress:

```bash
speedtest-go --debug
```
