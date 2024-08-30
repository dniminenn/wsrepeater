package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"wsrepeater/internal/utils"
)

var (
	moonCacheTTL = 1 * time.Hour
)

func StartMoonPrefetcher() {
	fmt.Printf("Starting Moon prefetcher\n")

	// Perform an initial prefetch right away
	prefetchMoonData()

	// Set up a ticker to run the prefetch function at regular intervals
	ticker := time.NewTicker(moonCacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			prefetchMoonData()
		}
	}
}

func prefetchMoonData() {
	responseBody, err := fetchMoonData()
	if err != nil {
		log.Printf("Error prefetching moon data: %v", err)
		return
	}

	// Store in cache with the defined TTL
	cacheMutex.Lock()
	cache["moon"] = cachedItem{
		content:    responseBody,
		expiryTime: time.Now().Add(moonCacheTTL),
	}
	cacheMutex.Unlock()

	fmt.Println("Moon data prefetched successfully")
}

// ProxyMoon handles the request to fetch the moon phase and illumination data
func ProxyMoon(w http.ResponseWriter, r *http.Request) {
	cacheKeyMoon := "moon"

	// Check the cache for moon data
	cacheMutex.Lock()
	item, found := cache[cacheKeyMoon]
	cacheMutex.Unlock()

	if found && time.Now().Before(item.expiryTime) {
		// Serve from cache
		w.Header().Set("Content-Type", "application/json")
		w.Write(item.content)
		return
	}

	// If not found in cache or expired, fetch the data
	responseBody, err := fetchMoonData()
	if err != nil {
		log.Printf("Error fetching moon data: %v", err)
		http.Error(w, "Failed to fetch moon data", http.StatusInternalServerError)
		return
	}

	// Store in cache with the defined TTL
	cacheMutex.Lock()
	cache[cacheKeyMoon] = cachedItem{
		content:    responseBody,
		expiryTime: time.Now().Add(moonCacheTTL),
	}
	cacheMutex.Unlock()

	// Serve the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

func fetchMoonData() ([]byte, error) {
	apiKey := os.Getenv("ASTRO_API_KEY")

	observationsResponse, err := getCached1DayObservations()
	if err != nil {
		return nil, fmt.Errorf("Error getting 1-day observations data: %v", err)
	}

	// Extract the latest observation to get lat/lon
	observations := observationsResponse["observations"].([]interface{})
	latestObservation := observations[len(observations)-1].(map[string]interface{})
	latitude := fmt.Sprintf("%f", latestObservation["lat"].(float64))
	longitude := fmt.Sprintf("%f", latestObservation["lon"].(float64))
	elevation := "0" // Set elevation manually or retrieve if available

	// Set date and time
	currentDate := time.Now().Format("2006-01-02")
	currentTime := time.Now().Format("15:04:05")

	// Build the URL for the AstronomyAPI moon endpoint
	moonURL := fmt.Sprintf("https://api.astronomyapi.com/api/v2/bodies/positions?latitude=%s&longitude=%s&elevation=%s&from_date=%s&to_date=%s&time=%s", latitude, longitude, elevation, currentDate, currentDate, currentTime)

	// Create the Authorization header using Basic Authentication
	authHeader := "Basic " + apiKey

	// Fetch data from AstronomyAPI
	client := &http.Client{}
	req, err := http.NewRequest("GET", moonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request to AstronomyAPI: %v", err)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error fetching moon data from AstronomyAPI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Received non-OK HTTP status from AstronomyAPI: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading AstronomyAPI response: %v", err)
	}

	var moonData map[string]interface{}
	if err := json.Unmarshal(body, &moonData); err != nil {
		return nil, fmt.Errorf("Error parsing AstronomyAPI JSON response: %v", err)
	}

	// Extract the moon phase and illumination data
	table := moonData["data"].(map[string]interface{})["table"].(map[string]interface{})["rows"].([]interface{})
	moonEntry := table[1].(map[string]interface{})["cells"].([]interface{})[0].(map[string]interface{})

	angleStr := moonEntry["extraInfo"].(map[string]interface{})["phase"].(map[string]interface{})["angel"].(string)

	// Convert the angle to a moon phase, 0 = New Moon, 360 = Full Moon, allow a +/- 10 degree margin
	angle, err := strconv.ParseFloat(angleStr, 64)
	if err != nil {
		return nil, fmt.Errorf("Error converting moon angle to integer: %v", err)
	}
	moonPhase := utils.MoonPhaseFromAngle(angle)
	illumination := utils.CalculateMoonIllumination(angle)

	response := map[string]interface{}{
		"phase":        moonPhase,
		"angle":        angleStr,
		"illumination": illumination,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling moon response: %v", err)
	}

	return responseBody, nil
}
