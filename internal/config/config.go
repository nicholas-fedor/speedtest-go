package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/viper"

	"github.com/nicholas-fedor/speedtest-go/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/speedtest"
)

// Config holds the application configuration.
type Config struct {
	ServerIDs     []int
	CustomURL     string
	SavingMode    bool
	JSONOutput    bool
	JSONLOutput   bool
	UnixOutput    bool
	Location      string
	City          string
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

// Setup applies global side effects based on the configuration and returns
// the resulting Config so callers receive any defaults applied here.
func Setup(cfg Config) Config {
	speedtest.SetUnit(parser.ParseUnit(cfg.Unit))

	if cfg.SavingMode && !cfg.JSONOutput && !cfg.JSONLOutput && !cfg.UnixOutput {
		cfg.UnixOutput = true
	}

	if cfg.JSONOutput || cfg.JSONLOutput || cfg.UnixOutput {
		log.SetOutput(io.Discard)
		log.SetFlags(0)
	}

	return cfg
}

// InitViper configures viper for config file and environment variable loading.
// An empty cfgFile means the default $HOME/.speedtest-go.yaml is used.
func InitViper(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to find home directory: %w", err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".speedtest-go")
	}

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())

		return nil
	}

	// Missing default config file is optional; any other read/parse error is fatal.
	var notFound viper.ConfigFileNotFoundError
	if cfgFile == "" && errors.As(err, &notFound) {
		return nil
	}

	return fmt.Errorf("failed to read config file: %w", err)
}

// Load reads the current viper state and returns a Config.
func Load() (Config, error) {
	cfg := Config{
		ServerIDs:     viper.GetIntSlice("server"),
		CustomURL:     viper.GetString("custom-url"),
		SavingMode:    viper.GetBool("saving-mode"),
		JSONOutput:    viper.GetBool("json"),
		JSONLOutput:   viper.GetBool("jsonl"),
		UnixOutput:    viper.GetBool("unix"),
		Location:      viper.GetString("location"),
		City:          viper.GetString("city"),
		Proxy:         viper.GetString("proxy"),
		Source:        viper.GetString("source"),
		DNSBindSource: viper.GetBool("dns-bind-source"),
		Multi:         viper.GetBool("multi"),
		Thread:        viper.GetInt("thread"),
		Search:        viper.GetString("search"),
		UserAgent:     viper.GetString("ua"),
		NoDownload:    viper.GetBool("no-download"),
		NoUpload:      viper.GetBool("no-upload"),
		PingMode:      viper.GetString("ping-mode"),
		Unit:          viper.GetString("unit"),
		Debug:         viper.GetBool("debug"),
	}

	return cfg, nil
}
