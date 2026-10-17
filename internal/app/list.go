package app

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/config"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/output"
	"github.com/nicholas-fedor/speedtest-go/v2/internal/runner"
)

// runList lists the servers the settings select.
//
// Parameters:
//   - ctx: ends the server lookups. The CLI cancels it on an interrupt.
//   - cfg: the settings.
//   - out: receives the list.
//
// Returns:
//   - error: the error of setting up the client or of a lookup, which wraps the context's error when it ended one.
func runList(ctx context.Context, cfg config.Config, out io.Writer) error {
	cfg = config.Setup(cfg)

	//nolint:contextcheck // speedtest.New takes no context, and only its debug log of the settings uses one.
	speedtestClient, err := setupSpeedtestClient(cfg)
	if err != nil {
		return err
	}

	servers, codes, err := runner.ListServers(ctx, speedtestClient, runSettings(cfg))
	if err != nil {
		return err //nolint:wrapcheck // ListServers describes its errors.
	}

	if len(codes) > 0 {
		log.Printf("Found %d Public Servers in %s\n", len(servers), strings.Join(codes, ", "))
	} else {
		log.Printf("Found %d Public Servers\n", len(servers))
	}

	if cfg.MaskISP {
		output.WriteMaskedServerList(out, servers)
	} else {
		output.WriteServerList(out, servers)
	}

	return nil
}
