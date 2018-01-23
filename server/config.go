package server

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

// Config is the Server Config
type Config struct {
	ListenAddr string `yaml:"listenaddr"`
	ListenPort int    `yaml:"listenport"`
	// Resolvers are a list of IPs to send DNS queries to
	Resolvers []string `yaml:"resolvers"`
}

// LoadConfig loads a config from a byte buffer
func LoadConfig(buf []byte) (Config, error) {
	cfg := Config{}
	err := yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("Fuck! unable to unmarshal: %v", err)
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = 9000
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "0.0.0.0"
	}
	if len(cfg.Resolvers) == 0 {
		cfg.Resolvers = []string{"8.8.8.8", "8.8.4.4"}
	}

	return cfg, nil
}
