package main

import (
	"log"
	"net/http"
	"wsrepeater/internal/config"
	"wsrepeater/internal/handlers"
)

func main() {
	// Load environment variables from .env file if not already set
	config.LoadConfig()

	http.HandleFunc("/ecowitt/report", handlers.ConvertAndForward)
	http.HandleFunc("/latest", handlers.GetLatestData)

	go handlers.StartWorkerPool()

	log.Println("Starting server on :5000")
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
