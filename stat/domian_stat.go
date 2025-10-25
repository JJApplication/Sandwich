package stat

import (
	"os"
	"sandwich/config"
	"sandwich/json"
	"sandwich/log"
	"sandwich/structure"
	"sync/atomic"
)

const (
	DomainStat = "domain"
)

func LoadDomainStat() *structure.Map[*int64] {
	m := structure.NewMap[*int64]()
	cfg := config.Get()

	data, err := os.ReadFile(cfg.Stat.DomainFile)
	if err != nil {
		return m
	}
	var res map[string]int64
	if err = json.Unmarshal(data, &res); err != nil {
		return m
	}
	for k, v := range res {
		m.Put(k, &v)
	}

	return m
}

func SaveDomainStat() {
	cfg := config.Get()
	if _, err := os.Stat(cfg.Stat.DomainFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.GeoFile, data, os.ModePerm)
	}
	domainStatByte, err := C().Get(DomainStat)
	if err != nil {
		log.ErrorF("Get DomainStat failed: %v\n", err)
		return
	}
	_ = os.WriteFile(cfg.Stat.DomainFile, domainStatByte, os.ModePerm)
}

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
