package common

import "sync"

// Deduplicator interface remains unchanged
type Deduplicator[T any] interface {
	// Wait either returns the finished result, or marks the caller as responsible for computing
	Wait(key string) (T, error, bool)
	// Finish is called once the computation is done
	Finish(key string, result T, err error)
}

// inFlightCall is the shared state for a single in-progress request
type inFlightCall[T any] struct {
	result T
	err    error
	ready  chan struct{} // closed when result is ready
}

// inFlightDedup holds all in-progress requests
type inFlightDedup[T any] struct {
	mu    sync.Mutex
	calls map[string]*inFlightCall[T]
}

// NewInFlightDedup creates a new deduplicator
func NewInFlightDedup[T any]() Deduplicator[T] {
	return &inFlightDedup[T]{calls: make(map[string]*inFlightCall[T])}
}

// Wait either returns the finished result, or marks the caller as responsible for computing
func (d *inFlightDedup[T]) Wait(key string) (T, error, bool) {
	d.mu.Lock()
	if call, exists := d.calls[key]; exists {
		d.mu.Unlock()
		<-call.ready // wait until the call is finished
		return call.result, call.err, true
	}

	// This caller is responsible for executing the function
	call := &inFlightCall[T]{ready: make(chan struct{})}
	d.calls[key] = call
	d.mu.Unlock()

	var zero T
	return zero, nil, false
}

// Finish is called once the computation is done
func (d *inFlightDedup[T]) Finish(key string, result T, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if call, exists := d.calls[key]; exists {
		call.result = result
		call.err = err
		close(call.ready) // notify all waiters
		delete(d.calls, key)
	}
}
