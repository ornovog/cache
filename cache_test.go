package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caching-layer/common"
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
	cachedFunc := common.NewCachedFunction(slowFunction, newLRUStorage[string](commonTTL, maxEntries))

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
	cachedFunc := common.NewCachedFunction(errorFunction, newLRUStorage[string](commonTTL, maxEntries))

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
	cachedFunc := common.NewCachedFunction(simpleFunction, newLRUStorage[int](commonTTL, maxEntries))

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
	storage := newLRUStorage[string](testTTL, maxEntries)

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

	cachedFunc := common.NewCachedFunction(slowFuncWithCounter, newLRUStorage[string](commonTTL, maxEntries))

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

//func TestCacheCapacityAndLRUEviction(t *testing.T) {
//	// Create a cache with small capacity for testing
//	cache := NewCache(3, 5*time.Minute)
//
//	// We'll manually test the cache capacity since NewCachedFunction
//	// creates its own cache. In a production system, we might want
//	// to make the cache configurable.
//
//	// Add entries up to capacity
//	for i := 0; i < 3; i++ {
//		key := fmt.Sprintf("[%d]", i)
//		cache.set(key, nil, nil)
//	}
//
//	// Verify cache is at capacity
//	stats := cache.GetStats()
//	assert.Equal(t, 3, stats["entries"])
//
//	// Add one more entry - should evict the oldest (entry 0)
//	cache.set("[3]", nil, nil)
//
//	// Should still be at capacity
//	stats = cache.GetStats()
//	assert.Equal(t, 3, stats["entries"])
//
//	// Entry 0 should be evicted
//	cache.mu.RLock()
//	_, exists := cache.entries["[0]"]
//	cache.mu.RUnlock()
//	assert.False(t, exists, "Oldest entry should be evicted")
//
//	// Entries 1, 2, 3 should exist
//	for i := 1; i <= 3; i++ {
//		cache.mu.RLock()
//		_, exists := cache.entries[fmt.Sprintf("[%d]", i)]
//		cache.mu.RUnlock()
//		assert.True(t, exists, fmt.Sprintf("Entry %d should exist", i))
//	}
//}
//
//func TestLRUOrdering(t *testing.T) {
//	cache := NewCache(3, 5*time.Minute)
//
//	// Add 3 entries
//	cache.set("[0]", nil, nil)
//	cache.set("[1]", nil, nil)
//	cache.set("[2]", nil, nil)
//
//	// Access entry 0 to make it most recently used
//	cache.get("[0]")
//
//	// Add another entry - should evict entry 1 (oldest unused)
//	cache.set("[3]", nil, nil)
//
//	// Entry 1 should be evicted (was least recently used)
//	cache.mu.RLock()
//	_, exists := cache.entries["[1]"]
//	cache.mu.RUnlock()
//	assert.False(t, exists, "Entry 1 should be evicted as LRU")
//
//	// Entries 0, 2, 3 should exist
//	for _, key := range []string{"[0]", "[2]", "[3]"} {
//		cache.mu.RLock()
//		_, exists := cache.entries[key]
//		cache.mu.RUnlock()
//		assert.True(t, exists, fmt.Sprintf("Entry %s should exist", key))
//	}
//}
//
//func TestConcurrentCacheAccess(t *testing.T) {
//	cachedFunc := NewCachedFunction(slowFunction).(func(int) (string, error))
//
//	var wg sync.WaitGroup
//	numGoroutines := 50
//	numKeys := 10
//
//	// Launch many goroutines accessing different keys
//	for i := 0; i < numGoroutines; i++ {
//		wg.Add(1)
//		go func(goroutineID int) {
//			defer wg.Done()
//
//			// Each goroutine accesses multiple keys
//			for j := 0; j < numKeys; j++ {
//				key := j % 5 // Use only 5 different keys to ensure some cache hits
//				result, err := cachedFunc(key)
//				require.NoError(t, err)
//				assert.Equal(t, fmt.Sprintf("result-%d", key), result)
//			}
//		}(i)
//	}
//
//	wg.Wait()
//	// If we get here without race conditions or deadlocks, the test passes
//}
//
//func TestCacheStatsAndClear(t *testing.T) {
//	cache := NewCache(100, 5*time.Minute)
//
//	// Initially empty
//	stats := cache.GetStats()
//	assert.Equal(t, 0, stats["entries"])
//	assert.Equal(t, 100, stats["max_entries"])
//	assert.Equal(t, 300.0, stats["ttl_seconds"]) // 5 minutes
//
//	// Add some entries
//	for i := 0; i < 5; i++ {
//		cache.set(fmt.Sprintf("[%d]", i), nil, nil)
//	}
//
//	// Check stats
//	stats = cache.GetStats()
//	assert.Equal(t, 5, stats["entries"])
//
//	// Clear cache
//	cache.Clear()
//
//	// Should be empty again
//	stats = cache.GetStats()
//	assert.Equal(t, 0, stats["entries"])
//}

// Benchmark tests (bonus requirement)
func BenchmarkDirectFunction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = slowFunction(i % 100)
	}
}

func BenchmarkCachedFunctionCold(b *testing.B) {
	cachedFunc := common.NewCachedFunction(slowFunction, newLRUStorage[string](commonTTL, maxEntries))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cachedFunc(i) // Always different keys (cold cache)
	}
}

func BenchmarkCachedFunctionWarm(b *testing.B) {
	cachedFunc := common.NewCachedFunction(slowFunction, newLRUStorage[string](commonTTL, maxEntries))

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
	cachedFunc := common.NewCachedFunction(slowFunction, newLRUStorage[string](commonTTL, maxEntries))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := 0
		for pb.Next() {
			_, _ = cachedFunc(id % 50) // Mix of cache hits and misses
			id++
		}
	})
}
