package common

type EvictionPolicy interface {
	Touch(key string)
	Add(key string)
	Remove(key string)
	EvictIfNeeded(evict func(string), currentSize int, maxEntries int)
}
