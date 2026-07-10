// Package config provides tiny, dependency-free helpers for reading
// configuration from the environment. Each service builds its own typed
// config struct on top of these primitives.
package config

import (
	"os"
	"strconv"
)

// Get returns the value of key, or fallback when unset or empty.
func Get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// GetInt returns the value of key parsed as int, or fallback.
func GetInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
