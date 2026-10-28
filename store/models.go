package store

import (
	"net/http"
	"time"
)

type Channel struct {
	Slug      string
	CreatedAt time.Time
}

type Request struct {
	ID          string
	ChannelSlug string
	ReceivedAt  time.Time
	Method      string
	Path        string
	Query       map[string][]string
	Headers     http.Header
	Body        []byte
	ContentType string
	BodySize    int
}
