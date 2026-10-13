// Package main demonstrates naive speedtest.
package main

import (
	"fmt"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

func main() {
	// _, _ = speedtest.FetchUserInfo()

	// Configure a client to choose servers near a city label or coordinates, or to send from an IP address or
	// network interface, then use its methods instead of the package functions below.
	// speedtestClient := speedtest.New(speedtest.WithUserConfig(&speedtest.UserConfig{CityFlag: "tokyo"}))
	// speedtestClient := speedtest.New(speedtest.WithUserConfig(&speedtest.UserConfig{LocationFlag: "34.69,135.50"}))
	// speedtestClient := speedtest.New(speedtest.WithUserConfig(&speedtest.UserConfig{Source: "192.168.1.101"}))

	// Search server using serverID.
	// eg: fetch server with ID 28910.
	// speedtest.ErrServerNotFound will be returned if the server cannot be found.
	// server, err := speedtest.FetchServerByID("28910")
	serverList, _ := speedtest.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	for _, server := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		checkError(server.PingTest(nil))
		checkError(server.DownloadTest())
		checkError(server.UploadTest())

		// Note: The unit of server.DLSpeed, server.ULSpeed is bytes per second, this is a float64.
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Latency: %s, Download: %s, Upload: %s\n",
			server.Latency,
			server.DLSpeed,
			server.ULSpeed,
		)
		server.Context.Reset()
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
