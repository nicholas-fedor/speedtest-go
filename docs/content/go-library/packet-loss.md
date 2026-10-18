---
title: Packet Loss
description: Measure packet loss to a server with Client.PacketLoss, or sample it as it arrives with the packet loss analyzer
weight: 4
---

Packet loss is measured over UDP, so it does not go through a proxy. A server that does not serve the packet loss protocol returns `ErrUnsupported`. [How a Test Works](/getting-started/how-it-works/#packet-loss) explains the counts and how the loss is calculated.

| Approach | Use it to |
| --- | --- |
| [`Client.PacketLoss`](#one-measurement) | Measure the loss to one server for a while |
| [The packet loss analyzer](#the-packet-loss-analyzer) | Receive each sample as it arrives, or combine several servers |

## One measurement

`PacketLoss` measures until its context ends, so give it a timeout. A failed measurement still returns the counts it saw:

```go
lossCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
packetLoss, err := client.PacketLoss(lossCtx, server)
cancel()

if err != nil {
	log.Println(err)
}

fmt.Println(packetLoss, packetLoss.LossPercent())
```

`LossPercent` returns -1 when the server received nothing.

## The packet loss analyzer

`client.NewPacketLossAnalyzer(options)` creates an analyzer that connects the way the client does. Pass `nil` for the default options.

| Method | Does |
| --- | --- |
| `Run(host, callback)` | Measures one server, and calls `callback` with each sample |
| `RunMulti(hosts)` | Measures several servers at once, and returns their combined counts |
| `RunWithContext`, `RunMultiWithContext` | The same, ending when a context ends |

This program uses both. The [packet loss example](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/packet_loss/main.go) runs the analyzer on every available server at once.

```go
package main

import (
	"context"
	"fmt"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
	"log"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Note: The current packet loss analyzer does not support udp over http.
// This means we cannot get packet loss through a proxy.
func main() {
	// Retrieve the server with the lowest latency
	speedtestClient, err := speedtest.New()
	checkError(err)
	serverList, err := speedtestClient.FetchServers(context.Background())
	checkError(err)
	targets, err := serverList.FindServer([]int{})
	checkError(err)

	// Create a packet loss analyzer that connects the way the client does, with default options
	analyzer := speedtestClient.NewPacketLossAnalyzer(nil)

	// Perform packet loss analysis on the target server
	for _, server := range targets {
		err := analyzer.Run(server.Host, func(packetLoss *speedtest.PacketLossStats) {
			fmt.Println(packetLoss, server.Host, server.Name)
			// fmt.Println(packetLoss.Loss())
		})
		checkError(err)
	}

	// or test several servers at the same time.
	packetLoss, err := analyzer.RunMulti(serverList.Available().Hosts())
	checkError(err)
	fmt.Println(packetLoss)
}
```
