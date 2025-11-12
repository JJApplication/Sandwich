package stat

import (
	"sandwich/config"
	"sandwich/json"
	"sandwich/log"
	"sync/atomic"
)

const (
	DomainStat = "domain"
)

func AddDomainStat(domain string) {
	cfg := config.Get()
	if !cfg.Stat.EnableStat {
		return
	}
	if domain == "" {
		return
	}

	// 原子操作geo指针时 只需要读锁
	ds, ok := domainStat.Get(domain)
	if !ok {
		domainStat.Put(domain, new(int64))
	} else {
		atomic.AddInt64(ds, 1)
	}
}

func GetDomainStat() []byte {
	data, err := C().Get(DomainStat)
	if err != nil {
		return nil
	}
	return data
}

func syncDomainStat() {
	domainDataMap := make(map[string]int64)

	domainStat.Range(func(key string, value *int64) bool {
		domainDataMap[key] = *value
		return true
	})

	data, err := json.Marshal(domainDataMap)
	if err != nil {
		log.ErrorF("sync domainStat failed: %v\n", err)
	}
	C().Set(DomainStat, data)
}
