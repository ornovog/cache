# Go Caching Layer

A modular and extensible function result caching library for Go, featuring:

- ✅ Transparent memoization for functions
- 🔁 Pluggable eviction policies (LRU, LFU)
- ⏱️ TTL (Time-to-Live) expiration
- 🧠 In-flight deduplication (single-flight)
- 🧪 Type-safe API via generics and reflection

The design emphasizes easy replacement and configuration of underlying components such as eviction policies, making it simple to switch from LRU to LFU or other strategies.

---

## 🚀 Getting Started

```bash
go get github.com/ornovog/cache
```

---

## 🧩 Example Usage

### Basic Function Caching

```go
cachedFetch := cache.NewCachedFunction(fetchDataFromRemote)

result, err := cachedFetch(42)
fmt.Println(result)
```

### Caching a Function Without Error

```go
cachedMultiply := cache.NewCachedFunction(func(x, y int) int {
    return x * y * 42
})

fmt.Println(cachedMultiply(3, 7))
```

---

## 🔧 Features

### ✅ Transparent Wrapping
Use `NewCachedFunction` to automatically memoize any function with or without error return.

### 🧠 In-Flight Deduplication
Avoids redundant executions of the same function call across goroutines:

```go
// These goroutines will share the same execution path for identical arguments
for i := 0; i < 10; i++ {
    go func() {
        _, _ = cachedFetch(100)
    }()
}
```

### 🔁 Pluggable Eviction Strategies

Supports both LRU and LFU eviction policies, and can be easily extended:

```go
store := cache.NewStorage[string](time.Minute, 100, cache.NewLRUPolicy())
store := cache.NewStorage[string](time.Minute, 100, cache.NewLFUPolicy())
```

This flexibility is built-in by design, allowing users to swap implementations without changing core logic.

---

## 🧪 Testing

```bash
go test -v ./...
```

## 🧪 Benchmarking
```bash
go test -bench=. -benchmem ./...
```

Includes:
- Unit tests for correctness
- Concurrency tests
- Benchmark tests for warm/cold cache and parallel performance

---