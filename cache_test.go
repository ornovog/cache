package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ornovog/cache/common"
	"github.com/ornovog/cache/evictions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func slowFunction(id int) (string, error) {
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf("result-%d", id), nil
}

func errorFunction(shouldError bool) (string, error) {
	time.Sleep(50 * time.Millisecond)
	if shouldError {
		return "", fmt.Errorf("test error")
	}
	return "success", nil
}

func simpleFunction(x int) int {
	time.Sleep(50 * time.Millisecond)
	return x * 2
}

func TestBasicCaching(t *testing.T) {
	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))

	// First call should execute the function
	start := time.Now()
	result1, err1 := cachedFunc(1)
	duration1 := time.Since(start)

	require.NoError(t, err1)
	assert.Equal(t, "result-1", result1)
	assert.GreaterOrEqual(t, duration1, 100*time.Millisecond)

	// Second call should return from cache (much faster)
	start = time.Now()
	result2, err2 := cachedFunc(1)
	duration2 := time.Since(start)

	require.NoError(t, err2)
	assert.Equal(t, "result-1", result2)
	assert.Less(t, duration2, 10*time.Millisecond) // Should be much faster

	// Different parameter should execute the function again
	start = time.Now()
	result3, err3 := cachedFunc(2)
	duration3 := time.Since(start)

	require.NoError(t, err3)
	assert.Equal(t, "result-2", result3)
	assert.GreaterOrEqual(t, duration3, 100*time.Millisecond)
}

func TestCachingWithErrors(t *testing.T) {
	cachedFunc := NewCachedFunction(errorFunction).(func(bool) (string, error))

	// First call with error
	result1, err1 := cachedFunc(true)
	assert.Error(t, err1)
	assert.Equal(t, "", result1)

	// Second call with same parameter should return cached error
	start := time.Now()
	result2, err2 := cachedFunc(true)
	duration := time.Since(start)

	assert.Error(t, err2)
	assert.Equal(t, "", result2)
	assert.Less(t, duration, 10*time.Millisecond) // Should be from cache

	// Call with different parameter should succeed
	result3, err3 := cachedFunc(false)
	require.NoError(t, err3)
	assert.Equal(t, "success", result3)
}

func TestFunctionWithoutErrorReturn(t *testing.T) {
	cachedFunc := NewCachedFunction(simpleFunction).(func(int) int)

	// First call
	start := time.Now()
	result1 := cachedFunc(5)
	duration1 := time.Since(start)

	assert.Equal(t, 10, result1)
	assert.GreaterOrEqual(t, duration1, 50*time.Millisecond)

	// Second call should be cached
	start = time.Now()
	result2 := cachedFunc(5)
	duration2 := time.Since(start)

	assert.Equal(t, 10, result2)
	assert.Less(t, duration2, 10*time.Millisecond)
}

func TestTTLExpiration(t *testing.T) {
	testTTL := 200 * time.Millisecond
	storage := common.NewStorage[string](testTTL, maxEntries, evictions.NewLRUPolicy())

	// Test cache expiration directly
	testKey := "[1]"

	// Set a value with TTL
	storage.Set(testKey, "value1", nil)
	val, err, ok := storage.Get(testKey)
	require.True(t, ok)
	assert.Equal(t, "value1", val)

	// Wait for expiration
	time.Sleep(testTTL + 100*time.Millisecond) // Wait for expiration
	val, err, ok = storage.Get(testKey)
	require.False(t, ok)
	assert.Equal(t, val, "")
	assert.NoError(t, err)
}

func TestConcurrentCallDeduplication(t *testing.T) {
	var executionCount int64

	slowFuncWithCounter := func(id int) (string, error) {
		atomic.AddInt64(&executionCount, 1)
		time.Sleep(200 * time.Millisecond)
		return fmt.Sprintf("result-%d", id), nil
	}

	cachedFunc := NewCachedFunction(slowFuncWithCounter).(func(int) (string, error))

	// Launch 10 concurrent calls with the same parameter
	var wg sync.WaitGroup
	numGoroutines := 10
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := cachedFunc(42)
			results[index] = result
			errors[index] = err
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// All calls should succeed with the same result
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i])
		assert.Equal(t, "result-42", results[i])
	}

	// Function should only be executed once despite 10 concurrent calls
	assert.Equal(t, int64(1), atomic.LoadInt64(&executionCount))

	// Total time should be close to single execution time, not 10x
	assert.Less(t, duration, 400*time.Millisecond)
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
}

func TestCacheCapacityAndLRUEviction(t *testing.T) {
	ttl := 5 * time.Minute
	store := common.NewStorage[string](ttl, 3, evictions.NewLRUPolicy())

	// Fill cache to capacity
	store.Set("a", "A", nil)
	store.Set("b", "B", nil)
	store.Set("c", "C", nil)

	// All should be present
	if _, _, ok := store.Get("a"); !ok {
		t.Error("expected 'a' in cache")
	}
	if _, _, ok := store.Get("b"); !ok {
		t.Error("expected 'b' in cache")
	}
	if _, _, ok := store.Get("c"); !ok {
		t.Error("expected 'c' in cache")
	}

	// Add one more to trigger eviction (should evict LRU: 'a')
	store.Set("d", "D", nil)

	// 'a' should be evicted
	if _, _, ok := store.Get("a"); ok {
		t.Error("expected 'a' to be evicted (LRU)")
	}
	// b, c, d should remain
	for _, k := range []string{"b", "c", "d"} {
		if _, _, ok := store.Get(k); !ok {
			t.Errorf("expected '%s' to remain in cache", k)
		}
	}
}

func TestLRUOrdering(t *testing.T) {
	ttl := 5 * time.Minute
	capacity := 3
	store := common.NewStorage[string](ttl, capacity, evictions.NewLRUPolicy())

	// Insert 3 keys
	store.Set("x", "X", nil)
	store.Set("y", "Y", nil)
	store.Set("z", "Z", nil)

	// Access 'x' to make it most recently used
	store.Get("x")

	// Add another key -> should evict 'y' (was least recently used)
	store.Set("w", "W", nil)

	// Check that 'y' is gone, but 'x', 'z', and 'w' remain
	if _, _, ok := store.Get("y"); ok {
		t.Error("expected 'y' to be evicted (LRU)")
	}
	for _, k := range []string{"x", "z", "w"} {
		if _, _, ok := store.Get(k); !ok {
			t.Errorf("expected '%s' to remain in cache", k)
		}
	}
}

func TestConcurrentCacheAccess(t *testing.T) {
	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))

	var wg sync.WaitGroup
	numGoroutines := 50
	numKeys := 10

	// Launch many goroutines accessing different keys
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine accesses multiple keys
			for j := 0; j < numKeys; j++ {
				key := j % 5 // Use only 5 different keys to ensure some cache hits
				result, err := cachedFunc(key)
				require.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("result-%d", key), result)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without race conditions or deadlocks, the test passes
}

// Benchmark tests (bonus requirement)
func BenchmarkDirectFunction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = slowFunction(i % 100)
	}
}

func BenchmarkCachedFunctionCold(b *testing.B) {
	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cachedFunc(i) // Always different keys (cold cache)
	}
}

func BenchmarkCachedFunctionWarm(b *testing.B) {
	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))

	// Warm up cache with a few entries
	for i := 0; i < 10; i++ {
		_, _ = cachedFunc(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cachedFunc(i % 10) // Always hit cache
	}
}

func BenchmarkHighConcurrency(b *testing.B) {
	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := 0
		for pb.Next() {
			_, _ = cachedFunc(id % 50) // Mix of cache hits and misses
			id++
		}
	})
}
