package stat

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"sync/atomic"
	"time"
)

var (
	bc     *bigcache.BigCache
	total  int64
	api    int64
	static int64
	fail   int64
	today  int64
)

func C() *bigcache.BigCache {
	return bc
}

func init() {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             1024,
		LifeWindow:         48 * time.Hour,
		CleanWindow:        30 * time.Minute,
		MaxEntriesInWindow: 1024 * 1024,
		MaxEntrySize:       1024,
	})

	bc = cache
}

func initCacheFromFile() {
	m := LoadStat()
	if m != nil {
		atomic.StoreInt64(&total, m["total"])
		atomic.StoreInt64(&api, m["api"])
		atomic.StoreInt64(&static, m["static"])
		atomic.StoreInt64(&fail, m["fail"])
		atomic.StoreInt64(&today, m["today"])
	}
	// 立即初始化一次
	go syncStat()
}
