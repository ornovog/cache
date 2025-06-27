package common

import "time"

type Entry[T any] interface {
	IsExpired() bool
	Value() T
	Error() error
	RefreshLastUsed()
}

type entryWithTTL[T any] struct {
	value     T
	err       error
	expiresAt time.Time
	lastUsed  time.Time
}

// NewEntryWithTTL creates a new entry with a specified value, error, and TTL.
func NewEntryWithTTL[T any](value T, err error, ttl time.Duration) Entry[T] {
	return &entryWithTTL[T]{
		value:     value,
		err:       err,
		expiresAt: time.Now().Add(ttl),
		lastUsed:  time.Now(),
	}
}

func (e *entryWithTTL[T]) IsExpired() bool { return time.Now().After(e.expiresAt) }
func (e *entryWithTTL[T]) Value() T        { return e.value }
func (e *entryWithTTL[T]) Error() error    { return e.err }
func (e *entryWithTTL[T]) RefreshLastUsed() {
	e.lastUsed = time.Now()
}
