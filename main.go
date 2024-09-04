package main

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

const (
	numShards       = 256
	totalOperations = 2000000
)

// CacheMode defines the operating mode of the cache
type CacheMode int

const (
	Memory CacheMode = iota
)

// ShardedCache defines the cache system with multiple shards
type ShardedCache struct {
	shards [numShards]*CacheShard
	mu     sync.RWMutex
}

// CacheShard defines a shard within the cache
type CacheShard struct {
	items map[string]*Item
	mu    sync.RWMutex
}

// Item defines the cache data structure
type Item struct {
	Value      interface{}
	Expiration int64
}

// NewShardedCache initializes a new ShardedCache
func NewShardedCache() *ShardedCache {
	cache := &ShardedCache{}
	for i := 0; i < numShards; i++ {
		cache.shards[i] = &CacheShard{
			items: make(map[string]*Item),
		}
	}
	return cache
}

// getShard selects the shard based on the key
func (sc *ShardedCache) getShard(key string) *CacheShard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return sc.shards[uint(hasher.Sum32())%numShards]
}

// Set stores a value in the cache
func (sc *ShardedCache) Set(key string, value interface{}, ttl int64) {
	shard := sc.getShard(key)
	expiration := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

	shard.mu.Lock()
	shard.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
	}
	shard.mu.Unlock()
}

// Get retrieves a value from the cache
func (sc *ShardedCache) Get(key string) (interface{}, bool) {
	shard := sc.getShard(key)

	shard.mu.RLock()
	item, found := shard.items[key]
	shard.mu.RUnlock()

	if !found || time.Now().Unix() > item.Expiration {
		if found {
			shard.mu.Lock()
			delete(shard.items, key)
			shard.mu.Unlock()
		}
		return nil, false
	}
	return item.Value, true
}

// clear clears the cache
func (sc *ShardedCache) clear() {
	for _, shard := range sc.shards {
		shard.mu.Lock()
		shard.items = make(map[string]*Item)
		shard.mu.Unlock()
	}
}

// BenchmarkSet performs a benchmark for the Set operation
func BenchmarkSet(cache *ShardedCache) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < totalOperations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Set(fmt.Sprintf("key%d", i), i, 3600)
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

// BenchmarkGet performs a benchmark for the Get operation
func BenchmarkGet(cache *ShardedCache) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < totalOperations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Get(fmt.Sprintf("key%d", i))
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}

func main() {
	// Perform benchmarking in Memory mode
	fmt.Println("Benchmarking Memory mode...")

	cacheMemory := NewShardedCache()
	setTimeMemory := BenchmarkSet(cacheMemory)
	fmt.Printf("Memory mode Set benchmark for %d requests: %v\n", totalOperations, setTimeMemory)

	getTimeMemory := BenchmarkGet(cacheMemory)
	fmt.Printf("Memory mode Get benchmark for %d requests: %v\n", totalOperations, getTimeMemory)
}