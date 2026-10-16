# Multi-Server Speed Test Example

This example demonstrates how to perform a speed test using multiple servers simultaneously with the speedtest-go library. It showcases the multi-server testing capabilities, where one server acts as the primary (main) server and others serve as auxiliary servers to distribute the load.

## Overview

The program fetches a list of Speedtest servers, picks the one with the lowest latency as the main server, and then performs both download and upload tests across the main server and the other available servers in the list at once. The main server always gets connections, and the auxiliary servers share the rest.

## Key Features

- **Multi-server testing**: Utilizes multiple servers simultaneously for more accurate and comprehensive speed measurements
- **Load distribution**: The main server always gets connections, while the auxiliary servers share the rest
- **Context support**: Uses Go contexts for proper cancellation and timeout handling

## How to Run

1. Ensure you have Go installed and the speedtest-go library available
2. Navigate to the example directory:

   ```bash
   cd example/multi
   ```

3. Run the example:

   ```bash
   go run main.go
   ```

## Code Explanation

The main functionality is contained in the `main()` function:

```go
func main() {
    speedtestClient, err := speedtest.New()
    checkError(err)

    serverList, _ := speedtestClient.FetchServers()
    targets, _ := serverList.FindServer([]int{})

    if len(targets) > 0 {
        // Use s as the main server and the rest of the list as auxiliary servers.
        // The main server always gets connections, and the auxiliary servers share the rest.
        s := targets[0]
        download, err := speedtestClient.DownloadMulti(context.TODO(), s, serverList)
        checkError(err)

        upload, err := speedtestClient.UploadMulti(context.TODO(), s, serverList)
        checkError(err)

        fmt.Printf("Download: %s, Upload: %s\n", download.Rate, upload.Rate)
    }
}
```

### Breakdown

1. **Fetch Servers**: Retrieves the list of available speed test servers
2. **Find the Main Server**: Selects the server with the lowest latency, since an empty slice names no server IDs
3. **Multi-Download Test**: Performs download speed test across multiple servers
4. **Multi-Upload Test**: Performs upload speed test across multiple servers
5. **Display Results**: Prints the measured download and upload speeds

## Expected Output

When run successfully, the program will output something similar to:

```text
Download: 45.67 Mbps, Upload: 12.34 Mbps
```

## Notes

- The actual speeds will vary based on your internet connection and the selected servers
- Error handling is implemented via the `checkError` function, which will terminate the program on any errors
- This example uses `context.TODO()` for simplicity. In production code, consider using proper context management for timeouts and cancellation
