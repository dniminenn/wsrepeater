package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	sunCacheTicker = 5 * time.Minute
)

func StartSunPrefetcher() {
	prefetchSunriseSunset()

	ticker := time.NewTicker(sunCacheTicker)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			prefetchSunriseSunset()
		}
	}
}

func prefetchSunriseSunset() {
	_, err := fetchAndCacheSunriseSunset()
	if err != nil {
		log.Printf("Error prefetching sunrise-sunset data: %v", err)
	}
	fmt.Println("Sunrise-Sunset data prefetched successfully")
}

func fetchAndCacheSunriseSunset() ([]byte, error) {
	observationsResponse, err := getCached1DayObservations()
	if err != nil {
		return nil, fmt.Errorf("error getting 1-day observations data: %v", err)
	}

	observations := observationsResponse["observations"].([]interface{})
	latestObservation := observations[len(observations)-1].(map[string]interface{})
	latitude := fmt.Sprintf("%f", latestObservation["lat"].(float64))
	longitude := fmt.Sprintf("%f", latestObservation["lon"].(float64))
	timezone := latestObservation["tz"].(string)

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("error loading timezone from observation data: %v", err)
	}

	sunURL := fmt.Sprintf("https://api.sunrise-sunset.org/json?lat=%s&lng=%s&formatted=0", latitude, longitude)
	cacheKeySun := "sunriseSunset"

	cacheMutex.Lock()
	item, found := cache[cacheKeySun]
	cacheMutex.Unlock()

	if found && time.Now().Before(item.expiryTime) {
		return item.content, nil
	}

	resp, err := http.Get(sunURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching sunrise-sunset data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status from sunrise-sunset API: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading sunrise-sunset API response: %v", err)
	}

	// Calculate local midnight in the weather station's time zone
	now := time.Now().In(loc)
	localMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	ttl := time.Until(localMidnight)

	// Store in cache with expiration at local midnight
	cacheMutex.Lock()
	cache[cacheKeySun] = cachedItem{
		content:    body,
		expiryTime: now.Add(ttl),
	}
	cacheMutex.Unlock()

	return body, nil
}

// ProxySunriseSunset handles the request to fetch the sunrise and sunset data
func ProxySunriseSunset(w http.ResponseWriter, r *http.Request) {
	data, err := fetchAndCacheSunriseSunset()
	if err != nil {
		log.Printf("Error fetching sunrise-sunset data: %v", err)
		http.Error(w, "Failed to fetch sunrise-sunset data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
