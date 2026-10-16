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
//
// Parameters:
//   - ctx: ends the server lookups. The CLI cancels it on an interrupt.
//   - cfg: the settings.
//
// Returns:
//   - error: the error of setting up the client or of a lookup, which wraps the context's error when it ended one.
func RunList(ctx context.Context, cfg config.Config) error {
	cfg = config.Setup(cfg)

	//nolint:contextcheck // speedtest.New takes no context, and only its debug log of the settings uses one.
	speedtestClient, err := setupSpeedtestClient(cfg)
	if err != nil {
		return err
	}

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
