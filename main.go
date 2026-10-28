package main

import (
	"flag"
	"fmt"
	"log"
	"mockapi/handler"
	"mockapi/store"
	"mockapi/ui"
	"net/http"
	"os"
	"strconv"
)

func main() {
	// Defaults: read from env vars first, then allow CLI flags to override.
	defaultPort := 9000
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			defaultPort = n
		}
	}

	defaultCap := 100
	if v := os.Getenv("CAP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			defaultCap = n
		}
	}

	port := flag.Int("port", defaultPort, "port to listen on")
	capFlag := flag.Int("cap", defaultCap, "global request cap")
	verbose := flag.Bool("verbose", false, "log request headers and body")
	flag.Parse()

	st := store.New(*capFlag)

	mux := http.NewServeMux()

	// GET / → redirect to dashboard.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/_/", http.StatusFound)
	})

	// Dashboard API routes.
	mux.Handle("/_/api/", handler.NewAPIHandler(st))

	// Embedded UI.
	mux.Handle("/_/", http.StripPrefix("/_/", ui.Handler()))

	// Capture handler — catch-all for slug-based routes.
	mux.Handle("/", handler.CaptureLogger(handler.ServeCapture(st), *verbose))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("MockAPI listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
