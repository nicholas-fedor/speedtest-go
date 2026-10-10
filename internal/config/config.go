package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/parser"
	"github.com/nicholas-fedor/speedtest-go/v2/speedtest"
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
	PingMode      speedtest.Proto
	Unit          speedtest.UnitType
	Debug         bool
	BaseURL       string
}

// EnvPrefix is the prefix of the environment variables that supply settings. A setting's variable is the prefix,
// an underscore, and the setting name in upper case with hyphens as underscores, such as SPEEDTEST_GO_PING_MODE.
const EnvPrefix = "SPEEDTEST_GO"

// Errors for setting values.
var (
	// ErrInvalidSetting is wrapped by every error that rejects a setting value. The CLI exits with code 2 for it.
	ErrInvalidSetting = errors.New("invalid setting")
	// errNotWholeNumber rejects a count that does not parse as an integer.
	errNotWholeNumber = errors.New("not a whole number")
	// errNegative rejects a count below zero.
	errNegative = errors.New("negative")
)

// Setup applies global side effects based on the configuration and returns
// the resulting Config so callers receive any defaults applied here.
func Setup(cfg Config) Config {
	speedtest.SetUnit(cfg.Unit)

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
// Environment variables are read with [EnvPrefix].
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

	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
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

// Load reads the current viper state and returns a validated Config.
//
// Returns:
//   - Config: the settings.
//   - error: every invalid setting joined, each wrapping [ErrInvalidSetting].
func Load() (Config, error) {
	var errs []error

	invalid := func(flag string, err error) {
		errs = append(errs, fmt.Errorf("%w --%s: %w", ErrInvalidSetting, flag, err))
	}

	pingMode, err := parser.ParseProto(viper.GetString("ping-mode"))
	if err != nil {
		invalid("ping-mode", err)
	}

	unit, err := parser.ParseUnit(viper.GetString("unit"))
	if err != nil {
		invalid("unit", err)
	}

	thread, err := parseThread(viper.GetString("thread"))
	if err != nil {
		invalid("thread", err)
	}

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
		Thread:        thread,
		Search:        viper.GetString("search"),
		UserAgent:     viper.GetString("ua"),
		NoDownload:    viper.GetBool("no-download"),
		NoUpload:      viper.GetBool("no-upload"),
		PingMode:      pingMode,
		Unit:          unit,
		Debug:         viper.GetBool("debug"),
		BaseURL:       viper.GetString("base-url"),
	}

	if cfg.BaseURL != "" {
		_, err = speedtest.ParseBaseURL(cfg.BaseURL)
		if err != nil {
			invalid("base-url", err)
		}
	}

	err = checkOutputFormats(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

// parseThread parses the connection count.
//
// Parameters:
//   - raw: the setting value. Empty means zero, which lets the library pick.
//
// Returns:
//   - int: the connection count.
//   - error: the parse error, or an error for a negative count.
func parseThread(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	thread, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is %w", raw, errNotWholeNumber)
	}

	if thread < 0 {
		return 0, fmt.Errorf(
			"%d is %w: use 0 for the default or a positive count",
			thread,
			errNegative,
		)
	}

	return thread, nil
}

// checkOutputFormats rejects more than one of the json, jsonl, and unix output formats.
//
// Parameters:
//   - cfg: the settings to check.
//
// Returns:
//   - error: an error wrapping [ErrInvalidSetting] that names the chosen formats.
func checkOutputFormats(cfg Config) error {
	var chosen []string

	for _, format := range []struct {
		name string
		set  bool
	}{
		{name: "--json", set: cfg.JSONOutput},
		{name: "--jsonl", set: cfg.JSONLOutput},
		{name: "--unix", set: cfg.UnixOutput},
	} {
		if format.set {
			chosen = append(chosen, format.name)
		}
	}

	if len(chosen) > 1 {
		return fmt.Errorf("%w: %s cannot be combined, choose one output format",
			ErrInvalidSetting, strings.Join(chosen, ", "))
	}

	return nil
}
