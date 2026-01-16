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
//   - Speedtest: The main client structure that manages speedtest operations
//   - Server: Represents a speedtest server with properties like URL, location, latency, speeds, and packet loss
//   - User: Contains user information including IP, ISP, and coordinates
//   - Manager/DataManager: Handles data collection and rate calculations during tests
//   - Chunk: Manages individual data transfer chunks with rate and duration tracking
//   - ByteRate: Represents data transfer rates with flexible unit formatting (bps, Kbps, Mbps, etc.)
//
// ## Main Functions
//
//   - New(): Creates a new speedtest client with optional configuration
//   - FetchUserInfo(): Retrieves user information from speedtest.net
//   - FetchServers(): Discovers available speedtest servers
//   - Server.PingTest(): Measures latency to a server
//   - Server.DownloadTest(): Performs download speed test
//   - Server.UploadTest(): Performs upload speed test
//   - Server.TestAll(): Runs complete test suite (ping, download, upload)
//
// # Key Features
//
//   - Multi-threaded Testing: Configurable concurrent connections for more accurate speed measurements
//   - EWMA Rate Calculation: Uses Exponentially Weighted Moving Average for stable speed estimates
//   - Multiple Ping Protocols: Supports HTTP, TCP, and ICMP ping methods
//   - Flexible Configuration: Customizable timeouts, proxy settings, source addresses, and debug modes
//   - Distance-based Server Selection: Automatically selects geographically optimal servers
//   - Statistics: Comprehensive statistics including mean, standard deviation, and coefficient of variation
//
// # Subpackages
//
//   - transport: Implements TCP and UDP transport layers for low-level network operations
//   - internal: Provides Welford's algorithm for online statistical calculations
//
// # Dependencies
//
// The package has minimal external dependencies, primarily using Go's standard library
// with optional dependencies for CLI tooling and testing.
//
// # Example Usage
//
//	client := speedtest.New()
//	user, _ := client.FetchUserInfo()
//	servers, _ := client.FetchServers()
//	server := servers[0]
//	server.PingTest()
//	server.DownloadTest()
//	server.UploadTest()
//	fmt.Printf("Download: %s, Upload: %s, Latency: %s\n",
//	    server.DLSpeed, server.ULSpeed, server.Latency)
package speedtest
