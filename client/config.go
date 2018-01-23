package client

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	ListenAddr string `yaml:"listenaddr"`
	ListenPort int    `yaml:"listenport"`
	DOHURL     string `yaml:"dohurl"`
}

// LoadConfig loads a config from a byte buffer
func LoadConfig(buf []byte) (Config, error) {
	cfg := Config{}
	err := yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("Fuck! unable to unmarshal: %v", err)
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = 13000
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1"
	}
	if cfg.DOHURL == "" {
		cfg.DOHURL = "http://127.0.0.1"
	}

	return cfg, nil
}
