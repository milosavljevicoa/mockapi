package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func CaptureLogger(next http.Handler, verbose bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var body []byte
		if r.Body != nil {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				body = []byte{}
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Milliseconds()

		var sb strings.Builder
		fmt.Fprintf(&sb, "[%s] %s | %d | %dms", r.Method, r.URL.RequestURI(), rw.statusCode, duration)

		if verbose {
			sb.WriteString("\n  Headers:")
			for k, vals := range r.Header {
				sb.WriteString("\n    ")
				sb.WriteString(k)
				sb.WriteString(": ")
				sb.WriteString(strings.Join(vals, ", "))
			}
			sb.WriteString("\n  Body:\n    ")
			ct := r.Header.Get("Content-Type")
			if strings.Contains(strings.ToLower(ct), "application/json") && len(body) > 0 {
				var buf bytes.Buffer
				if err := json.Indent(&buf, body, "    ", "    "); err == nil {
					sb.Write(buf.Bytes())
				} else {
					sb.Write(body)
				}
			} else {
				sb.Write(body)
			}
		}

		log.Print(sb.String())
	})
}
