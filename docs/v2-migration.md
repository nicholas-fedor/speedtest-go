# Migrating to v2

This guide covers the changes a program that uses the speedtest-go library needs when it moves from v1.8.4 to v2. The CLI changes are listed at the end.

In short, v2 moves every measurement onto the client. A client is built from validated options, each test is a method that takes a context and a server and returns its result, and `Server` holds only the details speedtest.net lists. No results are stored on the server or the client, so nothing needs resetting between tests.

## Module path

The module path has a `/v2` suffix:

```bash
go get github.com/nicholas-fedor/speedtest-go/v2
```

```go
import "github.com/nicholas-fedor/speedtest-go/v2/speedtest"
```

The `speedtest/transport` package is now internal and can no longer be imported. See [Packet loss](#packet-loss) for its replacements.

## Before and after

A v1 program that tested the server with the lowest latency:

```go
client := speedtest.New()

serverList, _ := client.FetchServers()
targets, _ := serverList.FindServer([]int{})

for _, server := range targets {
    _ = server.PingTest(nil)
    _ = server.DownloadTest()
    _ = server.UploadTest()

    fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", server.Latency, server.DLSpeed, server.ULSpeed)
    server.Context.Reset()
}
```

The same program in v2:

```go
client, err := speedtest.New()
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

serverList, err := client.FetchServers(ctx)
if err != nil {
    log.Fatal(err)
}

targets, err := serverList.FindServer([]int{})
if err != nil {
    log.Fatal(err)
}

for _, server := range targets {
    ping, err := client.Ping(ctx, server)
    if err != nil {
        log.Fatal(err)
    }

    download, err := client.Download(ctx, server)
    if err != nil {
        log.Fatal(err)
    }

    upload, err := client.Upload(ctx, server)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", ping.Latency, download.Rate, upload.Rate)
}
```

## Creating a client

`Speedtest` is renamed `Client`. `New` takes options and returns `(*Client, error)`. It validates every option and returns an error wrapping `ErrInvalidOption` that lists each invalid one. The client owns its HTTP client, dialers, and transport, so it never changes `http.DefaultClient` or `net.DefaultResolver`.

`UserConfig`, `WithUserConfig`, `NewUserConfig`, and `WithDoer` are removed. Each setting has its own option. The `RoundTrip` method is removed as well, since the client's own transport sets the user agent.

| v1 | v2 |
| --- | --- |
| `UserConfig.T` | `WithHTTPClient(&http.Client{Transport: t})`. A client with its own `Transport` keeps it, and the user agent, proxy, source, DSCP, and dialer control options then do not apply to HTTP requests. |
| `WithDoer(client)` | `WithHTTPClient(client)`, with the same caveat. The client is copied, so yours is never modified. |
| `UserConfig.UserAgent` | `WithUserAgent(userAgent)` |
| `UserConfig.Proxy` | `WithProxy(proxy)` |
| `UserConfig.Source` | `WithSource(source)`, which takes an IP address, a hostname, or a network interface name |
| `UserConfig.DNSBindSource` | `WithDNSBindSource()` |
| `UserConfig.DialerControl` | `WithDialerControl(control)` |
| `UserConfig.Debug` | `WithLogger(logger)`. See [Logging](#logging). |
| `UserConfig.PingMode` | `WithPingMode(mode)` |
| `UserConfig.SavingMode` | `WithConnections(1)` |
| `UserConfig.MaxConnections`, `SetNThread(n)` | `WithConnections(n)`, which must be at least 1. Leave it out for the default. |
| `SetCaptureTime(duration)` | `WithTestDuration(duration)` |
| `SetRateCaptureFrequency(duration)` | Removed, with no replacement. |
| `UserConfig.CityFlag` | `WithLocation(location)`, with `location` from `GetLocation(city)` |
| `UserConfig.LocationFlag` | `WithLocation(location)`, with `location` from `ParseLocation(name, "lat,lon")` |
| `UserConfig.Location` | `WithLocation(location)` |
| `UserConfig.Keyword` | `WithSearch(keyword)` |

v2 also adds `WithBaseURL`, `WithDSCP`, `WithProtocol` for TCP download and upload tests, and `WithTransferSizes`.

```go
tokyo, err := speedtest.GetLocation("tokyo")
if err != nil {
    log.Fatal(err)
}

client, err := speedtest.New(
    speedtest.WithLocation(tokyo),
    speedtest.WithConnections(4),
    speedtest.WithPingMode(speedtest.TCP),
)
if err != nil {
    log.Fatal(err)
}
```

### Logging

`Debug`, `NewDebug`, and `UserConfig.Debug` are removed, and the library no longer writes to stdout. A client sends its debug messages to the `Logger` that `WithLogger` sets, and discards them without one. `Logger` has one method, `Debug(ctx, msg, args...)`, whose arguments alternate keys and values in the style of `log/slog`, so a `*slog.Logger` fits behind a short adapter:

```go
type slogLogger struct {
    logger *slog.Logger
}

func (l slogLogger) Debug(ctx context.Context, msg string, args ...any) {
    l.logger.DebugContext(ctx, msg, args...)
}
```

```go
client, err := speedtest.New(speedtest.WithLogger(slogLogger{logger: slog.Default()}))
```

## Package functions and client state

The package-level functions that used a hidden default client are removed. Call the method on your own client instead. Every method that sends a request takes a context, so the `Context` variants are removed as well:

| v1 | v2 |
| --- | --- |
| `FetchUserInfo()`, `FetchUserInfoContext(ctx)`, and the client methods of the same names | `client.FetchUserInfo(ctx)` |
| `FetchServers()`, `FetchServerListContext(ctx)`, and the client methods of the same names | `client.FetchServers(ctx)` |
| `FetchServerByID(id)`, `client.FetchServerByID(id)`, and `client.FetchServerByIDContext(ctx, id)` | `client.FetchServerByID(ctx, id)` |
| `CustomServer(host)` | `client.CustomServer(host)`, which sends no request and takes no context |

`FetchServers` returns the context's error when the context ends before its latency probes finish, instead of a list whose probes all failed.

The client no longer keeps the caller's details. `GetUser` and `SetUser` are removed: keep the `*User` that `FetchUserInfo` returns and pass it to `JSON` or `JSONL`.

## Measurements

The measurement methods on `Server` are removed, along with `Server.Context`. Each one is a client method that takes a context and returns its result:

| v1 | v2 |
| --- | --- |
| `server.PingTest(callback)`, `server.PingTestContext(ctx, callback)` | `client.Ping(ctx, server, speedtest.WithLatencyCallback(callback))`, which returns `PingStats` |
| `server.HTTPPing`, `server.TCPPing`, `server.ICMPPing` | `client.Ping` with the client's `WithPingMode`. `WithPingCount` and `WithPingInterval` replace the echo count and frequency, and `PingStats.Latencies` holds each round trip. A different mode needs a second client. |
| `server.DownloadTest()`, `server.DownloadTestContext(ctx)` | `client.Download(ctx, server)`, which returns `Transfer` |
| `server.UploadTest()`, `server.UploadTestContext(ctx)` | `client.Upload(ctx, server)`, which returns `Transfer` |
| `server.MultiDownloadTestContext(ctx, servers)` | `client.DownloadMulti(ctx, server, servers)` |
| `server.MultiUploadTestContext(ctx, servers)` | `client.UploadMulti(ctx, server, servers)` |
| `server.TestAll()` | `client.Ping`, `client.Download`, and `client.Upload` in turn |
| `server.CheckResultValid()` | Removed. It reported whether neither speed was more than 100 times the other, which you can check on the two `Transfer` rates. |

`Ping`, `Download`, and `Upload` end early when their context ends, and then return its error.

### Where the results went

The result fields on `Server` are removed. Read them from the values the methods return:

| v1 | v2 |
| --- | --- |
| `server.Latency` after `PingTest` | `ping.Latency` |
| `server.MaxLatency`, `server.MinLatency`, `server.Jitter` | `ping.Max`, `ping.Min`, `ping.Jitter` |
| `server.DLSpeed`, `server.ULSpeed` | `download.Rate`, `upload.Rate` |
| `server.TestDuration` | `ping.Duration`, `download.Duration`, `upload.Duration` |
| `server.PacketLoss` | `client.PacketLoss(ctx, server)` |

`Server.Latency` still exists, but it now holds only the quick probe that `FetchServers` sends to rank servers, or `PingTimeout` when the probe failed. Code that only deletes its `PingTest` call still compiles and reads the probe instead of a measured latency, so read `PingStats.Latency` instead.

### Rates, totals, and callbacks

v1 counted every test on the client's single data manager, which added up until `Reset`. In v2 each download or upload test measures on its own:

| v1 | v2 |
| --- | --- |
| `client.SetCallbackDownload(callback)`, `client.SetCallbackUpload(callback)` | `speedtest.WithProgressCallback(callback)`, passed to the one test it applies to. The callback receives a `TransferProgress` with the rate, the elapsed time, and the estimated time left. |
| `client.GetTotalDownload()`, `client.GetTotalUpload()` | `transfer.Bytes` |
| `client.Reset()` | Not needed. |
| `client.Snapshots()` | Removed. `transfer.Connections` reports each connection's bytes, requests, failures, and active time. |
| `client.Wait()`, `GetAvgDownloadRate`, `GetEWMADownloadRate`, and the other `Manager` methods | Removed. |

`Manager`, `Chunk`, `DataManager`, `NewDataManager`, `DataChunk`, `DataType`, `TestDirection`, `Snapshot`, and `Snapshots` are removed with no replacement. They are internal to the library now.

`Transfer.Rate` is the bytes per second over the test's capture window, measured from the bytes counted. Upload tests count only the bytes a server confirmed with a successful response, and `Transfer.ConfirmationRatio` reports the share of the bytes sent that were confirmed. The progress callback receives a smoothed live estimate of the rate while the test runs.

```go
download, err := client.Download(ctx, server, speedtest.WithProgressCallback(func(p speedtest.TransferProgress) {
    fmt.Println("Download:", p.Rate, "time left:", p.Remaining.Round(time.Second))
}))
if err != nil {
    log.Fatal(err)
}

for _, connection := range download.Connections {
    fmt.Println(connection.ServerID, connection.Rate(), connection.Failures)
}
```

## JSON output

`client.JSON(servers)` and `client.JSONL(server)` are package functions that take the caller's details and the results:

| v1 | v2 |
| --- | --- |
| `client.JSON(servers)` | `speedtest.JSON(user, results)`, with `results` a `[]speedtest.Result` |
| `client.JSONL(server)` | `speedtest.JSONL(user, result)`, with `result` a `speedtest.Result` |

A `Result` collects one server's measurements. Leave a measurement nil when it was not taken:

```go
result := speedtest.Result{
    Server:     server,
    Ping:       &ping,
    Download:   &download,
    Upload:     &upload,
    PacketLoss: nil,
}

report, err := speedtest.JSON(user, []speedtest.Result{result})
```

The JSON keeps the v1 keys in the same order. Each server gains `cc`, its country code, and `userInfo` gains `country`. `Result.Report` returns the `ResultReport` that each server's entry is built from.

## Packet loss

`client.PacketLoss(ctx, server)` measures the packet loss to one server until its context ends, through the same source and dialer settings as the client's other connections:

```go
lossCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

packetLoss, err := client.PacketLoss(lossCtx, server)
if err != nil {
    log.Println(err)
}

fmt.Println(packetLoss)
```

The packet loss analyzer remains, and `client.NewPacketLossAnalyzer(options)` builds one with the client's network setup. Its results use the new `speedtest.PacketLossStats` type:

| v1 | v2 |
| --- | --- |
| `transport.PLoss` | `speedtest.PacketLossStats`, with the same `Sent`, `Dup`, and `Max` fields and the same `String`, `Loss`, and `LossPercent` methods |
| `func(packetLoss *transport.PLoss)` callbacks of `Run` and `RunWithContext` | `func(packetLoss *speedtest.PacketLossStats)` |
| `*transport.PLoss` from `RunMulti` and `RunMultiWithContext` | `*speedtest.PacketLossStats` |
| `transport.ErrUnsupported` | `speedtest.ErrUnsupported` |
| `transport.Client`, `transport.PacketLossSender`, and the other `transport` errors | Removed, with no replacement. |

`Loss` and `LossPercent` stay within 0 to 1 and 0 to 100, and return -1 only when the server received nothing. `RunMulti` combines servers by adding each one's expected datagrams, so its combined `Max` is one less than the datagrams sent to all of them. v1 added the `Max` values, which made combined losses too low.

## Units and locations

| v1 | v2 |
| --- | --- |
| `SetUnit(unit)` and `rate.Byte(unit)` | `rate.Format(unit)` |
| `DecimalBitsUnits`, `DecimalBytesUnits`, `BinaryBitsUnits`, `BinaryBytesUnits` | Removed. `Format` picks the label. |
| `PrintCityList()` | `Cities()`, which returns the predefined locations sorted by label for you to print |

The binary bit labels are `Kibps`, `Mibps`, and `Gibps`, where v1 printed `KiMbps` and `KiGbps`. `GetLocation` and `Cities` return copies, so changing one never changes the predefined cities.

## Errors

These sentinels are removed: `ErrClientNil`, `ErrRequestNil`, `ErrMultipleCallsChunkHandler`, and `ErrFailedGetBufferPool`.

These are new:

| Error | Returned when |
| --- | --- |
| `ErrInvalidOption` | `New` or a call is given an invalid option. |
| `ErrUnexpectedStatus` | A server answers with a status outside 2xx. It is wrapped in a `*StatusError` with the URL and status code. v1 treated such responses as successes. |
| `ErrUnsupported` | A server does not serve the packet loss protocol. |
| `ErrInvalidBaseURL` | `WithBaseURL` or `ParseBaseURL` is given a URL that is not absolute http or https. |
| `ErrInvalidTransferSize` | `WithTransferSizes` is given a size outside its table. |
| `ErrInvalidDSCP`, `ErrDSCPUnsupported` | `WithDSCP` is given a value outside 0 to 63, or the platform cannot mark connections. |
| `ErrInterfaceNotFound`, `ErrNoInterfaceAddress` | `ResolveInterface` is given a name that no interface has, or an interface without a usable address. |

```go
_, err := client.FetchServers(ctx)

var statusErr *speedtest.StatusError
if errors.As(err, &statusErr) {
    fmt.Println(statusErr.URL, statusErr.StatusCode)
}
```

## CLI

The `speedtest-go` command keeps its flags, with these changes:

- Settings come from environment variables prefixed with `SPEEDTEST_GO_`, such as `SPEEDTEST_GO_PING_MODE`. Unprefixed variables such as `DEBUG` or `PING_MODE` no longer apply.
- Unknown `--ping-mode` or `--unit` values, a negative `--thread`, and more than one of `--json`, `--jsonl`, and `--unix` are errors that exit with code 2, where v1 fell back to defaults.
- Failures go to stderr and exit with code 1, where v1 exited with 0.
- `--server` IDs that speedtest.net does not list are an error, where v1 skipped them.
- The settings file moved from `~/.speedtest-go.yaml` to `$XDG_CONFIG_HOME/speedtest-go/config.yaml`, such as `~/.config/speedtest-go/config.yaml` on Linux, and the old file is no longer read. Its keys are the flag names in camelCase, such as `pingMode` in place of `ping-mode`, and an unknown key, a repeated key, or a value of the wrong type is an error. `SPEEDTEST_GO_CONFIG` can name the file too.
