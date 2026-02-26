package statusdb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCreateMemoryDB 测试创建内存数据库
func TestCreateMemoryDB(t *testing.T) {
	db := CreateMemoryDB()
	assert.NotNil(t, db)
	assert.Equal(t, int64(0), db.Length())
	assert.Equal(t, 64, db.shardCount)
	db.Close()
}

// TestAddAndGet 测试添加和获取
func TestAddAndGet(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	item := Item{
		Domain: "www.example.com",
		Dns:    "8.8.8.8",
		Time:   time.Now(),
		Retry:  0,
	}

	// 添加
	db.Add("www.example.com", item)
	assert.Equal(t, int64(1), db.Length())

	// 获取
	result, ok := db.Get("www.example.com")
	assert.True(t, ok)
	assert.Equal(t, "www.example.com", result.Domain)
	assert.Equal(t, "8.8.8.8", result.Dns)
}

// TestSet 测试设置操作
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

	// 首次设置
	db.Set("test.com", item1)
	assert.Equal(t, int64(1), db.Length())

	result, _ := db.Get("test.com")
	assert.Equal(t, "1.1.1.1", result.Dns)
	assert.Equal(t, 0, result.Retry)

	// 更新
	db.Set("test.com", item2)
	assert.Equal(t, int64(1), db.Length()) // 长度不变

	result, _ = db.Get("test.com")
	assert.Equal(t, "8.8.8.8", result.Dns)
	assert.Equal(t, 1, result.Retry)
}

// TestDel 测试删除操作
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

	// 删除不存在的键
	db.Del("not.exist")
	assert.Equal(t, int64(0), db.Length())
}

// TestScan 测试扫描操作
func TestScan(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// 添加多个域名
	domains := []string{"a.com", "b.com", "c.com", "d.com"}
	for _, domain := range domains {
		db.Add(domain, Item{Domain: domain, Dns: "1.1.1.1"})
	}

	// 扫描所有域名
	scanned := make(map[string]bool)
	err := db.Scan(func(key string, value Item) error {
		scanned[key] = true
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, len(domains), len(scanned))
	for _, domain := range domains {
		assert.True(t, scanned[domain], "域名 %s 应该被扫描到", domain)
	}
}

// TestConcurrentAdd 测试并发添加
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
	assert.Equal(t, expected, db.Length(), "并发添加后长度应该正确")
}

// TestConcurrentReadWrite 测试并发读写
func TestConcurrentReadWrite(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// 预先添加一些数据
	for i := 0; i < 100; i++ {
		domain := fmt.Sprintf("test-%d.com", i)
		db.Add(domain, Item{Domain: domain, Dns: "1.1.1.1"})
	}

	var wg sync.WaitGroup
	operations := 1000

	// 并发读
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			_, _ = db.Get(domain)
		}
	}()

	// 并发写
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			db.Set(domain, Item{Domain: domain, Dns: "8.8.8.8", Retry: i})
		}
	}()

	// 并发删除
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			domain := fmt.Sprintf("test-%d.com", i%100)
			db.Del(domain)
		}
	}()

	wg.Wait()
	// 测试通过即表示无数据竞争
}

// TestSharding 测试分片均匀性
func TestSharding(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// 添加大量域名
	totalDomains := 10000
	for i := 0; i < totalDomains; i++ {
		domain := fmt.Sprintf("domain-%d.example.com", i)
		db.Add(domain, Item{Domain: domain})
	}

	// 统计每个分片的数量
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

	// 计算平均值和方差
	avg := float64(totalDomains) / float64(db.shardCount)
	var variance float64
	for _, count := range shardCounts {
		diff := float64(count) - avg
		variance += diff * diff
	}
	variance /= float64(db.shardCount)
	stdDev := variance // 简化计算

	// 分布应该相对均匀 (标准差 < 平均值的 20%)
	assert.Less(t, stdDev, avg*avg*0.04, "分片分布不够均匀")
}

// TestExpiration 测试数据过期
func TestExpiration(t *testing.T) {
	db := CreateMemoryDB()
	defer db.Close()

	// 设置1秒过期
	db.SetExpiration(1 * time.Second)

	oldItem := Item{
		Domain: "old.com",
		Dns:    "1.1.1.1",
		Time:   time.Now().Add(-2 * time.Second), // 2秒前
	}

	newItem := Item{
		Domain: "new.com",
		Dns:    "8.8.8.8",
		Time:   time.Now(), // 刚刚
	}

	db.Add("old.com", oldItem)
	db.Add("new.com", newItem)

	assert.Equal(t, int64(2), db.Length())

	// 手动触发清理
	db.cleanup()

	// 旧数据应该被清理
	assert.Equal(t, int64(1), db.Length())
	_, ok := db.Get("old.com")
	assert.False(t, ok)

	_, ok = db.Get("new.com")
	assert.True(t, ok)
}

// BenchmarkAdd 基准测试添加性能
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

// BenchmarkGet 基准测试获取性能
func BenchmarkGet(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	// 预先添加数据
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

// BenchmarkConcurrentAdd 并发添加基准测试
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

// BenchmarkGetShard 基准测试分片查找性能
func BenchmarkGetShard(b *testing.B) {
	db := CreateMemoryDB()
	defer db.Close()

	domain := "benchmark.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.getShard(domain)
	}
}
