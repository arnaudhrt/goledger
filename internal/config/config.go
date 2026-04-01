package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration.
type Config struct {
	DisplayCurrency string     `toml:"display_currency"`
	AutoFetchRates  bool       `toml:"auto_fetch_rates"`
	Categories      Categories `toml:"categories"`
}

// Categories groups category lists by entry type.
type Categories struct {
	Exp []string `toml:"exp"`
	Inc []string `toml:"inc"`
	Inv []string `toml:"inv"`
}

// Default returns a Config with sensible defaults matching the spec.
func Default() Config {
	return Config{
		DisplayCurrency: "THB",
		AutoFetchRates:  true,
		Categories: Categories{
			Exp: []string{
				"housing", "housing:rent", "housing:electric", "housing:water",
				"food", "food:dining", "food:grocery", "food:coffee",
				"transport", "transport:grab", "transport:fuel",
				"fun", "fun:sub", "fun:social",
				"health", "health:gym", "health:medical",
				"bills", "bills:internet", "bills:mobile",
			},
			Inc: []string{"salary", "freelance", "other"},
			Inv: []string{"crypto", "stocks", "savings"},
		},
	}
}

// Load reads the config from ~/.spend/config.toml.
// If the file doesn't exist, it writes the default config and returns it.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	cfg := Default()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := Save(cfg); err != nil {
			return Config{}, fmt.Errorf("write default config: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to ~/.spend/config.toml.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".spend", "config.toml"), nil
}
