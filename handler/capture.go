package handler

import (
	"crypto/rand"
	"fmt"
	"io"
	"mockapi/store"
	"net/http"
	"strings"
	"time"
)

// newUUID generates a UUID v4 string using crypto/rand.
func newUUID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	// Set version 4 (bits 12-15 of byte 6).
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits (bits 6-7 of byte 8).
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ServeCapture returns an http.Handler that captures inbound webhook requests.
// It expects the first path segment to be a valid channel slug.
func ServeCapture(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the first path segment as the slug.
		path := r.URL.Path
		// Strip leading slash.
		trimmed := strings.TrimPrefix(path, "/")
		// Split on "/" to get the first segment.
		parts := strings.SplitN(trimmed, "/", 2)
		slug := parts[0]

		// Validate the slug.
		if err := store.ValidateSlug(slug); err != nil {
			http.Error(w, "invalid channel slug: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get or create the channel.
		st.GetOrCreateChannel(slug)

		// Read full body.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}

		// Build and store the request.
		req := &store.Request{
			ID:          newUUID(),
			ChannelSlug: slug,
			ReceivedAt:  time.Now().UTC(),
			Method:      r.Method,
			Path:        r.URL.Path,
			Query:       r.URL.Query(),
			Headers:     r.Header.Clone(),
			Body:        body,
			ContentType: r.Header.Get("Content-Type"),
			BodySize:    len(body),
		}
		st.AddRequest(req)

		// Look up the configured response rule for this channel/method/path.
		status, body := st.GetResponseRule(slug, r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}
