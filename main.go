package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ornovog/cache/common"
)

// fetchDataFromRemote simulates a long-running function that fetches data
func fetchDataFromRemote(id int) (string, error) {
	log.Printf("Executing fetchDataFromRemote for ID: %d", id)
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Result for ID %d", id), nil
}

// expensiveComputation simulates another expensive operation
func expensiveComputation(a, b int) int {
	log.Printf("Executing expensiveComputation for a=%d, b=%d", a, b)
	time.Sleep(1 * time.Second)
	return a * b * 42
}

func main() {
	log.Println("=== Caching Layer Demo ===")

	// Example 1: Basic caching with the original function signature
	log.Println("\n--- Example 1: Basic Caching ---")
	cachedFetch := common.NewCachedFunction(fetchDataFromRemote)

	// First call - will execute the function
	start := time.Now()
	result, err := cachedFetch(42)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("First call result: %s (took %v)\n", result, time.Since(start))

	// Second call - will return from cache instantly
	start = time.Now()
	result, err = cachedFetch(42)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Second call result: %s (took %v)\n", result, time.Since(start))

	// Example 2: In-flight deduplication
	log.Println("\n--- Example 2: In-flight Deduplication ---")
	cachedFetchConcurrent := common.NewCachedFunction(fetchDataFromRemote)

	var wg sync.WaitGroup
	start = time.Now()

	// Launch 10 concurrent calls with the same parameter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			result, err := cachedFetchConcurrent(100)
			if err == nil {
				fmt.Printf("Goroutine %d got: %s\n", goroutineID, result)
			} else {
				fmt.Printf("Goroutine %d error: %v\n", goroutineID, err)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("All 10 concurrent calls completed in %v (should be ~2s, not 20s)\n", time.Since(start))

	// Example 3: Function without error return
	log.Println("\n--- Example 3: Function Without Error Return ---")
	cachedComputation := common.NewCachedFunction(expensiveComputation)

	start = time.Now()
	result1 := cachedComputation(5, 10)
	fmt.Printf("First computation result: %d (took %v)\n", result1, time.Since(start))

	start = time.Now()
	result2 := cachedComputation(5, 10)
	fmt.Printf("Second computation result: %d (took %v)\n", result2, time.Since(start))

	// Example 4: Different parameters
	log.Println("\n--- Example 4: Different Parameters ---")
	start = time.Now()
	result3 := cachedComputation(3, 7)
	fmt.Printf("Different params result: %d (took %v)\n", result3, time.Since(start))

	// Example 5: Cache capacity and LRU eviction demo
	log.Println("\n--- Example 5: Cache Capacity Test ---")
	testCacheCapacity()

	log.Println("\n=== Demo Complete ===")
}

// testCacheCapacity demonstrates cache capacity limits and LRU eviction
func testCacheCapacity() {
	// Create a simple function for testing
	testFunc := func(id int) string {
		return fmt.Sprintf("data-%d", id)
	}

	cachedTestFunc := common.NewCachedFunction(testFunc)
	// Fill cache beyond capacity to test eviction
	// Note: In a real test, we'd use a smaller cache size for demonstration
	log.Println("Testing cache behavior with multiple entries...")

	// Add a few entries
	for i := 0; i < 5; i++ {
		result := cachedTestFunc(i)
		fmt.Printf("Added entry %d: %s\n", i, result)
		time.Sleep(10 * time.Millisecond) // Small delay to see the ordering
	}

	// Access entry 1 again to make it most recently used
	result := cachedTestFunc(1)
	fmt.Printf("Accessed entry 1 again: %s\n", result)
}
