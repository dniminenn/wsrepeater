package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfig() {
	// Load the .env file if environment variables are not set
	err := godotenv.Load()
	if err != nil && !fileExists(".env") {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// List of required environment variables
	requiredEnvVars := []string{
		"WUNDERGROUND_ID",
		"WUNDERGROUND_PASS",
		"STATION_SOFTWARE",
		"WUNDERGROUND_API_KEY",
		"ASTRO_API_KEY",
	}

	// Check if all required environment variables are set
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Environment variable %s is not set", envVar)
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
