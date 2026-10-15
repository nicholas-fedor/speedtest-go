// Package main demonstrates naive speedtest.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

func main() {
	// Options configure the client to choose servers near a city label or coordinates, or to send from an IP address
	// or network interface.
	// tokyo, _ := speedtest.GetLocation("tokyo")
	// speedtestClient, err := speedtest.New(speedtest.WithLocation(tokyo))
	// osaka, _ := speedtest.ParseLocation("osaka", "34.69,135.50")
	// speedtestClient, err := speedtest.New(speedtest.WithLocation(osaka))
	// speedtestClient, err := speedtest.New(speedtest.WithSource("192.168.1.101"))
	speedtestClient, err := speedtest.New()
	checkError(err)

	// _, _ = speedtestClient.FetchUserInfo()

	// Search server using serverID.
	// eg: fetch server with ID 28910.
	// speedtest.ErrServerNotFound will be returned if the server cannot be found.
	// server, err := speedtestClient.FetchServerByID("28910")
	serverList, _ := speedtestClient.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	ctx := context.Background()

	for _, server := range targets {
		// Please make sure your host can access this test server,
		// otherwise you will get an error.
		// It is recommended to replace a server at this time
		ping, err := speedtestClient.Ping(ctx, server)
		checkError(err)

		download, err := speedtestClient.Download(ctx, server)
		checkError(err)

		upload, err := speedtestClient.Upload(ctx, server)
		checkError(err)

		// Note: Rate is in bytes per second, a float64 that prints in Mbps.
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Latency: %s, Download: %s, Upload: %s\n",
			ping.Latency,
			download.Rate,
			upload.Rate,
		)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
