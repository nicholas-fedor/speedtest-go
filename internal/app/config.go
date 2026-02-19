package app

import (
	"io"
	"log"

	"github.com/nicholas-fedor/speedtest-go/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

// Config holds the application configuration.
type Config struct {
	ShowList      bool
	ServerIDs     []int
	CustomURL     string
	SavingMode    bool
	JSONOutput    bool
	JSONLOutput   bool
	UnixOutput    bool
	Location      string
	City          string
	ShowCityList  bool
	Proxy         string
	Source        string
	DNSBindSource bool
	Multi         bool
	Thread        int
	Search        string
	UserAgent     string
	NoDownload    bool
	NoUpload      bool
	PingMode      string
	Unit          string
	Debug         bool
}

// setupConfig sets up global configuration based on flags.
func setupConfig(cfg Config) {
	speedtest.SetUnit(parser.ParseUnit(cfg.Unit))

	// discard standard log.
	log.SetOutput(io.Discard)
	log.SetFlags(0)

	// start unix output for saving mode by default.
	if cfg.SavingMode && !cfg.JSONOutput && !cfg.JSONLOutput && !cfg.UnixOutput {
		cfg.UnixOutput = true
	}
}
