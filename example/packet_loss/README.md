# Packet Loss Analysis Example

This example demonstrates how to perform packet loss analysis on speedtest servers using the speedtest-go library. It measures packet loss by sending UDP packets to multiple servers concurrently and provides both individual server results and a mixed (aggregated) packet loss calculation.

## Overview

The program fetches a list of available speedtest servers, creates a packet loss analyzer with configurable options, and then performs packet loss measurements on all available servers simultaneously using goroutines. After individual analyses, it computes a mixed packet loss result across all tested servers.

## Key Features

- **Concurrent analysis**: Tests packet loss to multiple servers simultaneously for efficiency
- **UDP-based measurement**: Uses UDP packets to measure packet loss accurately
- **Mixed results**: Provides aggregated packet loss statistics across all servers
- **Configurable options**: Allows customization of packet sending intervals

## Limitations

- Does not support UDP over HTTP proxies
- Packet loss measurement cannot be performed through proxies

## How to Run

1. Ensure you have Go installed and the speedtest-go library available
2. Navigate to the example directory:

   ```bash
   cd example/packet_loss
   ```

3. Run the example:

   ```bash
   go run main.go
   ```

## Code Explanation

The main functionality is contained in the `main()` function:

```go
func main() {
    // 0. Fetching servers
    serverList, err := speedtest.FetchServers()
    checkError(err)

    // 1. Retrieve available servers
    targets := serverList.Available()

    // 2. Create a packet loss analyzer, use default options
    analyzer := speedtest.NewPacketLossAnalyzer(&speedtest.PacketLossAnalyzerOptions{
        PacketSendingInterval: time.Millisecond * 100,
    })

    wg := &sync.WaitGroup{}
    // 3. Perform packet loss analysis on all available servers
    for _, server := range *targets {
        wg.Add(1)
        go func(server *speedtest.Server, analyzer *speedtest.PacketLossAnalyzer) {
            defer wg.Done()
            err = analyzer.Run(server.Host, func(packetLoss *transport.PLoss) {
                fmt.Println(packetLoss, server.Host, server.Name)
            })
            if err != nil {
                fmt.Println(err)
            }
        }(server, analyzer)
    }
    wg.Wait()

    // use mixed PacketLoss
    mixed, err := analyzer.RunMulti(serverList.Hosts())
    checkError(err)
    fmt.Printf("Mixed packets lossed: %.2f%%\n", mixed.LossPercent())
    fmt.Printf("Mixed packets lossed: %.2f\n", mixed.Loss())
    fmt.Printf("Mixed packets lossed: %s\n", mixed)
}
```

### Breakdown

1. **Fetch Servers**: Retrieves the list of available speedtest servers
2. **Get Available Targets**: Filters for servers that are currently available
3. **Create Analyzer**: Initializes a packet loss analyzer with a 100ms packet sending interval
4. **Concurrent Analysis**: Launches goroutines to analyze packet loss for each server, printing results as they complete
5. **Mixed Analysis**: Performs an aggregated packet loss calculation across all server hosts
6. **Display Results**: Prints both individual server results and the mixed packet loss statistics

## Expected Output

When run successfully, the program will output individual server packet loss results followed by mixed results:

```text
{Loss:0.05 LossPercent:5.00} speedtest.example.com Example Server
{Loss:0.02 LossPercent:2.00} speedtest2.example.com Example Server 2
...
Mixed packets lossed: 3.50%
Mixed packets lossed: 0.04
Mixed packets lossed: {Loss:0.035 LossPercent:3.50}
```

## Notes

- Packet loss values represent the percentage of packets lost during transmission
- The mixed result provides an aggregate view across all tested servers
- Actual packet loss will vary based on network conditions and server locations
- Error handling is implemented via the `checkError` function, which will terminate the program on any errors
- The example uses a 100ms interval between packet sends; this can be adjusted via the `PacketLossAnalyzerOptions`
