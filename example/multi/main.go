// Package main demonstrates multi-server speedtest.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

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

		_, _ = fmt.Fprintf(os.Stdout, "Download: %s, Upload: %s\n", download.Rate, upload.Rate)
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
