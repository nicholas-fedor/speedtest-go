package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
)

// RunList lists available speedtest servers.
func RunList(cfg config.Config) error {
	cfg = config.Setup(cfg)

	speedtestClient, err := setupSpeedtestClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// retrieving servers
	servers, err := speedtestClient.FetchServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(cfg.CountryCodes) > 0 {
		var codes []string

		servers, codes, err = filterByCountry(ctx, speedtestClient, cfg, servers)
		if err != nil {
			return err
		}

		log.Printf("Found %d Public Servers in %s\n", len(servers), strings.Join(codes, ", "))
	} else {
		log.Printf("Found %d Public Servers\n", len(servers))
	}

	if cfg.MaskISP {
		output.ShowMaskedServerList(servers)
	} else {
		output.ShowServerList(servers)
	}

	return nil
}
