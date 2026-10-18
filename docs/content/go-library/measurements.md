---
title: Measurements
description: Find servers, run the speedtest client's tests, read their results, and write them as JSON
weight: 3
---

## Finding servers

| Call | Returns |
| --- | --- |
| `client.FetchUserInfo(ctx)` | Your IP address, ISP, coordinates, and country, as speedtest.net reports them |
| `client.FetchServers(ctx)` | The servers near you, or near the [location option](../options/#server-list) |
| `client.FetchServerByID(ctx, id)` | One server by its ID, or `ErrServerNotFound` |
| `client.CustomServer(host)` | A server that speedtest.net does not need to list |
| `servers.FindServer(ids)` | The servers with those IDs. With no IDs, or none that match, the server with the lowest latency |
| `servers.FilterCountry(codes...)` | The servers in those countries, by two-letter code |
| `servers.Available()` | The servers with a measured latency |

## Tests

Each test takes a context and a server, and returns its own result. Ending the context ends the test early, which makes it fail.

| Method | Measures | Returns |
| --- | --- | --- |
| `Ping(ctx, server, opts...)` | Latency, with the client's ping mode | `PingStats` |
| `Download(ctx, server, opts...)` | The download rate | `Transfer` |
| `Upload(ctx, server, opts...)` | The upload rate | `Transfer` |
| `DownloadMulti(ctx, main, pool, opts...)` | The download rate across several servers at once. The main server gets the main share of the connections, and the other available servers in the pool share the rest | `Transfer` |
| `UploadMulti(ctx, main, pool, opts...)` | The upload rate across several servers, shared like `DownloadMulti` | `Transfer` |
| `PacketLoss(ctx, server)` | Packet loss, until the context ends. See [Packet Loss](../packet-loss/) | `PacketLossStats` |

### Options for one test

| Option | For | Sets | Default |
| --- | --- | --- | --- |
| `WithPingCount(n)` | `Ping` | The number of round trips | 10 |
| `WithPingInterval(d)` | `Ping` | The wait after each round trip | 200 milliseconds |
| `WithLatencyCallback(fn)` | `Ping` | A function that receives each round trip as it is measured | None |
| `WithProgressCallback(fn)` | Transfers | A function that receives the rate estimate, the elapsed time, and the estimated time left while the test runs | None |

## Results

`PingStats`:

| Field | Holds |
| --- | --- |
| `Latency` | The mean round trip |
| `Jitter` | The standard deviation of the round trips |
| `Min`, `Max` | The shortest and longest round trips |
| `Latencies` | Every round trip, in the order they were sent |
| `Duration` | How long the measurement took |

`Transfer`:

| Field | Holds |
| --- | --- |
| `Rate` | Bytes per second over the test's capture window, or -1 when no data was counted and more than a tenth of the requests failed |
| `Bytes` | The bytes downloaded, or the upload bytes the servers confirmed |
| `Sent` | The upload bytes sent, confirmed or not. 0 for downloads |
| `Window` | The capture window the rate is measured over |
| `Duration` | How long the test took |
| `Requests`, `Failures` | How many requests the test sent, and how many failed |
| `Connections` | The bytes, requests, failures, and active time of each connection |

A `ByteRate` such as `Transfer.Rate` prints in Mbps. `Format` chooses another unit, such as `rate.Format(speedtest.UnitTypeBinaryBits)`, and `Mbps` and `Gbps` return numbers.

## JSON

A `speedtest.Result` collects one server's results. `speedtest.JSON` and `speedtest.JSONL` write results in the CLI's JSON format, described in [Reading the Results](/getting-started/results/#json-reports):

```go
report, err := speedtest.JSON(user, []speedtest.Result{{
	Server:   server,
	Ping:     &ping,
	Download: &download,
	Upload:   &upload,
}})
```

## Errors

Check errors with `errors.Is` and `errors.As`:

| Error | Means |
| --- | --- |
| `ErrInvalidOption` | `New` rejected an option |
| `ErrServerNotFound` | No server is available, or none has the requested ID |
| `*StatusError`, wrapping `ErrUnexpectedStatus` | A server answered with an HTTP status outside 2xx |
| `ErrUnsupported` | A server does not serve the packet loss protocol |
| `context.Canceled`, `context.DeadlineExceeded` | The test's context ended before it finished |
