package evictions

import (
	"log"
	"sync"
)

type lfuEvictionPolicy struct {
	counts map[string]int
	mu     sync.Mutex
}

func NewLFUPolicy() *lfuEvictionPolicy {
	return &lfuEvictionPolicy{
		counts: make(map[string]int),
	}
}

func (l *lfuEvictionPolicy) Touch(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.counts[key]++
}

func (l *lfuEvictionPolicy) Add(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.counts[key] = 1
}

func (l *lfuEvictionPolicy) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.counts, key)
}

func (l *lfuEvictionPolicy) EvictIfNeeded(evict func(string), currentSize int, maxEntries int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for currentSize >= maxEntries {
		// Find the key with the lowest frequency
		var keyToEvict string
		minCount := -1
		for k, count := range l.counts {
			if minCount == -1 || count < minCount {
				minCount = count
				keyToEvict = k
			}
		}

		if keyToEvict == "" {
			return // nothing to evict
		}

		// Remove from internal tracking
		delete(l.counts, keyToEvict)

		// Call external eviction callback
		evict(keyToEvict)
		currentSize--

		log.Printf("LFU Evicted key: %s (count: %d)", keyToEvict, minCount)
	}
}
