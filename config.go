package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config struct represents the structure of your configuration file
type Config struct {
	JMAP     JMAPConfig
	Listmonk ListmonkConfig
}

// ServerConfig represents the server-related configurations
type JMAPConfig struct {
	Endpoint string
	Username string
	Password string
}

// DatabaseConfig represents the database-related configurations
type ListmonkConfig struct {
	BaseURL  string
	Username string
	Password string
}

func readConfig() (*Config, error) {
	var config Config
	configFile := "/etc/subscribe-responder.toml"

	// Open the config file
	if _, err := os.Stat(configFile); err != nil {
		return nil, fmt.Errorf("Config file not found: %w", err)
	}

	// Parse the TOML file
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("Error decoding TOML file: %w", err)
	}

	return &config, nil
}
