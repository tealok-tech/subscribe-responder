package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config struct represents the structure of your configuration file
type Config struct {
	JMAP                  JMAPConfig
	Listmonk              ListmonkConfig
	SubscriptionResponder SubscriptionResponderConfig
}

// ServerConfig represents the server-related configurations
type JMAPConfig struct {
	Endpoint string
	Username string
	Password string
}

// DatabaseConfig represents the database-related configurations
type ListmonkConfig struct {
	// The base URL for listmonk, like https://mailing-list.domain.org
	BaseURL string
	// The list that new subscribers are added to. We poll for subscribers on this
	// list, send them the transactional email, then move their subscription to the TargetList.
	NewSubscriberList uint
	// The password to authenticate to listmonk
	Password string
	// The list that contains our actual content. Subscriptions get moved here after our welcome email
	// and stay here over the long haul.
	TargetList uint
	// The template ID for the transactional email to send to new subscribers
	TransactionalTemplateID uint
	// The username to authenticate to listmonk
	Username string
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

type SubscriptionResponderConfig struct {
	// When present, only respond to requests from emails that match the provided regular expression.
	EmailFilterRegex string
}
