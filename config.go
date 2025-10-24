package gosmig

import "time"

const defaultTimeout = 10 * time.Second

type Config struct {
	Timeout time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		Timeout: defaultTimeout,
	}
}

func (c *Config) ensureDefaults() {
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
}
