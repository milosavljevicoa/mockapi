package store

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// Store is a thread-safe in-memory store for channels and requests.
type Store struct {
	mu       sync.RWMutex
	channels map[string]*Channel
	requests []*Request
	cap      int
}

// New creates a new Store with the given global request capacity.
func New(cap int) *Store {
	return &Store{
		channels: make(map[string]*Channel),
		requests: make([]*Request, 0),
		cap:      cap,
	}
}

// GetOrCreateChannel returns the channel for the given slug, creating it if
// it does not already exist. Thread-safe.
func (s *Store) GetOrCreateChannel(slug string) *Channel {
	// Optimistic read first.
	s.mu.RLock()
	ch, ok := s.channels[slug]
	s.mu.RUnlock()
	if ok {
		return ch
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock.
	if ch, ok = s.channels[slug]; ok {
		return ch
	}
	ch = &Channel{
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
	}
	s.channels[slug] = ch
	return ch
}

// CreateChannel explicitly creates a channel with the given slug.
// Returns an error if the slug already exists.
func (s *Store) CreateChannel(slug string) (*Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.channels[slug]; ok {
		return nil, errors.New("channel already exists")
	}
	ch := &Channel{
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
	}
	s.channels[slug] = ch
	return ch, nil
}

// GetChannel returns the channel for the given slug.
func (s *Store) GetChannel(slug string) (*Channel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.channels[slug]
	return ch, ok
}

// ListChannels returns all channels sorted by most recently active (channel
// with the latest request first). Channels with no requests are sorted after
// active ones, among themselves by creation time descending.
func (s *Store) ListChannels() []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build a map of slug -> latest request time.
	latest := make(map[string]time.Time, len(s.channels))
	for _, r := range s.requests {
		if t, ok := latest[r.ChannelSlug]; !ok || r.ReceivedAt.After(t) {
			latest[r.ChannelSlug] = r.ReceivedAt
		}
	}

	channels := make([]*Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		channels = append(channels, ch)
	}

	sort.Slice(channels, func(i, j int) bool {
		ti := latest[channels[i].Slug]
		tj := latest[channels[j].Slug]
		if ti.IsZero() && tj.IsZero() {
			// Both have no requests; sort by creation time descending.
			return channels[i].CreatedAt.After(channels[j].CreatedAt)
		}
		if ti.IsZero() {
			return false // i has no requests, j does — j wins
		}
		if tj.IsZero() {
			return true // i has requests, j doesn't — i wins
		}
		return ti.After(tj)
	})

	return channels
}

// AddRequest appends a request to the global slice. If the global cap is
// exceeded, the globally oldest request is evicted first.
func (s *Store) AddRequest(r *Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cap > 0 && len(s.requests) >= s.cap {
		// Evict the oldest (front of slice) and nil it out so the GC
		// can reclaim the *Request before reslicing.
		s.requests[0] = nil
		s.requests = s.requests[1:]
	}
	s.requests = append(s.requests, r)
}

// GetRequests returns all requests for the given channel slug, newest first.
func (s *Store) GetRequests(slug string) []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*Request
	for _, r := range s.requests {
		if r.ChannelSlug == slug {
			out = append(out, r)
		}
	}

	// Reverse so newest is first (the global slice is oldest-first).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// GetRequest returns a single request by channel slug and request ID.
func (s *Store) GetRequest(slug, id string) (*Request, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.requests {
		if r.ChannelSlug == slug && r.ID == id {
			return r, true
		}
	}
	return nil, false
}

// ClearRequests removes all requests for the given channel slug.
func (s *Store) ClearRequests(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := s.requests[:0]
	for _, r := range s.requests {
		if r.ChannelSlug != slug {
			out = append(out, r)
		}
	}
	// Nil out the tail of the backing array so the GC can reclaim evicted
	// *Request pointers that are no longer reachable through s.requests.
	tail := s.requests[len(out):]
	for i := range tail {
		tail[i] = nil
	}
	s.requests = out
}
