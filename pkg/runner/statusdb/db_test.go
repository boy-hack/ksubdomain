package statusdb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCreateMemoryDB tests creating an in-memory database
func TestCreateMemoryDB(t *testing.T) {
	db := CreateMemoryDB()
	assert.NotNil(t, db)
	assert.Equal(t, int64(0), db.Length())
	assert.Equal(t, 64, db.shardCount)
	db.Close()
}

// TestAddAndGet tests adding and retrieving items
func TestAddAndGet(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	item := Item{
		Domain: "www.example.com",
		Dns:    "8.8.8.8",
		Time:   time.Now(),
		Retry:  0,
	}

	// Add
	db.Add("www.example.com", item)
	assert.Equal(t, int64(1), db.Length())

	// Get
	result, ok := db.Get("www.example.com")
	assert.True(t, ok)
	assert.Equal(t, "www.example.com", result.Domain)
	assert.Equal(t, "8.8.8.8", result.Dns)
}

// TestSet tests the set operation
func TestSet(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	item1 := Item{
		Domain: "test.com",
		Dns:    "1.1.1.1",
		Retry:  0,
	}

	item2 := Item{
		Domain: "test.com",
		Dns:    "8.8.8.8",
		Retry:  1,
	}

	// First set
	db.Set("test.com", item1)
	assert.Equal(t, int64(1), db.Length())

	result, _ := db.Get("test.com")
	assert.Equal(t, "1.1.1.1", result.Dns)
	assert.Equal(t, 0, result.Retry)

	// Update
	db.Set("test.com", item2)
	assert.Equal(t, int64(1), db.Length()) // Length should not change

	result, _ = db.Get("test.com")
	assert.Equal(t, "8.8.8.8", result.Dns)
	assert.Equal(t, 1, result.Retry)
}

// TestDel tests the delete operation
func TestDel(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	item := Item{
		Domain: "delete.me",
		Dns:    "1.1.1.1",
	}

	db.Add("delete.me", item)
	assert.Equal(t, int64(1), db.Length())

	db.Del("delete.me")
	assert.Equal(t, int64(0), db.Length())

	_, ok := db.Get("delete.me")
	assert.False(t, ok)

	// Delete a non-existent key
	db.Del("not.exist")
	assert.Equal(t, int64(0), db.Length())
}

// TestScan tests the scan operation
func TestScan(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// Add multiple domains
	domains := []string{"a.com", "b.com", "c.com", "d.com"}
	for _, domain := range domains {
		db.Add(domain, Item{Domain: domain, Dns: "1.1.1.1"})
	}

	// Scan all domains
	scanned := make(map[string]bool)
	err := db.Scan(func(key string, value Item) error {
		scanned[key] = true
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, len(domains), len(scanned))
	for _, domain := range domains {
		assert.True(t, scanned[domain], "Domain %s should have been scanned", domain)
	}
}

// TestConcurrentAdd tests concurrent add operations
func TestConcurrentAdd(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	concurrency := 100
	itemsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				domain := fmt.Sprintf("domain-%d-%d.com", id, j)
				db.Add(domain, Item{
					Domain: domain,
					Dns:    "8.8.8.8",
				})
			}
		}(i)
	}

	wg.Wait()

	expected := int64(concurrency * itemsPerGoroutine)
	assert.Equal(t, expected, db.Length(), "Length should be correct after concurrent add")
}

// TestConcurrentReadWrite tests concurrent read and write operations
func TestConcurrentReadWrite(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// Pre-add some data
	for i := 0; i < 100; i++ {
		domain := fmt.Sprintf("test-%d.com", i)
		db.Add(domain, Item{Domain: domain, Dns: "1.1.1.1"})
	}

	var wg sync.WaitGroup
	operations := 1000

	// Concurrent reads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			_, _ = db.Get(domain)
		}
	}()

	// Concurrent writes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			db.Set(domain, Item{Domain: domain, Dns: "8.8.8.8", Retry: i})
		}
	}()

	// Concurrent deletes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			db.Del(domain)
		}
	}()

	wg.Wait()
	// Test passes if there is no data race
}

// TestSharding tests shard distribution uniformity
func TestSharding(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// Add a large number of domains
	totalDomains := 10000
	for i := 0; i < totalDomains; i++ {
		domain := fmt.Sprintf("domain-%d.example.com", i)
		db.Add(domain, Item{Domain: domain})
	}

	// Count the number of items per shard
	shardCounts := make([]int, db.shardCount)
	for i := 0; i < totalDomains; i++ {
		domain := fmt.Sprintf("domain-%d.example.com", i)
		shard := db.getShard(domain)
		for idx, s := range db.shards {
			if s == shard {
				shardCounts[idx]++
				break
			}
		}
	}

	// Calculate mean and variance
	avg := float64(totalDomains) / float64(db.shardCount)
	var variance float64
	for _, count := range shardCounts {
		diff := float64(count) - avg
		variance += diff * diff
	}
	variance /= float64(db.shardCount)
	stdDev := variance // Simplified calculation

	// Distribution should be relatively uniform (std dev < 20% of mean)
	assert.Less(t, stdDev, avg*avg*0.04, "Shard distribution is not uniform enough")
}

// TestExpiration tests data expiration
func TestExpiration(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// Set 1-second expiration
	db.SetExpiration(1 * time.Second)

	oldItem := Item{
		Domain: "old.com",
		Dns:    "1.1.1.1",
		Time:   time.Now().Add(-2 * time.Second), // 2 seconds ago
	}

	newItem := Item{
		Domain: "new.com",
		Dns:    "8.8.8.8",
		Time:   time.Now(), // Just now
	}

	db.Add("old.com", oldItem)
	db.Add("new.com", newItem)

	assert.Equal(t, int64(2), db.Length())

	// Manually trigger cleanup
	db.cleanup()

	// Old data should be cleaned up
	assert.Equal(t, int64(1), db.Length())
	_, ok := db.Get("old.com")
	assert.False(t, ok)

	_, ok = db.Get("new.com")
	assert.True(t, ok)
}

// BenchmarkAdd benchmarks add performance
func BenchmarkAdd(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	item := Item{
		Domain: "benchmark.com",
		Dns:    "8.8.8.8",
		Time:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain := fmt.Sprintf("domain-%d.com", i)
		db.Add(domain, item)
	}
}

// BenchmarkGet benchmarks get performance
func BenchmarkGet(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	// Pre-add data
	for i := 0; i < 10000; i++ {
		domain := fmt.Sprintf("domain-%d.com", i)
		db.Add(domain, Item{Domain: domain})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain := fmt.Sprintf("domain-%d.com", i%10000)
		_, _ = db.Get(domain)
	}
}

// BenchmarkConcurrentAdd benchmarks concurrent add performance
func BenchmarkConcurrentAdd(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			domain := fmt.Sprintf("domain-%d.com", i)
			db.Add(domain, Item{Domain: domain, Dns: "8.8.8.8"})
			i++
		}
	})
}

// BenchmarkGetShard benchmarks shard lookup performance
func BenchmarkGetShard(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	domain := "benchmark.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.getShard(domain)
	}
}
