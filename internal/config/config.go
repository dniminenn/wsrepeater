package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfig() {
	// Check if the required environment variables are already set
	if os.Getenv("WUNDERGROUND_ID") == "" || os.Getenv("WUNDERGROUND_PASS") == "" || os.Getenv("STATION_SOFTWARE") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Error loading .env file: %v", err)
		}
	}
}
