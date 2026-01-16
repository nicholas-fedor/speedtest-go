// Package main demonstrates naive speedtest.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/showwin/speedtest-go/speedtest"
)

func main() {
	// _, _ = speedtest.FetchUserInfo()
	// Get a list of servers near a specified location
	// user.SetLocationByCity("Tokyo")
	// user.SetLocation("Osaka", 34.6952, 135.5006)

	// Select a network card as the data interface.
	// speedtest.WithUserConfig(&speedtest.UserConfig{Source: "192.168.1.101"})(speedtestClient)

	// Search server using serverID.
	// eg: fetch server with ID 28910.
	// speedtest.ErrEmptyServers will be returned if the server cannot be found.
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
		log.Fatal(err)
	}
}
