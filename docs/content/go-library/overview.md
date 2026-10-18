---
title: Overview
description: Install the speedtest package, and measure latency, download and upload rates, and packet loss from a Go program
weight: 1
---

```bash
go get github.com/nicholas-fedor/speedtest-go/v2
```

The package is `github.com/nicholas-fedor/speedtest-go/v2/speedtest`. Its full API is on [pkg.go.dev](https://pkg.go.dev/github.com/nicholas-fedor/speedtest-go/v2/speedtest).

## How it works

1. `speedtest.New` builds a client from [options](../options/), and reports every invalid one.
2. The client fetches the servers near you, or near a location you choose.
3. Each test is a client method that takes a context and a server, and returns its own result. [Measurements](../measurements/) covers each one.

No result is stored on the client or the server, so nothing needs resetting between tests.

## Example

This program finds the server with the lowest latency, then measures latency, download and upload rates, and packet loss. It is based on the [naive example](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/naive/main.go).

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

func main() {
	// Options configure the client, and New reports every invalid one. For example:
	//  - speedtest.WithProxy("socks5://127.0.0.1:7890") uses a proxy.
	//  - speedtest.WithSource("eth0") sends from a source IP address or network interface name.
	//  - speedtest.WithLocation(tokyo), with tokyo from speedtest.GetLocation("tokyo"), chooses servers near a city
	//    label instead of your own location. speedtest.ParseLocation("osaka", "34.6952,135.5006") takes coordinates.
	speedtestClient, err := speedtest.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Get user's network information
	// user, _ := speedtestClient.FetchUserInfo(ctx)

	// Search server using serverID.
	// eg: fetch server with ID 28910.
	// speedtest.ErrServerNotFound will be returned if the server cannot be found.
	// server, err := speedtestClient.FetchServerByID(ctx, "28910")

	serverList, err := speedtestClient.FetchServers(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Without server IDs, FindServer returns the server with the lowest latency.
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range targets {
		ping, err := speedtestClient.Ping(ctx, s)
		if err != nil {
			log.Fatal(err)
		}

		download, err := speedtestClient.Download(ctx, s)
		if err != nil {
			log.Fatal(err)
		}

		upload, err := speedtestClient.Upload(ctx, s)
		if err != nil {
			log.Fatal(err)
		}

		// Note: Rate is in bytes per second, a float64 that prints in Mbps. Each test reports its own results.
		fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", ping.Latency, download.Rate, upload.Rate)

		// PacketLoss measures until its context ends, and a failed measurement still returns what it saw.
		lossCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		packetLoss, err := speedtestClient.PacketLoss(lossCtx, s)
		cancel()

		if err != nil {
			log.Println(err)
		}

		fmt.Println(packetLoss)
	}
}
```

| More examples | Shows |
| --- | --- |
| [Multi-server](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/multi/main.go) | One test spread across several servers |
| [Packet loss](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/packet_loss/main.go) | The packet loss analyzer, across one server or several |

Upgrading from v1? The [migration guide](../migration/) maps each v1 function, method, and field to its v2 replacement.
