package stat

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"sandwich/structure"
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

	// geo数据
	geoIp *structure.Map[*int64] // 地区请求
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
		atomic.StoreInt64(&total, m.MustGet("total"))
		atomic.StoreInt64(&api, m.MustGet("api"))
		atomic.StoreInt64(&static, m.MustGet("static"))
		atomic.StoreInt64(&fail, m.MustGet("fail"))
		atomic.StoreInt64(&today, m.MustGet("today"))
	}
	// 立即初始化一次
	go syncStat()

	geoIp = LoadGeoStat()
}
