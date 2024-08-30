package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	WuHitCounter      uint64
	wuCacheTTL        = 2 * time.Minute
	wuHistoryCacheTTL time.Time
)

func StartWUPrefetcher() {
	prefetch()

	ticker := time.NewTicker(wuCacheTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			prefetch()
		}
	}
}

func prefetch() {
	_, err := getCached1DayObservations()
	if err != nil {
		log.Printf("Error prefetching 1-day observations data: %v", err)
	}

	_, err = getCached7DayHistory()
	if err != nil {
		log.Printf("Error prefetching 7-day history data: %v", err)
	}
}

func ProxyWUToday(w http.ResponseWriter, r *http.Request) {
	stationID := os.Getenv("WUNDERGROUND_ID")
	observationsResponse, err := getCached1DayObservations()
	if err != nil {
		log.Printf("Error getting 1-day observations data: %v", err)
		http.Error(w, "Failed to fetch 1-day observations data", http.StatusInternalServerError)
		return
	}

	observations := observationsResponse["observations"].([]interface{})

	var latestEpoch int64
	var latestQcStatus int
	extremes := map[string]float64{
		"tempHigh":           -9999,
		"tempLow":            9999,
		"windspeedHigh":      -9999,
		"windspeedLow":       9999,
		"windgustHigh":       -9999,
		"windgustLow":        9999,
		"dewptHigh":          -9999,
		"dewptLow":           9999,
		"pressureMax":        -9999,
		"pressureMin":        9999,
		"humidityHigh":       -9999,
		"humidityLow":        9999,
		"uvHigh":             -9999,
		"solarRadiationHigh": -9999,
	}

	for _, obs := range observations {
		observation := obs.(map[string]interface{})
		imperial := observation["imperial"].(map[string]interface{})

		epoch := int64(observation["epoch"].(float64))
		qcStatus := int(observation["qcStatus"].(float64))
		if epoch > latestEpoch {
			latestEpoch = epoch
			latestQcStatus = qcStatus
		}

		updateExtremes(extremes, imperial, "tempHigh", "tempLow")
		updateExtremes(extremes, imperial, "windspeedHigh", "windspeedLow")
		updateExtremes(extremes, imperial, "windgustHigh", "windgustLow")
		updateExtremes(extremes, imperial, "dewptHigh", "dewptLow")
		updateExtremes(extremes, imperial, "pressureMax", "pressureMin")
		updateExtremes(extremes, observation, "humidityHigh", "humidityLow")
		updateExtremeValue(extremes, observation, "uvHigh")
		updateExtremeValue(extremes, observation, "solarRadiationHigh")
	}

	response := map[string]interface{}{
		"dailyHistory": map[string]interface{}{
			"observations": []interface{}{
				map[string]interface{}{
					"stationID":    stationID,
					"tz":           observations[0].(map[string]interface{})["tz"],
					"obsTimeUtc":   time.Unix(latestEpoch, 0).UTC().Format(time.RFC3339),
					"obsTimeLocal": time.Unix(latestEpoch, 0).Format("2006-01-02 15:04:05"),
					"epoch":        latestEpoch,
					"qcStatus":     latestQcStatus,
					"lat":          observations[0].(map[string]interface{})["lat"],
					"lon":          observations[0].(map[string]interface{})["lon"],
					"imperial": map[string]interface{}{
						"tempHigh":      extremes["tempHigh"],
						"tempLow":       extremes["tempLow"],
						"windspeedHigh": extremes["windspeedHigh"],
						"windspeedLow":  extremes["windspeedLow"],
						"windgustHigh":  extremes["windgustHigh"],
						"windgustLow":   extremes["windgustLow"],
						"dewptHigh":     extremes["dewptHigh"],
						"dewptLow":      extremes["dewptLow"],
						"pressureMax":   extremes["pressureMax"],
						"pressureMin":   extremes["pressureMin"],
					},
					"humidityHigh":       extremes["humidityHigh"],
					"humidityLow":        extremes["humidityLow"],
					"uvHigh":             extremes["uvHigh"],
					"solarRadiationHigh": extremes["solarRadiationHigh"],
				},
			},
		},
		"allObservations": observations,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Failed to prepare response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}

func getCached1DayObservations() (map[string]interface{}, error) {
	cacheKey := "wutoday"
	cacheMutex.Lock()
	item, found := cache[cacheKey]
	cacheMutex.Unlock()

	if !found || time.Now().After(item.expiryTime) {
		atomic.AddUint64(&WuHitCounter, 1) // Increment the WU hit counter

		stationID := os.Getenv("WUNDERGROUND_ID")
		apiKey := os.Getenv("WUNDERGROUND_API_KEY")
		observationsURL := fmt.Sprintf("https://api.weather.com/v2/pws/observations/all/1day?stationId=%s&format=json&units=e&apiKey=%s&numericPrecision=decimal", stationID, apiKey)

		resp, err := http.Get(observationsURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching 1-day observations data: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK HTTP status: %v", resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading 1-day observations response: %v", err)
		}

		var observationsResponse map[string]interface{}
		if err := json.Unmarshal(body, &observationsResponse); err != nil {
			return nil, fmt.Errorf("error parsing 1-day observations JSON response: %v", err)
		}

		cacheMutex.Lock()
		cache[cacheKey] = cachedItem{
			content:    body,
			expiryTime: time.Now().Add(wuCacheTTL),
		}
		cacheMutex.Unlock()

		return observationsResponse, nil
	}

	var observationsResponse map[string]interface{}
	if err := json.Unmarshal(item.content, &observationsResponse); err != nil {
		return nil, fmt.Errorf("error parsing cached 1-day observations JSON response: %v", err)
	}

	return observationsResponse, nil
}

