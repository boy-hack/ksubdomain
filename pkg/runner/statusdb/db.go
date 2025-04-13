package statusdb

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

type Item struct {
	Domain      string    // 查询域名
	Dns         string    // 查询dns
	Time        time.Time // 发送时间
	Retry       int       // 重试次数
	DomainLevel int       // 域名层级
}

// StatusDb 使用分片锁实现的高性能状态数据库
type StatusDb struct {
	// 使用分片锁减少锁竞争
	shards     []*DbShard
	shardCount int
	length     int64
	// 添加过期时间配置
	expiration time.Duration
	// 清理频率
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// DbShard 数据库分片，每个分片有自己的锁
type DbShard struct {
	items map[string]*Item // 使用指针存储Item，减少内存使用
	mu    sync.RWMutex
}

// CreateMemoryDB 创建一个内存数据库
func CreateMemoryDB() *StatusDb {
	// 使用64个分片以减少锁竞争
	shardCount := 64
	db := &StatusDb{
		shards:          make([]*DbShard, shardCount),
		shardCount:      shardCount,
		length:          0,
		expiration:      5 * time.Minute, // 默认条目过期时间
		cleanupInterval: 3 * time.Minute, // 默认清理频率
		stopCleanup:     make(chan struct{}),
	}

	// 初始化每个分片
	for i := 0; i < shardCount; i++ {
		db.shards[i] = &DbShard{
			items: make(map[string]*Item),
		}
	}

	// 启动自动清理协程
	go db.startCleanupTimer()

	return db
}

// 定期清理过期数据
func (r *StatusDb) startCleanupTimer() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopCleanup:
			return
		}
	}
}

// 清理过期数据
func (r *StatusDb) cleanup() {
	now := time.Now()
	threshold := now.Add(-r.expiration)

	for _, shard := range r.shards {
		shard.mu.Lock()
		for domain, item := range shard.items {
			if item.Time.Before(threshold) {
				delete(shard.items, domain)
				atomic.AddInt64(&r.length, -1)
			}
		}
		shard.mu.Unlock()
	}
}

// SetExpiration 设置条目过期时间
func (r *StatusDb) SetExpiration(d time.Duration) {
	r.expiration = d
}

// getShard 获取给定域名应该所在的分片，使用更好的哈希函数
func (r *StatusDb) getShard(domain string) *DbShard {
	// 使用fnv哈希算法，分布更均匀
	h := fnv.New32a()
	h.Write([]byte(domain))
	return r.shards[h.Sum32()%uint32(r.shardCount)]
}

// Add 添加一个项
func (r *StatusDb) Add(domain string, tableData Item) {
	shard := r.getShard(domain)
	shard.mu.Lock()
	_, exists := shard.items[domain]
	if !exists {
		atomic.AddInt64(&r.length, 1)
		// 复制一份存入map，使用指针减少内存
		itemCopy := tableData
		shard.items[domain] = &itemCopy
	} else {
		// 更新现有条目
		*(shard.items[domain]) = tableData
	}
	shard.mu.Unlock()
}

// Set 设置一个项
func (r *StatusDb) Set(domain string, tableData Item) {
	shard := r.getShard(domain)
	shard.mu.Lock()
	if _, exists := shard.items[domain]; !exists {
		// 新条目
		atomic.AddInt64(&r.length, 1)
		itemCopy := tableData
		shard.items[domain] = &itemCopy
	} else {
		// 更新现有条目
		*(shard.items[domain]) = tableData
	}
	shard.mu.Unlock()
}

// Get 获取一个项
func (r *StatusDb) Get(domain string) (Item, bool) {
	shard := r.getShard(domain)
	shard.mu.RLock()
	item, ok := shard.items[domain]
	var result Item
	if ok {
		result = *item // 解引用返回副本
	}
	shard.mu.RUnlock()
	return result, ok
}

// Length 获取元素总数
func (r *StatusDb) Length() int64 {
	return atomic.LoadInt64(&r.length)
}

// Del 删除一个项
func (r *StatusDb) Del(domain string) {
	shard := r.getShard(domain)
	shard.mu.Lock()
	_, ok := shard.items[domain]
	if ok {
		delete(shard.items, domain)
		atomic.AddInt64(&r.length, -1)
	}
	shard.mu.Unlock()
}

// Scan 遍历所有元素，优化并行性能
func (r *StatusDb) Scan(f func(key string, value Item) error) {
	// 安全检查：如果回调函数为nil，直接返回
	if f == nil {
		return
	}

	// 首先收集所有数据的副本，避免在遍历过程中持有锁
	allItems := make(map[string]Item)

	// 依次获取每个分片的数据
	for _, shard := range r.shards {
		shard.mu.RLock()
		for k, v := range shard.items {
			if v != nil { // 确保不是nil指针
				allItems[k] = *v
			}
		}
		shard.mu.RUnlock()
	}

	// 在获得所有数据副本后，执行回调
	for k, v := range allItems {
		// 如果回调返回错误，记录但继续执行
		if err := f(k, v); err != nil {
			continue
		}
	}
}

// Close 关闭数据库
func (r *StatusDb) Close() {
	// 停止清理协程
	close(r.stopCleanup)

	// 清空所有分片数据
	for _, shard := range r.shards {
		shard.mu.Lock()
		shard.items = nil
		shard.mu.Unlock()
	}
}
