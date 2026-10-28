package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mockapi/store"
	"net/http"
	"strings"
	"time"
)

// channelJSON is the JSON representation of a Channel.
type channelJSON struct {
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// requestJSON is the JSON representation of a Request.
type requestJSON struct {
	ID          string              `json:"id"`
	ChannelSlug string              `json:"channel_slug"`
	ReceivedAt  time.Time           `json:"received_at"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Query       map[string][]string `json:"query"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"body"`
	ContentType string              `json:"content_type"`
	BodySize    int                 `json:"body_size"`
}

func toChannelJSON(ch *store.Channel) channelJSON {
	return channelJSON{
		Slug:      ch.Slug,
		CreatedAt: ch.CreatedAt,
	}
}

func toRequestJSON(r *store.Request) requestJSON {
	// Convert http.Header (map[string][]string) to plain map for JSON.
	headers := make(map[string][]string, len(r.Headers))
	for k, v := range r.Headers {
		headers[k] = v
	}
	return requestJSON{
		ID:          r.ID,
		ChannelSlug: r.ChannelSlug,
		ReceivedAt:  r.ReceivedAt,
		Method:      r.Method,
		Path:        r.Path,
		Query:       r.Query,
		Headers:     headers,
		Body:        r.Body,
		ContentType: r.ContentType,
		BodySize:    r.BodySize,
	}
}

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// NewAPIHandler returns an http.Handler for all routes under /_/api/.
func NewAPIHandler(st *store.Store) http.Handler {
	mux := http.NewServeMux()

	// GET /_/api/channels
	mux.HandleFunc("GET /_/api/channels", func(w http.ResponseWriter, r *http.Request) {
		channels := st.ListChannels()
		out := make([]channelJSON, len(channels))
		for i, ch := range channels {
			out[i] = toChannelJSON(ch)
		}
		writeJSON(w, http.StatusOK, out)
	})

	// POST /_/api/channels
	mux.HandleFunc("POST /_/api/channels", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Slug string `json:"slug"`
		}
		// Always attempt JSON decoding. ContentLength may be -1 for chunked
		// transfers, so checking ContentLength == 0 alone is insufficient.
		// io.EOF means the body was empty, which is valid (slug auto-generates).
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}

		slug := strings.TrimSpace(body.Slug)

		if slug == "" {
			// Auto-generate a slug.
			slug = store.GenerateSlug(func(candidate string) bool {
				_, exists := st.GetChannel(candidate)
				return exists
			})
		} else {
			// Validate the provided slug.
			if err := store.ValidateSlug(slug); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		ch, err := st.CreateChannel(slug)
		if err != nil {
			writeError(w, http.StatusConflict, "channel already exists")
			return
		}

		writeJSON(w, http.StatusCreated, toChannelJSON(ch))
	})

	// GET /_/api/channels/{slug}
	mux.HandleFunc("GET /_/api/channels/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		ch, ok := st.GetChannel(slug)
		if !ok {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeJSON(w, http.StatusOK, toChannelJSON(ch))
	})

	// DELETE /_/api/channels/{slug}/requests
	mux.HandleFunc("DELETE /_/api/channels/{slug}/requests", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if _, ok := st.GetChannel(slug); !ok {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		st.ClearRequests(slug)
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /_/api/channels/{slug}/requests
	mux.HandleFunc("GET /_/api/channels/{slug}/requests", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if _, ok := st.GetChannel(slug); !ok {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		requests := st.GetRequests(slug)
		out := make([]requestJSON, len(requests))
		for i, req := range requests {
			out[i] = toRequestJSON(req)
		}
		writeJSON(w, http.StatusOK, out)
	})

	// GET /_/api/channels/{slug}/requests/{id}
	mux.HandleFunc("GET /_/api/channels/{slug}/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		id := r.PathValue("id")
		req, ok := st.GetRequest(slug, id)
		if !ok {
			writeError(w, http.StatusNotFound, "request not found")
			return
		}
		writeJSON(w, http.StatusOK, toRequestJSON(req))
	})

	return mux
}
