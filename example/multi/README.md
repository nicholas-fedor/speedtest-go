# Multi-Server Speed Test Example

This example demonstrates how to perform a speed test using multiple servers simultaneously with the speedtest-go library. It showcases the multi-server testing capabilities, where one server acts as the primary (main) server and others serve as auxiliary servers to distribute the load.

## Overview

The program fetches a list of available SpeedTest servers, selects all of them as targets, and then performs both download and upload tests using multiple servers concurrently. The first server in the list is designated as the main server, which receives a greater proportion of the testing load compared to the auxiliary servers.

## Key Features

- **Multi-server testing**: Utilizes multiple servers simultaneously for more accurate and comprehensive speed measurements
- **Load distribution**: The main server handles a larger share of the traffic, while auxiliary servers assist in the testing process
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
    serverList, _ := speedtest.FetchServers()
    targets, _ := serverList.FindServer([]int{})

    if len(targets) > 0 {
        // Use s as main server and use targets as auxiliary servers.
        // The main server is loaded at a greater proportion than the auxiliary servers.
        s := targets[0]
        checkError(s.MultiDownloadTestContext(context.TODO(), targets))
        checkError(s.MultiUploadTestContext(context.TODO(), targets))
        fmt.Printf("Download: %s, Upload: %s\n", s.DLSpeed, s.ULSpeed)
    }
}
```

### Breakdown

1. **Fetch Servers**: Retrieves the list of available speed test servers
2. **Find Targets**: Selects servers to use (empty slice means use all available servers)
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
- This example uses `context.TODO()` for simplicity; in production code, consider using proper context management for timeouts and cancellation
