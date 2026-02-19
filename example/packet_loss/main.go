// Package main provides an example of using the packet loss analyzer.
package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nicholas-fedor/speedtest-go/speedtest"
	"github.com/nicholas-fedor/speedtest-go/speedtest/transport"
)

const packetSendingInterval = 100 * time.Millisecond

// Note: The current packet loss analyzer does not support udp over http.
// This means we cannot get packet loss through a proxy.
func main() {
	// 0. Fetching servers
	serverList, err := speedtest.FetchServers()
	checkError(err)

	// 1. Retrieve available servers
	targets := serverList.Available()

	// 2. Create a packet loss analyzer, use default options
	analyzer := speedtest.NewPacketLossAnalyzer(&speedtest.PacketLossAnalyzerOptions{
		PacketSendingInterval: packetSendingInterval,
	})

	waitGroup := &sync.WaitGroup{}
	// 3. Perform packet loss analysis on all available servers
	for _, server := range *targets {
		waitGroup.Add(1)
		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		// go func(server *speedtest.Server, analyzer *speedtest.PacketLossAnalyzer,
		//	ctx context.Context, cancel context.CancelFunc) {
		go func(server *speedtest.Server, analyzer *speedtest.PacketLossAnalyzer) {
			// defer cancel()
			defer waitGroup.Done()
			// Note: Please call ctx.cancel at the appropriate time to release resources if you use analyzer.RunWithContext
			// we using analyzer.Run() here.
			err = analyzer.Run(server.Host, func(packetLoss *transport.PLoss) {
				_, _ = fmt.Fprintln(os.Stdout, packetLoss, server.Host, server.Name)
			})
			// err = analyzer.RunWithContext(ctx, server.Host, func(packetLoss *transport.PLoss) {
			//	fmt.Println(packetLoss, server.Host, server.Name)
			// })
			if err != nil {
				_, _ = fmt.Fprintln(os.Stdout, err)
			}
			// }(server, analyzer, ctx, cancel)
		}(server, analyzer)
	}

	waitGroup.Wait()

	// use mixed PacketLoss
	mixed, err := analyzer.RunMulti(serverList.Hosts())
	checkError(err)

	_, _ = fmt.Fprintf(os.Stdout, "Mixed packets lossed: %.2f%%\n", mixed.LossPercent())
	_, _ = fmt.Fprintf(os.Stdout, "Mixed packets lossed: %.2f\n", mixed.Loss())
	_, _ = fmt.Fprintf(os.Stdout, "Mixed packets lossed: %s\n", mixed)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
