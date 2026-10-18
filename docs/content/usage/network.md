---
title: Network paths
description: Send speedtest-go's traffic from a chosen address or interface, mark it with DSCP, or route it through a proxy
weight: 3
---

| To | Use |
| --- | --- |
| Test one of several network connections | `--source` |
| Send DNS lookups from the same source | `--dns-bind-source` |
| Measure a quality of service class | `--dscp` |
| Route the test through a proxy | `--proxy` |

## Source address or interface

On a host with several network connections, `--source` picks the one to test. It takes an IP address of this host or a network interface name.

```bash
speedtest-go --source 192.168.1.101
speedtest-go --source eth0
```

How an interface name is used depends on the platform:

| Platform | Behavior |
| --- | --- |
| Linux | The API, ping, download, upload, and packet loss connections are bound to the interface with `SO_BINDTODEVICE`, so their traffic leaves through it whatever the routing table prefers. Each one uses the interface's address of the matching IP version. |
| Other platforms | Those connections use the interface's IPv4 address, or its first IPv6 address that is not link-local, as their source. |

Things to know:

- Linux kernels before 5.7 allow binding to an interface only with the `CAP_NET_RAW` capability.
- DNS lookups use the system resolver, unless `--dns-bind-source` is set, which sends them from the source as well.
- These are invalid settings:
  - An interface without an IPv4 address, or an IPv6 address that is not link-local
  - A value that is neither an IP address nor an interface

## DSCP marking

On a network that applies quality of service, `--dscp` marks the API, ping, download, upload, and packet loss connections with a Differentiated Services code point from 0 to 63. The test then measures the treatment that class of traffic receives.

| Code point | Class |
| --- | --- |
| 46 | Expedited Forwarding, often used for voice |
| 34 | Assured Forwarding 41 |

```bash
speedtest-go --dscp 46
```

Things to know:

- The code point is set on IPv4 and IPv6 sockets on Linux, macOS, and the BSDs, and on DNS sockets too when `--dns-bind-source` is set.
- Other platforms cannot set it, so a nonzero value there is an invalid setting, as is a value outside 0 to 63.
- A connection that cannot be marked fails, rather than running unmarked.

## Proxy

`--proxy` sends the test through an HTTP, HTTPS, or SOCKS proxy:

```bash
speedtest-go --proxy socks5://127.0.0.1:7890
```

TCP connections cannot go through an HTTP proxy, so `--protocol tcp` together with `--proxy` is an invalid setting. Packet loss is measured over UDP, which does not go through the proxy.
