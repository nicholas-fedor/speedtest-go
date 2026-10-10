package app

import (
	"fmt"
	"log"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
)

// RunList lists available speedtest servers.
func RunList(cfg config.Config) error {
	cfg = config.Setup(cfg)

	speedtestClient := setupSpeedtestClient(cfg)

	// retrieving servers
	servers, err := speedtestClient.FetchServers()
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	log.Printf("Found %d Public Servers\n", len(servers))
	output.ShowServerList(servers)

	return nil
}
