package app

import (
	"fmt"
	"log"

	"github.com/showwin/speedtest-go/internal/output"
	"github.com/showwin/speedtest-go/internal/parser"
	"github.com/showwin/speedtest-go/speedtest"
)

// RunList lists available speedtest servers.
func RunList(cfg Config) error {
	setupConfig(cfg)

	// 0. speed test setting
	speedtestClient := speedtest.New(speedtest.WithUserConfig(
		&speedtest.UserConfig{
			UserAgent:      cfg.UserAgent,
			Proxy:          cfg.Proxy,
			Source:         cfg.Source,
			DNSBindSource:  cfg.DNSBindSource,
			Debug:          cfg.Debug,
			PingMode:       parser.ParseProto(cfg.PingMode),
			SavingMode:     cfg.SavingMode,
			MaxConnections: cfg.Thread,
			CityFlag:       cfg.City,
			LocationFlag:   cfg.Location,
			Keyword:        cfg.Search,
		}))

	// retrieving servers
	servers, err := speedtestClient.FetchServers()
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	log.Printf("Found %d Public Servers\n", len(servers))
	output.ShowServerList(servers)

	return nil
}
