package utils

import (
	"math"
	"path/filepath"
	"strings"
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

// MoonPhaseFromAngle determines the moon phase based on the angle of the moon.
func MoonPhaseFromAngle(angle float64) string {
	// Normalize angle to 0-360 range
	for angle < 0 {
		angle += 360
	}
	angle = math.Mod(angle, 360)

	// Define phases with a margin for angle transition
	switch {
	case angle >= 350 || angle <= 10:
		return "New Moon"
	case angle > 10 && angle < 80:
		return "Waxing Crescent"
	case angle >= 80 && angle <= 100:
		return "First Quarter"
	case angle > 100 && angle < 170:
		return "Waxing Gibbous"
	case angle >= 170 && angle <= 190:
		return "Full Moon"
	case angle > 190 && angle < 260:
		return "Waning Gibbous"
	case angle >= 260 && angle <= 280:
		return "Last Quarter"
	case angle > 280 && angle < 350:
		return "Waning Crescent"
	default:
		return "Unknown Phase"
	}
}

// CalculateMoonIllumination calculates the illumination percentage of the moon.
func CalculateMoonIllumination(angle float64) float64 {
	return (1 - math.Cos(angle*math.Pi/180)) / 2 * 100
}

func HasExtension(path string) bool {
	ext := filepath.Ext(path)
	return len(ext) > 0 && !strings.Contains(ext[1:], "/") && ext != ".xml"
}
