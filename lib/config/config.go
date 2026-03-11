package config

import "github.com/jaredwarren/clock/internal/config"

type Config = config.Config
type TickConfig = config.TickConfig
type NumConfig = config.NumConfig

var DefaultConfig = config.DefaultConfig

func ReadConfig(filepath string) (*Config, error) {
	return config.ReadConfig(filepath)
}

func WriteConfig(filepath string, c *Config) error {
	return config.WriteConfig(filepath, c)
}
