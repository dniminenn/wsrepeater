package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
	"wsrepeater/internal/config"
	"wsrepeater/internal/handlers"
	"wsrepeater/internal/middleware"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	config.LoadConfig()

	stats := middleware.NewStats()

	cacheDurations := map[string]time.Duration{
		"/":                     120 * time.Minute,
		"/stats":                0 * time.Second,
		"/latest":               1 * time.Minute,
		"/weekly":               5 * time.Minute,
		"/moon":                 20 * time.Minute,
		"/wutoday":              5 * time.Minute,
		"/rss/nb10_e.xml":       5 * time.Minute,
		"/rss/nb16_e.xml":       5 * time.Minute,
		"/rss/city/nb-17_e.xml": 5 * time.Minute,
	}

	defaultCacheDuration := 1 * time.Minute
	staticCacheDuration := 24 * time.Hour

	mux := http.NewServeMux()
	mux.HandleFunc("/ecowitt/report", handlers.ConvertAndForward)                // Ingest data from ecowitt, forward to WeatherUnderground
	mux.HandleFunc("/latest", handlers.GetLatestDataWithCORS)                    // Serve latest data to the frontend
	mux.Handle("/rss/nb10_e.xml", http.HandlerFunc(handlers.ProxyRSSFeed))       // Proxy Environment Canada RSS feed
	mux.Handle("/rss/nb16_e.xml", http.HandlerFunc(handlers.ProxyRSSFeed))       // Proxy Environment Canada RSS feed
	mux.Handle("/rss/city/nb-17_e.xml", http.HandlerFunc(handlers.ProxyRSSFeed)) // Proxy Environment Canada RSS feed
	mux.HandleFunc("/wutoday", handlers.ProxyWUToday)                            // Today's observations from WeatherUnderground
	mux.HandleFunc("/weekly", handlers.ProxyWUHistory)                           // Weekly observations from WeatherUnderground
	mux.HandleFunc("/moon", handlers.ProxyMoon)                                  // Moon phase logic
	mux.HandleFunc("/sunrise-sunset", handlers.ProxySunriseSunset)               // Sunrise and sunset times
	mux.Handle("/", http.FileServer(getStaticFiles()))                           // Serve static files for the frontend
	mux.HandleFunc("/stats", stats.ServeStats)

	handler := middleware.GzipMiddleware(stats.Middleware(middleware.CacheControl(cacheDurations,
		defaultCacheDuration, staticCacheDuration)(mux)))

	go handlers.StartWorkerPool()
	go handlers.StartWUPrefetcher()
	go handlers.StartMoonPrefetcher()
	go handlers.StartRSSPrefetcher()
	go handlers.StartSunPrefetcher()

	log.Println("Starting server on :5000")
	if err := http.ListenAndServe(":5000", handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getStaticFiles() http.FileSystem {
	// Check if the ./html directory exists
	if _, err := os.Stat("./html"); !os.IsNotExist(err) {
		log.Println("Serving static files from ./html directory")
		return http.Dir("./html")
	}

	// Fallback to embedded static files
	log.Println("Serving static files from embedded filesystem")
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	return http.FS(fsys)
}
