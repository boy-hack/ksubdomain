package statusdb

import "sync"

type Item struct {
	Domain      string // 查询域名
	Dns         string // 查询dns
	Time        int64  // 发送时间
	Retry       int    // 重试次数
	DomainLevel int    // 域名层级
}

type StatusDb struct {
	Items map[string]Item
	Mu    sync.RWMutex
}

// 内存简易读写数据库，自带锁机制
func CreateMemoryDB() *StatusDb {
	db := &StatusDb{
		Items: map[string]Item{},
		Mu:    sync.RWMutex{},
	}
	return db
}

func (r *StatusDb) Set(domain string, tableData Item) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.Items[domain] = tableData
}
func (r *StatusDb) Get(domain string) (Item, bool) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	v, ok := r.Items[domain]
	return v, ok
}
func (r *StatusDb) Length() int {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	length := len(r.Items)
	return length
}
func (r *StatusDb) Del(domain string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	delete(r.Items, domain)
}

func (r *StatusDb) Scan(f func(key string, value Item) error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	for k, item := range r.Items {
		f(k, item)
	}
}
func (r *StatusDb) Close() {

}
