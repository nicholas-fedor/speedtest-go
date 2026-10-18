---
title: Test protocols
description: Choose how speedtest-go measures latency and moves data, and tune its connections and transfer sizes
weight: 4
---

| To | Use |
| --- | --- |
| Measure latency with TCP or ICMP | `--ping-mode` |
| Download and upload over plain TCP | `--protocol tcp` |
| Change the size of each transfer | `--dl-size`, `--ul-size` |
| Change the number of connections | `--thread` |
| Test with few resources | `--saving-mode` |
| Skip the download or upload test | `--no-download`, `--no-upload` |

## Ping methods

`--ping-mode` chooses how latency and jitter are measured:

| Mode | Measures | Notes |
| --- | --- | --- |
| `http` | HTTP requests to the server | The default |
| `tcp` | The speedtest.net TCP protocol's ping, normally on port 8080 | |
| `icmp` | ICMP echo requests | Needs raw sockets, so it needs root or `CAP_NET_RAW` on Linux |

```bash
speedtest-go --ping-mode tcp
```

## TCP transfers

By default, downloads and uploads use HTTP. With `--protocol tcp`, they use the plain TCP protocol that speedtest.net servers also serve, normally on port 8080, which avoids HTTP overhead.

```bash
speedtest-go --protocol tcp
```

Things to know:

- The ping method is chosen separately, with `--ping-mode`.
- The packet loss test works the same way with either protocol.
- TCP connections cannot go through an HTTP proxy, so `--protocol tcp` together with `--proxy` is an invalid setting.

## Transfer sizes

Each download and upload request moves one block of data. Larger blocks spend less of the test on starting requests, which can matter on fast links, while smaller blocks suit slow ones.

| Flag | Sizes | Default |
| --- | --- | --- |
| `--dl-size` | 350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000 | 1000 |
| `--ul-size` | 100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000 | 1000, or 800 with `--multi` |

What a size means:

| Protocol | Download size | Upload size |
| --- | --- | --- |
| HTTP | The side of the requested test image, so 2000 requests a 2000x2000 image | kB |
| TCP | kB | kB |

```bash
speedtest-go --dl-size 2000 --ul-size 1500
```

Any other value is an invalid setting.

## Connections

`--thread` sets the number of concurrent connections for each download and upload. Without it, downloads use one connection per CPU, and uploads up to 8. A negative value is an invalid setting.

```bash
speedtest-go --thread 4
```

## Saving mode

`--saving-mode` tests with a single connection and few resources, at the cost of accuracy, especially above 30 Mbps. To get more accurate results with it, run several tests and average them.

In the [saving mode results](/usage/results/saving-mode/), measured with an earlier version, saving mode used less than 10 MB of memory, against about 85 to 110 MB without it.
