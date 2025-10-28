package util

import "os"

// EnvOrDefault returns the environment variable value or fallback when it is empty.
func EnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
