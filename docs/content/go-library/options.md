---
title: Client Options
description: The options speedtest.New takes, for the transport, the network path, the tests, the server list, and logging
weight: 2
---

`speedtest.New` builds a client from options. It validates every option, and its error wraps `speedtest.ErrInvalidOption` once for each invalid one.

```go
tokyo, err := speedtest.GetLocation("tokyo")
if err != nil {
	log.Fatal(err)
}

client, err := speedtest.New(
	speedtest.WithLocation(tokyo),
	speedtest.WithProtocol(speedtest.TCPTest),
	speedtest.WithPingMode(speedtest.TCP),
	speedtest.WithConnections(4),
)
if err != nil {
	log.Fatal(err)
}
```

The client owns its HTTP client, dialers, and transport, so it never changes `http.DefaultClient` or `net.DefaultResolver`.

## Transport

| Option | Sets |
| --- | --- |
| `WithHTTPClient(client)` | The HTTP client for requests. It is copied, so yours is never changed. A client with its own `Transport` keeps it, and the user agent, proxy, source, DSCP, and dialer controls then do not apply to HTTP requests |
| `WithUserAgent(agent)` | The `User-Agent` header, in place of `DefaultUserAgent` |
| `WithProxy(url)` | A proxy for HTTP requests, in place of the one the environment sets |
| `WithBaseURL(url)` | The speedtest API base URL, in place of `DefaultBaseURL` |

## Network path

| Option | Sets |
| --- | --- |
| `WithSource(source)` | Where connections leave from: an IP address, a network interface name, or a hostname. On Linux, an interface name binds each socket to the interface |
| `WithDNSBindSource()` | DNS queries leave the same way as the other connections, through the pure Go resolver |
| `WithDSCP(code)` | A Differentiated Services code point from 0 to 63 on every socket. On a platform that cannot mark sockets, a nonzero value makes `New` fail with an error wrapping `ErrDSCPUnsupported` |
| `WithDialerControl(fn)` | A function that runs on each socket before it connects, after any source binding and DSCP marking. Each call adds another |

## Tests

| Option | Sets | Default |
| --- | --- | --- |
| `WithPingMode(mode)` | How latency is measured: `speedtest.HTTP`, `speedtest.TCP`, or `speedtest.ICMP` | `HTTP` |
| `WithProtocol(mode)` | The download and upload protocol: `speedtest.HTTPTest` or `speedtest.TCPTest` | `HTTPTest` |
| `WithConnections(n)` | The connections each transfer test uses | One per CPU for downloads, up to 8 for uploads |
| `WithTransferSizes(download, upload)` | The block size of each request, from the size tables. Over HTTP the download size is the side of the requested image, and over TCP it is in kB. The upload size is in kB. 0 keeps a default | 1000, and 800 for multi-server uploads |
| `WithTestDuration(d)` | How long each download and upload test runs | 15 seconds |

[Transfer sizes](/usage/protocols/#transfer-sizes) lists the sizes each table allows.

## Server list

| Option | Sets |
| --- | --- |
| `WithLocation(location)` | Ranks servers by their distance from a location, in place of the one speedtest.net reports |
| `WithSearch(keyword)` | Keeps only servers that match a keyword, such as a city or sponsor name |

A location comes from a city label, with `speedtest.GetLocation("tokyo")`, or from coordinates, with `speedtest.ParseLocation("osaka", "34.6952,135.5006")`. `speedtest.Cities()` lists the city labels.

## Logging

| Option | Sets |
| --- | --- |
| `WithLogger(logger)` | Where the client's debug messages go. Without it, they are discarded |

A `Logger` has one method, `Debug(ctx, msg, args...)`, whose arguments alternate keys and values in the style of `log/slog`. The [migration guide](../migration/#logging) shows a `*slog.Logger` adapter.
