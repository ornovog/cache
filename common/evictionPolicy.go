package common

type EvictionPolicy interface {
	Touch(key string)
	Add(key string)
	Remove(key string)
	EvictIfNeeded(entries map[string]any, maxEntries int)
}