func updateExtremes(extremes map[string]float64, data map[string]interface{}, highKey, lowKey string) {
	high := data[highKey].(float64)
	low := data[lowKey].(float64)

	if high > extremes[highKey] {
		extremes[highKey] = high
	}
	if low < extremes[lowKey] {
		extremes[lowKey] = low
	}
}

func updateExtremeValue(extremes map[string]float64, data map[string]interface{}, key string) {
	value := data[key].(float64)

	if value > extremes[key] {
		extremes[key] = value
	}
}

func ProxyWUHistory(w http.ResponseWriter, r *http.Request) {
	historyResponse, err := getCached7DayHistory()
	if err != nil {
		log.Printf("Error getting 7-day history data: %v", err)
		http.Error(w, "Failed to fetch 7-day history data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(historyResponse)
}

func getCached7DayHistory() ([]byte, error) {
	cacheKeyToday := "wuTodayHistory"
	cacheKeyRest := "wuRestHistory"

	cacheMutex.Lock()
	itemToday, foundToday := cache[cacheKeyToday]
	cacheMutex.Unlock()

	var todayData []interface{}
	var baseEpoch int64
	var localTz *time.Location
	if !foundToday || time.Now().After(itemToday.expiryTime) {
		stationID := os.Getenv("WUNDERGROUND_ID")
		apiKey := os.Getenv("WUNDERGROUND_API_KEY")
		todayURL := fmt.Sprintf("https://api.weather.com/v2/pws/observations/all/1day?stationId=%s&format=json&units=m&apiKey=%s&numericPrecision=decimal", stationID, apiKey)

		resp, err := http.Get(todayURL)
		atomic.AddUint64(&WuHitCounter, 1)
		if err != nil {
			return nil, fmt.Errorf("error fetching today's observations data: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK HTTP status for today's data: %v", resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading today's observations response: %v", err)
		}

		var dayData map[string]interface{}
		if err := json.Unmarshal(body, &dayData); err != nil {
			return nil, fmt.Errorf("error parsing today's JSON response: %v", err)
		}

		if obs, ok := dayData["observations"].([]interface{}); ok {
			todayData = obs
			if len(todayData) > 0 {
				baseEpoch = int64(todayData[0].(map[string]interface{})["epoch"].(float64)) // Get the base epoch
				timezone := dayData["observations"].([]interface{})[0].(map[string]interface{})["tz"].(string)

				// Load the location based on the timezone provided by WU
				localTz, err = time.LoadLocation(timezone)
				if err != nil {
					return nil, fmt.Errorf("error loading station timezone: %v", err)
				}
			} else {
				// Handle the case where there is no data yet for the new day
				// Fallback to the previous day
				baseEpoch = time.Now().Add(-24 * time.Hour).Unix()
				localTz, _ = time.LoadLocation("UTC") // Default to UTC if timezone is not available
			}
		} else {
			return nil, fmt.Errorf("error: observations data is not a slice")
		}

		cacheMutex.Lock()
		cache[cacheKeyToday] = cachedItem{
			content:    body,
			expiryTime: time.Now().Add(5 * time.Minute),
		}
		cacheMutex.Unlock()
	} else {
		var dayData map[string]interface{}
		if err := json.Unmarshal(itemToday.content, &dayData); err != nil {
			return nil, fmt.Errorf("error parsing cached today's JSON response: %v", err)
		}

		if obs, ok := dayData["observations"].([]interface{}); ok {
			todayData = obs
			if len(todayData) > 0 {
				baseEpoch = int64(todayData[0].(map[string]interface{})["epoch"].(float64)) // Get the base epoch
				timezone := dayData["observations"].([]interface{})[0].(map[string]interface{})["tz"].(string)

				localTz, _ = time.LoadLocation(timezone)
			} else {
				// Handle the case where there is no data yet for the new day
				// Fallback to the previous day
				baseEpoch = time.Now().Add(-24 * time.Hour).Unix()
				localTz, _ = time.LoadLocation("UTC") // Default to UTC if timezone is not available
			}
		} else {
			return nil, fmt.Errorf("error: cached observations data is not a slice")
		}
	}

	// Fetch the rest of the days (indexes 1 to 6) using the historical endpoint
	cacheMutex.Lock()
	itemRest, foundRest := cache[cacheKeyRest]
	cacheMutex.Unlock()

	var restOfTheWeekData [][]interface{}
	if !foundRest || time.Now().After(itemRest.expiryTime) {
		restOfTheWeekData = make([][]interface{}, 6)
		stationID := os.Getenv("WUNDERGROUND_ID")
		apiKey := os.Getenv("WUNDERGROUND_API_KEY")

		for i := 1; i <= 6; i++ {
			atomic.AddUint64(&WuHitCounter, 1)

			// Subtract one day in seconds (86400 seconds in a day) from the base epoch for each previous day
			requestEpoch := baseEpoch - int64(i*86400)
			requestDate := time.Unix(requestEpoch, 0).In(localTz).Format("20060102")

			historyURL := fmt.Sprintf("https://api.weather.com/v2/pws/history/all?stationId=%s&format=json&units=m&date=%s&apiKey=%s&numericPrecision=decimal", stationID, requestDate, apiKey)

			resp, err := http.Get(historyURL)
			if err != nil {
				return nil, fmt.Errorf("error fetching history data for date %s: %v", requestDate, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("received non-OK HTTP status for date %s: %v", requestDate, resp.Status)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("error reading history response for date %s: %v", requestDate, err)
			}

			var dayData map[string]interface{}
			if err := json.Unmarshal(body, &dayData); err != nil {
				return nil, fmt.Errorf("error parsing history JSON response for date %s: %v", requestDate, err)
			}

			if obs, ok := dayData["observations"].([]interface{}); ok {
				restOfTheWeekData[i-1] = obs
			} else {
				return nil, fmt.Errorf("error: observations data for date %s is not a slice", requestDate)
			}
		}

		responseBody, err := json.Marshal(restOfTheWeekData)
		if err != nil {
			return nil, fmt.Errorf("error marshaling rest of the week history response: %v", err)
		}

		// Calculate local midnight for the station's timezone using the baseEpoch
		baseTime := time.Unix(baseEpoch, 0).In(localTz)
		midnight := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day()+1, 0, 0, 0, 0, localTz)

		cacheMutex.Lock()
		cache[cacheKeyRest] = cachedItem{
			content:    responseBody,
			expiryTime: midnight,
		}
		cacheMutex.Unlock()
	} else {
		if err := json.Unmarshal(itemRest.content, &restOfTheWeekData); err != nil {
			return nil, fmt.Errorf("error parsing cached rest of the week JSON response: %v", err)
		}
	}

	// Combine today's data with the rest of the week
	weeklyData := make([][]interface{}, 7)
	weeklyData[0] = todayData

	for i := 1; i < 7; i++ {
		weeklyData[i] = restOfTheWeekData[i-1]
	}

	finalData := map[string]interface{}{
		"weeklyData": weeklyData,
	}

	finalResponse, err := json.Marshal(finalData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling final combined history response: %v", err)
	}

	return finalResponse, nil
}
