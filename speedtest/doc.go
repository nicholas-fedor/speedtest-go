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
//   - Speed Testing: Measures download and upload speeds over HTTP, or over the speedtest.net TCP protocol with
//     WithProtocol
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
//   - Measured Rates: Reports each test's rate from the bytes it counted, and passes a smoothed live estimate to
//     WithRateCallback while it runs
//   - Multiple Ping Protocols: Supports HTTP, TCP, and ICMP ping methods
//   - Flexible Configuration: Validated options for proxy settings, source addresses, test sizes, and logging
//   - Distance-based Server Selection: Automatically selects geographically optimal servers
//   - Statistics: StandardDeviation reports the mean, variance, standard deviation, minimum, and maximum of a set of
//     values
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
// # Migrating from v1
//
// The migration guide at https://github.com/nicholas-fedor/speedtest-go/blob/master/docs/v2-migration.md maps each
// v1 function, method, and field to its v2 replacement.
//
// # Example Usage
//
//	client, err := speedtest.New(speedtest.WithConnections(4))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//
//	servers, err := client.FetchServers(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// FindServer without IDs returns the server with the lowest latency.
//	targets, err := servers.FindServer(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	server := targets[0]
//
//	ping, err := client.Ping(ctx, server)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	download, err := client.Download(ctx, server, speedtest.WithRateCallback(func(rate speedtest.ByteRate) {
//	    fmt.Println("Download:", rate)
//	}))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	upload, err := client.Upload(ctx, server)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", ping.Latency, download.Rate, upload.Rate)
package speedtest
