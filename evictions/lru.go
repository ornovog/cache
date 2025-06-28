package evictions

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

func (l *lruEvictionPolicy) EvictIfNeeded(evict func(string), currentSize int, maxEntries int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for currentSize >= maxEntries && len(l.order) > 0 {
		oldest := l.order[0]
		l.order = l.order[1:]
		evict(oldest)
		log.Printf("LRU Evicted key: %s", oldest)
		currentSize--
	}
}
