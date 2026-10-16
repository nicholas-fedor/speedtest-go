// Package speedtest is a Go library for performing network speed tests,
// compatible with the speedtest.net protocol. It provides comprehensive speed
// testing capabilities including download and upload speed measurements, latency
// testing, and packet loss calculations.
//
// # Main Functionalities
//
//   - User Information: Fetches user details such as IP address, ISP, and geographic location from speedtest.net
//   - Server Discovery: Retrieves and manages lists of available speedtest servers,
//     with automatic distance calculation based on user location
//   - Latency Testing: Performs ping tests using HTTP, TCP, or ICMP protocols with jitter and latency statistics
//   - Speed Testing: Measures download speeds via HTTP GET requests and upload speeds
//     via HTTP POST requests to speedtest servers
//   - Packet Loss: Calculates uplink packet loss using TCP and UDP transport implementations
//   - Multi-server Testing: Supports concurrent testing across multiple servers for more accurate results
//
// # Key Types and Functions
//
// ## Core Types
//
//   - Client: The main client structure that manages speedtest operations
//   - Server: Represents a speedtest server with its URL, location, and the Latency of a quick probe
//   - Result: Collects a server's ping, download, upload, and packet loss measurements
//   - PacketLossStats: Holds the counts a server reports for packet loss, with the loss ratio and percentage
//   - User: Contains user information including IP, ISP, and coordinates
//   - ByteRate: Represents data transfer rates with flexible unit formatting (bps, Kbps, Mbps, etc.)
//
// ## Main Functions
//
//   - New(): Creates a new speedtest client from options such as WithProxy, WithSource, and WithLogger
//   - Client.FetchUserInfo(): Retrieves user information from speedtest.net
//   - Client.FetchServers(): Discovers available speedtest servers, sorted by distance
//   - JSON() and JSONL(): Marshal results with the user information
//   - ByteRate.Format(): Formats a rate in a chosen unit
//   - Cities(): Lists the predefined city labels for GetLocation
//   - Client.Ping(): Measures latency to a server
//   - Client.Download() and Client.Upload(): Measure download and upload speed, with each connection's results
//   - Client.DownloadMulti() and Client.UploadMulti(): Measure speed across several servers at once
//   - Client.PacketLoss(): Measures packet loss to a server
//
// # Key Features
//
//   - Multi-threaded Testing: Configurable concurrent connections for more accurate speed measurements
//   - EWMA Rate Calculation: Uses Exponentially Weighted Moving Average for stable speed estimates
//   - Multiple Ping Protocols: Supports HTTP, TCP, and ICMP ping methods
//   - Flexible Configuration: Validated options for proxy settings, source addresses, test sizes, and logging
//   - Distance-based Server Selection: Automatically selects geographically optimal servers
//   - Statistics: Comprehensive statistics including mean, standard deviation, and coefficient of variation
//
// # Subpackages
//
//   - internal/transport: Implements the TCP and UDP speedtest protocols for latency, transfers, and packet loss
//   - internal/meter: Runs the transfer workers and measures their rates
//   - internal/welford: Provides Welford's algorithm for online statistical calculations
//
// # Dependencies
//
// The package has minimal external dependencies, primarily using Go's standard library
// with optional dependencies for CLI tooling and testing.
//
// # Example Usage
//
//	client, err := speedtest.New(speedtest.WithConnections(4))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	ctx := context.Background()
//	servers, _ := client.FetchServers()
//	server := servers[0]
//	ping, _ := client.Ping(ctx, server)
//	download, _ := client.Download(ctx, server, speedtest.WithRateCallback(func(rate speedtest.ByteRate) {
//	    fmt.Println("Download:", rate)
//	}))
//	upload, _ := client.Upload(ctx, server)
//	fmt.Printf("Download: %s, Upload: %s, Latency: %s\n",
//	    download.Rate, upload.Rate, ping.Latency)
package speedtest
