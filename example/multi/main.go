// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: MIT

// Package main demonstrates multi-server speedtest.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

// main measures the download and upload rates across the lowest latency server and the other servers at once.
func main() {
	speedtestClient, err := speedtest.New()
	checkError(err)

	ctx := context.TODO()

	serverList, _ := speedtestClient.FetchServers(ctx)
	targets, _ := serverList.FindServer([]int{})

	if len(targets) > 0 {
		// Use s as the main server and the rest of the list as auxiliary servers.
		// The main server always gets connections, and the auxiliary servers share the rest.
		s := targets[0]

		download, err := speedtestClient.DownloadMulti(ctx, s, serverList)
		checkError(err)

		upload, err := speedtestClient.UploadMulti(ctx, s, serverList)
		checkError(err)

		_, _ = fmt.Fprintf(os.Stdout, "Download: %s, Upload: %s\n", download.Rate, upload.Rate)
	}
}

// checkError ends the program with err when it is not nil.
//
// Parameters:
//   - err: the error to check.
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
