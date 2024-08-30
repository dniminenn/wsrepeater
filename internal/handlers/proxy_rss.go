package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	cache      = make(map[string]cachedItem)
	cacheMutex sync.Mutex
	cacheTTL   = 15 * time.Minute
)

type cachedItem struct {
	content    []byte
	expiryTime time.Time
}

func StartRSSPrefetcher() {
	fmt.Printf("Starting RSS prefetcher\n")

	// Perform an initial prefetch right away
	prefetchRSSFeeds()

	// Set up a ticker to run the prefetch function at regular intervals
	ticker := time.NewTicker(cacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			prefetchRSSFeeds()
		}
	}
}

func prefetchRSSFeeds() {
	feeds := []string{
		"/rss/nb10_e.xml",
		"/rss/nb16_e.xml",
		"/rss/city/nb-17_e.xml",
	}

	for _, path := range feeds {
		_, err := fetchAndCacheRSSFeed(path)
		if err != nil {
			log.Printf("Error prefetching RSS feed for %s: %v", path, err)
		}
	}
	fmt.Println("RSS feeds prefetched successfully")
}

// ProxyRSSFeed handles the RSS feed proxying and caching.
func ProxyRSSFeed(w http.ResponseWriter, r *http.Request) {
	body, err := fetchAndCacheRSSFeed(r.URL.Path)
	if err != nil {
		log.Printf("Error in ProxyRSSFeed: %v", err)
		http.Error(w, "Failed to fetch RSS feed", http.StatusInternalServerError)
		return
	}

	// Serve the content
	w.Header().Set("Content-Type", "application/xml")
	w.Write(body)
}

func fetchAndCacheRSSFeed(path string) ([]byte, error) {
	var feedURL string

	switch path {
	case "/rss/nb10_e.xml":
		feedURL = "https://weather.gc.ca/rss/battleboard/nb10_e.xml"
	case "/rss/nb16_e.xml":
		feedURL = "https://weather.gc.ca/rss/battleboard/nb16_e.xml"
	case "/rss/city/nb-17_e.xml":
		feedURL = "https://weather.gc.ca/rss/city/nb-17_e.xml"
	default:
		return nil, fmt.Errorf("invalid feed request")
	}

	cacheKey := path

	// Check the cache
	cacheMutex.Lock()
	item, found := cache[cacheKey]
	cacheMutex.Unlock()

	if found && time.Now().Before(item.expiryTime) {
		return item.content, nil
	}

	// Fetch from source
	resp, err := http.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching RSS feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading RSS feed response: %v", err)
	}

	// Store in cache
	cacheMutex.Lock()
	cache[cacheKey] = cachedItem{
		content:    body,
		expiryTime: time.Now().Add(cacheTTL),
	}
	cacheMutex.Unlock()

	return body, nil
}
