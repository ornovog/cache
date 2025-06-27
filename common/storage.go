package common

import (
	"log"
	"sync"
	"time"
)

type Storage[T any] interface {
	Get(key string) (T, error, bool)
	Set(key string, value T, err error)
}

type storage[T any] struct {
	mu         sync.RWMutex
	entries    map[string]Entry[T]
	ttl        time.Duration
	maxEntries int
	eviction   EvictionPolicy
}

func NewStorage[T any](ttl time.Duration, maxEntries int, eviction EvictionPolicy) Storage[T] {
	return &storage[T]{
		entries:    make(map[string]Entry[T]),
		ttl:        ttl,
		maxEntries: maxEntries,
		eviction:   eviction,
	}
}

func (s *storage[T]) Get(key string) (T, error, bool) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok || e.IsExpired() {
		if ok {
			s.mu.Lock()
			delete(s.entries, key)
			s.eviction.Remove(key)
			s.mu.Unlock()
			log.Printf("Cache: Expired entry for key: %s", key)
		}
		var zero T
		return zero, nil, false
	}
	s.mu.Lock()
	e.RefreshLastUsed()
	s.eviction.Touch(key)
	s.mu.Unlock()
	return e.Value(), e.Error(), true
}

func (s *storage[T]) Set(key string, value T, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.entries) >= s.maxEntries {
		s.eviction.EvictIfNeeded(nil, s.maxEntries) // Simplified for now
	}
	s.entries[key] = NewEntryWithTTL(value, err, s.ttl)
	s.eviction.Add(key)
}
