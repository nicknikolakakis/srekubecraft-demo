package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Response struct {
	Message   string `json:"message"`
	Hostname  string `json:"hostname"`
	Timestamp string `json:"timestamp"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Main endpoint - returns hostname and timestamp
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		resp := Response{
			Message:   "Hello from Knative!",
			Hostname:  hostname,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
		slog.Info("request served", "path", r.URL.Path, "hostname", hostname)
	})

	// CPU burn endpoint for autoscaling demos
	// Usage: /burn?duration=5 (seconds, default 1)
	mux.HandleFunc("GET /burn", func(w http.ResponseWriter, r *http.Request) {
		durationStr := r.URL.Query().Get("duration")
		duration := 1
		if durationStr != "" {
			if d, err := strconv.Atoi(durationStr); err == nil && d > 0 && d <= 30 {
				duration = d
			}
		}

		start := time.Now()
		deadline := start.Add(time.Duration(duration) * time.Second)
		result := 0.0
		for time.Now().Before(deadline) {
			for i := 0; i < 1000; i++ {
				result += math.Sqrt(float64(i))
			}
		}

		hostname, _ := os.Hostname()
		resp := map[string]any{
			"message":    "CPU burn complete",
			"hostname":   hostname,
			"duration_s": duration,
			"elapsed_ms": time.Since(start).Milliseconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
		slog.Info("burn complete", "duration_s", duration, "hostname", hostname)
	})

	// CloudEvents receiver for Eventing demos
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		ceType := r.Header.Get("Ce-Type")
		ceSource := r.Header.Get("Ce-Source")
		ceID := r.Header.Get("Ce-Id")

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			body = map[string]any{"raw": "could not decode body"}
		}

		slog.Info("cloudevent received",
			"ce-type", ceType,
			"ce-source", ceSource,
			"ce-id", ceID,
			"body", body,
		)

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event received")
	})

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
