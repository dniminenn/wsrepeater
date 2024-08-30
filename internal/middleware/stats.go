package middleware

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"wsrepeater/internal/handlers"
	"wsrepeater/internal/utils"
)

var startTime = time.Now()

type EndpointStats struct {
	Hits        uint64
	TotalTime   int64
	SlowestTime int64
}

type Stats struct {
	endpoints sync.Map
}

func NewStats() *Stats {
	return &Stats{}
}

func (s *Stats) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path

		// Skip stats for paths with file extensions (except XML)
		if utils.HasExtension(path) {
			next.ServeHTTP(w, r)
			return
		}

		stat, _ := s.endpoints.LoadOrStore(path, &EndpointStats{})
		epStats := stat.(*EndpointStats)

		defer func() {
			elapsed := time.Since(start).Nanoseconds()
			atomic.AddUint64(&epStats.Hits, 1)
			atomic.AddInt64(&epStats.TotalTime, elapsed)
			for {
				oldSlowest := atomic.LoadInt64(&epStats.SlowestTime)
				if elapsed <= oldSlowest || atomic.CompareAndSwapInt64(&epStats.SlowestTime, oldSlowest, elapsed) {
					break
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (s *Stats) ServeStats(w http.ResponseWriter, r *http.Request) {
	// Check if JSON output is requested
	jsonOutput := r.URL.Query().Get("json") == "true"

	// Endpoint stats
	endpointStats := make(map[string]map[string]string)
	var keys []string
	s.endpoints.Range(func(key, value interface{}) bool {
		path := key.(string)
		stat := value.(*EndpointStats)
		avgTime := time.Duration(0)
		slowestTime := time.Duration(0)
		hits := atomic.LoadUint64(&stat.Hits)
		if hits > 0 {
			totalTime := time.Duration(atomic.LoadInt64(&stat.TotalTime))
			avgTime = totalTime / time.Duration(hits)
			slowestTime = time.Duration(atomic.LoadInt64(&stat.SlowestTime))
		}
		endpointStats[path] = map[string]string{
			"Hits":                strconv.FormatUint(hits, 10),
			"AverageResponseTime": fmt.Sprintf("%.2f ms", float64(avgTime)/float64(time.Millisecond)),
			"SlowestResponseTime": fmt.Sprintf("%.2f ms", float64(slowestTime)/float64(time.Millisecond)),
		}
		keys = append(keys, path)
		return true
	})

	// Sort the keys alphabetically
	sort.Strings(keys)

	// Runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	wuHits := atomic.LoadUint64(&handlers.WuHitCounter)

	programStats := map[string]interface{}{
		"Alloc":      formatBytes(memStats.Alloc),
		"Sys":        formatBytes(memStats.Sys),
		"NumGC":      memStats.NumGC,
		"Goroutines": runtime.NumGoroutine(),
		"CgoCalls":   runtime.NumCgoCall(),
		"GOMAXPROCS": runtime.GOMAXPROCS(0),
		"NumCPU":     runtime.NumCPU(),
		"Uptime":     fmt.Sprintf("%.2f hours", time.Since(startTime).Hours()),
		"WUHits":     strconv.FormatUint(wuHits, 10),
	}

	// JSON response
	if jsonOutput {
		stats := map[string]interface{}{
			"endpoints":    endpointStats,
			"programStats": programStats,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// HTML response
	w.Header().Set("Content-Type", "text/html")
	tmpl := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Server Stats</title>
		<style>
			body { font-family: Arial, sans-serif; background-color: #282a36; color: #f8f8f2; }
			table { width: 100%; border-collapse: collapse; margin-bottom: 20px; }
			th, td { padding: 8px 12px; border: 1px solid #44475a; text-align: left; }
			th { background-color: #44475a; }
			.container { max-width: 800px; margin: 40px auto; padding: 20px; background-color: #282a36; border-radius: 8px; box-shadow: 0 0 10px rgba(0,0,0,0.5); }
			h2 { margin-top: 0; color: #bd93f9; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>Endpoints</h2>
			<table>
				<tr>
					<th>Endpoint</th>
					<th>Hits</th>
					<th>Average Response Time</th>
					<th>Slowest Response Time</th>
				</tr>
				{{ range .Keys }}
				<tr>
					<td>{{ . }}</td>
					<td>{{ index $.EndpointStats . "Hits" }}</td>
					<td>{{ index $.EndpointStats . "AverageResponseTime" }}</td>
					<td>{{ index $.EndpointStats . "SlowestResponseTime" }}</td>
				</tr>
				{{ end }}
			</table>

			<h2>Program</h2>
			<table>
				<tr><th>Alloc</th><td>{{ .ProgramStats.Alloc }}</td></tr>
				<tr><th>Sys</th><td>{{ .ProgramStats.Sys }}</td></tr>
				<tr><th>Num GC</th><td>{{ .ProgramStats.NumGC }}</td></tr>
				<tr><th>Goroutines</th><td>{{ .ProgramStats.Goroutines }}</td></tr>
				<tr><th>Cgo Calls</th><td>{{ .ProgramStats.CgoCalls }}</td></tr>
				<tr><th>GOMAXPROCS</th><td>{{ .ProgramStats.GOMAXPROCS }}</td></tr>
				<tr><th>Num CPU</th><td>{{ .ProgramStats.NumCPU }}</td></tr>
				<tr><th>Uptime</th><td>{{ .ProgramStats.Uptime }}</td></tr>
				<tr><th>WU Hits</th><td>{{ .ProgramStats.WUHits }}</td></tr>
			</table>
		</div>
	</body>
	</html>
	`

	t, err := template.New("stats").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Keys          []string
		EndpointStats map[string]map[string]string
		ProgramStats  map[string]interface{}
	}{
		Keys:          keys,
		EndpointStats: endpointStats,
		ProgramStats:  programStats,
	}

	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
