package config

import (
	"os"

	"github.com/danielkrainas/canaria-api/logging"
)

type StorageConfig struct {
	Driver string
}

type Config struct {
	Storage *StorageConfig
}

func newConfig() *Config {
	config := &Config{}
	config.Storage = &StorageConfig{}
	config.Storage.Driver = "memory"
	return config
}

func validKey(key string, value string) bool {
	if value == "" {
		logging.Trace.Printf("WARNING: key %s should not be empty", key)
		return false
	}

	return true
}

func LoadConfig() (*Config, error) {
	config := newConfig()

	if config.Storage != nil {
		driverName := os.Getenv("CANARY_STORAGE")
		if validKey("CANARY_STORAGE", driverName) {
			config.Storage.Driver = driverName
		}
	}

	return config, nil
}
