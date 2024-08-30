package middleware

import (
	"net/http"
	"strconv"
	"time"
	"wsrepeater/internal/utils"
)

func CacheControl(cacheDurations map[string]time.Duration, defaultDuration time.Duration, staticDuration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Determine cache duration based on the request path
			duration := defaultDuration

			if utils.HasExtension(path) {
				duration = staticDuration
			} else if d, found := cacheDurations[path]; found {
				duration = d
			}

			setCacheControl(w, duration)
			next.ServeHTTP(w, r)
		})
	}
}

func setCacheControl(w http.ResponseWriter, duration time.Duration) {
	maxAge := int(duration.Seconds())
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
}
