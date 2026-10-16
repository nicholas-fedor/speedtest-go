# Naive Speed Test Example

This example demonstrates a basic usage of the `speedtest-go` library to perform an internet speed test. It fetches a list of Speedtest servers, picks the one with the lowest latency, and runs ping, download, and upload tests against it, printing the results.

## What It Does

The program performs the following steps:

1. Fetches a list of Speedtest servers.
2. Picks the server with the lowest latency as the target.
3. For the target server:
   - Performs a ping test to measure latency.
   - Runs a download speed test.
   - Runs an upload speed test.
   - Prints the latency, download speed, and upload speed.

## Prerequisites

- Go 1.27 or later
- Internet connection
- Access to Speedtest servers (ensure your network allows connections to speedtest.net servers)

## How to Run

1. Navigate to the `example/naive` directory:

   ```bash
   cd example/naive
   ```

2. Run the program:

   ```bash
   go run main.go
   ```

The program will automatically fetch servers and perform tests. Note that this may take some time depending on the number of servers and your internet connection.

## Example Output

```text
Latency: 15.2ms, Download: 45.67 Mbps, Upload: 12.34 Mbps
```

## Code Explanation

The main logic is in the `main()` function:

```go
speedtestClient, err := speedtest.New()
checkError(err)

serverList, _ := speedtestClient.FetchServers()
targets, _ := serverList.FindServer([]int{})

ctx := context.Background()

for _, s := range targets {
    ping, err := speedtestClient.Ping(ctx, s)
    checkError(err)

    download, err := speedtestClient.Download(ctx, s)
    checkError(err)

    upload, err := speedtestClient.Upload(ctx, s)
    checkError(err)

    fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", ping.Latency, download.Rate, upload.Rate)
}
```

- `speedtest.New()` creates a client, and its `FetchServers()` method retrieves a list of available Speedtest servers.
- `serverList.FindServer([]int{})` returns the server with the lowest latency, since an empty slice names no server IDs.
- The ping, download, and upload tests run one after another, and each returns its own result.
- Rates print in Mbps, and latencies print as durations.

## Additional Configuration Options

The code includes commented examples for additional configurations:

- Fetch user information: `speedtestClient.FetchUserInfo()`
- Choose servers near a city: `speedtest.New(speedtest.WithLocation(tokyo))`, with `tokyo` from `speedtest.GetLocation("tokyo")`
- Choose servers near coordinates: `speedtest.ParseLocation("osaka", "34.69,135.50")` gives a location for `speedtest.WithLocation`
- Send from an IP address or network interface: `speedtest.New(speedtest.WithSource("192.168.1.101"))`
- Fetch specific server by ID: `speedtestClient.FetchServerByID("28910")`

Uncomment and modify these lines as needed for more targeted testing.

## Error Handling

The `checkError` function provides basic error handling by panicking on any error. In a production application, you might want to implement more sophisticated error handling.

## Notes

- This example tests one server. Pass server IDs to `FindServer` to test specific servers instead.
- Ensure your firewall and network settings allow connections to Speedtest servers.
- Each test measures on its own, so nothing needs resetting between servers.
