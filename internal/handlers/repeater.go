package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"wsrepeater/internal/utils"
)

const wundergroundURL = "http://weatherstation.wunderground.com/weatherstation/updateweatherstation.php"
const movingAverageWindow = 5
const workerCount = 5

var (
	uvValues             []float64
	solarRadiationValues []float64
	uvMutex              sync.Mutex
	solarMutex           sync.Mutex
	latestData           map[string]string
	dataMutex            sync.Mutex
	jobQueue             = make(chan url.Values, 100)
)

func ConvertAndForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	ecowittData, err := url.ParseQuery(string(body))
	if err != nil {
		log.Printf("Error parsing query: %v", err)
		http.Error(w, "can't parse body", http.StatusBadRequest)
		return
	}

	uvValue, err := strconv.ParseFloat(ecowittData.Get("uv"), 64)
	if err != nil {
		log.Printf("Error parsing UV value: %v", err)
		http.Error(w, "can't parse UV value", http.StatusBadRequest)
		return
	}

	solarRadiationValue, err := strconv.ParseFloat(ecowittData.Get("solarradiation"), 64)
	if err != nil {
		log.Printf("Error parsing solar radiation value: %v", err)
		http.Error(w, "can't parse solar radiation value", http.StatusBadRequest)
		return
	}

	tempF, err := strconv.ParseFloat(ecowittData.Get("tempf"), 64)
	if err != nil {
		log.Printf("Error parsing temperature value: %v", err)
		http.Error(w, "can't parse temperature value", http.StatusBadRequest)
		return
	}

	humidity, err := strconv.ParseFloat(ecowittData.Get("humidity"), 64)
	if err != nil {
		log.Printf("Error parsing humidity value: %v", err)
		http.Error(w, "can't parse humidity value", http.StatusBadRequest)
		return
	}

	windSpeedValue, err := strconv.ParseFloat(ecowittData.Get("windspeedmph"), 64)
	if err != nil {
		log.Printf("Error parsing wind speed value: %v", err)
		http.Error(w, "can't parse wind speed value", http.StatusBadRequest)
		return
	}

	tempC := (tempF - 32) * 5 / 9
	dewPointC := utils.CalculateDewPoint(tempC, humidity)
	dewPointF := dewPointC*9/5 + 32

	smoothedUV := utils.SmoothValue(uvValue, &uvValues, &uvMutex)
	smoothedSolarRadiation := utils.SmoothValue(solarRadiationValue, &solarRadiationValues, &solarMutex)

	correctedUV := math.Round(smoothedUV * 0.94)
	correctedSolarRadiation := smoothedSolarRadiation * 0.94

	wundergroundID := os.Getenv("WUNDERGROUND_ID")
	wundergroundPW := os.Getenv("WUNDERGROUND_PASS")
	stationSoftware := os.Getenv("STATION_SOFTWARE")

	wundergroundData := url.Values{}
	wundergroundData.Set("ID", wundergroundID)
	wundergroundData.Set("PASSWORD", wundergroundPW)
	wundergroundData.Set("dateutc", ecowittData.Get("dateutc"))
	wundergroundData.Set("tempf", ecowittData.Get("tempf"))
	wundergroundData.Set("humidity", ecowittData.Get("humidity"))
	wundergroundData.Set("dewptf", fmt.Sprintf("%.2f", dewPointF))
	wundergroundData.Set("windspeedmph", fmt.Sprintf("%.2f", windSpeedValue))
	wundergroundData.Set("windgustmph", ecowittData.Get("windgustmph"))
	wundergroundData.Set("winddir", ecowittData.Get("winddir"))
	wundergroundData.Set("solarradiation", fmt.Sprintf("%.2f", correctedSolarRadiation))
	wundergroundData.Set("UV", fmt.Sprintf("%d", int(correctedUV)))
	wundergroundData.Set("baromin", ecowittData.Get("baromrelin"))
	wundergroundData.Set("absbaromin", ecowittData.Get("baromabsin"))
	wundergroundData.Set("rainin", ecowittData.Get("rainratein"))
	wundergroundData.Set("dailyrainin", ecowittData.Get("dailyrainin"))
	wundergroundData.Set("weeklyrainin", ecowittData.Get("weeklyrainin"))
	wundergroundData.Set("monthlyrainin", ecowittData.Get("monthlyrainin"))
	wundergroundData.Set("yearlyrainin", ecowittData.Get("yearlyrainin"))
	wundergroundData.Set("indoortempf", ecowittData.Get("tempinf"))
	wundergroundData.Set("indoorhumidity", ecowittData.Get("humidityin"))
	wundergroundData.Set("softwaretype", stationSoftware)
	wundergroundData.Set("realtime", "1")
	wundergroundData.Set("rtfreq", ecowittData.Get("interval"))
	wundergroundData.Set("action", "updateraw")

	go updateLatestData(ecowittData)

	jobQueue <- wundergroundData

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Data accepted for processing"))
}

func updateLatestData(data url.Values) {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	latestData = make(map[string]string)
	for key, values := range data {
		if len(values) > 0 {
			latestData[key] = values[0]
		}
	}
}

func GetLatestData(w http.ResponseWriter, r *http.Request) {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestData)
}

func GetLatestDataWithCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	dataMutex.Lock()
	defer dataMutex.Unlock()

	json.NewEncoder(w).Encode(latestData)
}

func StartWorkerPool() {
	for i := 0; i < workerCount; i++ {
		go worker()
	}
}

func worker() {
	for job := range jobQueue {
		resp, err := http.PostForm(wundergroundURL, job)
		if err != nil {
			log.Printf("Error forwarding to Wunderground: %v", err)
			continue
		}
		defer resp.Body.Close()

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading Wunderground response: %v", err)
			continue
		}

		// if respBody is anything other than "success", log it
		if !strings.Contains(string(respBody), "success") {
			log.Printf("Wunderground response: %s", respBody)
		}
	}
}
