# Naive Speed Test Example

This example demonstrates a basic usage of the `speedtest-go` library to perform internet speed tests against multiple Speedtest servers. It fetches a list of available servers, selects all of them as targets, and runs ping, download, and upload tests for each server, printing the results.

## What It Does

The program performs the following steps:

1. Fetches a list of Speedtest servers.
2. Finds all available servers as test targets.
3. For each target server:
   - Performs a ping test to measure latency.
   - Runs a download speed test.
   - Runs an upload speed test.
   - Prints the latency, download speed, and upload speed.
4. Resets the server context after each test.

## Prerequisites

- Go 1.16 or later
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
Latency: 22.1ms, Download: 42.89 Mbps, Upload: 11.98 Mbps
...
```

## Code Explanation

The main logic is in the `main()` function:

```go
serverList, _ := speedtest.FetchServers()
targets, _ := serverList.FindServer([]int{})

for _, s := range targets {
    checkError(s.PingTest(nil))
    checkError(s.DownloadTest())
    checkError(s.UploadTest())

    fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", s.Latency, s.DLSpeed, s.ULSpeed)
    s.Context.Reset()
}
```

- `speedtest.FetchServers()` retrieves a list of available Speedtest servers.
- `serverList.FindServer([]int{})` selects all servers since an empty slice is passed.
- For each server, tests are run sequentially.
- Speeds are reported in Mbps, latency in milliseconds.

## Additional Configuration Options

The code includes commented examples for additional configurations:

- Fetch user information: `speedtest.FetchUserInfo()`
- Set location by city: `user.SetLocationByCity("Tokyo")`
- Set custom location: `user.SetLocation("Osaka", 34.6952, 135.5006)`
- Select network interface: `speedtest.WithUserConfig(&speedtest.UserConfig{Source: "192.168.1.101"})`
- Fetch specific server by ID: `speedtest.FetchServerByID("28910")`

Uncomment and modify these lines as needed for more targeted testing.

## Error Handling

The `checkError` function provides basic error handling by logging fatal errors. In a production application, you might want to implement more sophisticated error handling.

## Notes

- This example tests against all available servers, which may be time-consuming. For quicker tests, consider limiting the number of servers or selecting specific ones.
- Ensure your firewall and network settings allow connections to Speedtest servers.
- The `s.Context.Reset()` call is important to clean up resources between tests.
