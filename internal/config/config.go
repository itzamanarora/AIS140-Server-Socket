package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings for the AIS-140 TCP capture server.
type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	ReadBuffer   int
}

// Addr returns the host:port listen address.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Load parses flags and environment variables into a Config.
// Precedence: CLI flags > environment variables > defaults.
func Load() Config {
	cfg := Config{
		Host:         envOr("AIS140_HOST", ""),
		Port:         envIntOr("AIS140_PORT", 5001),
		ReadTimeout:  envDurationOr("AIS140_READ_TIMEOUT", 5*time.Minute),
		WriteTimeout: envDurationOr("AIS140_WRITE_TIMEOUT", 30*time.Second),
		ReadBuffer:   envIntOr("AIS140_READ_BUFFER", 4096),
	}

	flag.StringVar(&cfg.Host, "host", cfg.Host, "TCP listen host (empty = all interfaces)")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "TCP listen port")
	flag.DurationVar(&cfg.ReadTimeout, "read-timeout", cfg.ReadTimeout, "per-connection read idle timeout")
	flag.DurationVar(&cfg.WriteTimeout, "write-timeout", cfg.WriteTimeout, "per-connection write timeout (reserved for future use)")
	flag.IntVar(&cfg.ReadBuffer, "read-buffer", cfg.ReadBuffer, "socket read buffer size in bytes")
	flag.Parse()

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
