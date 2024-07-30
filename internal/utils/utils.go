package utils

import (
	"math"
	"sync"
)

const movingAverageWindow = 5

func SmoothValue(value float64, values *[]float64, mutex *sync.Mutex) float64 {
	mutex.Lock()
	defer mutex.Unlock()

	*values = append(*values, value)
	if len(*values) > movingAverageWindow {
		*values = (*values)[1:]
	}

	sum := 0.0
	for _, v := range *values {
		sum += v
	}

	return sum / float64(len(*values))
}

func CalculateDewPoint(tempC, humidity float64) float64 {
	a := 17.27
	b := 237.7
	alpha := (a*tempC)/(b+tempC) + math.Log(humidity/100)
	dewPointC := (b * alpha) / (a - alpha)
	return dewPointC
}
