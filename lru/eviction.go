package lru

import (
	"log"
	"sync"
)

type lruEvictionPolicy struct {
	order []string
	mu    sync.Mutex
}

func NewLRUPolicy() *lruEvictionPolicy {
	return &lruEvictionPolicy{order: []string{}}
}

func (l *lruEvictionPolicy) Touch(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, k := range l.order {
		if k == key {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
	l.order = append(l.order, key)
}

func (l *lruEvictionPolicy) Add(key string) {
	l.Touch(key)
}

func (l *lruEvictionPolicy) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, k := range l.order {
		if k == key {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
}

func (l *lruEvictionPolicy) EvictIfNeeded(entries map[string]any, maxEntries int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for len(entries) > maxEntries && len(l.order) > 0 {
		oldest := l.order[0]
		delete(entries, oldest)
		l.order = l.order[1:]
		log.Printf("LRU Evicted key: %s", oldest)
	}
}
