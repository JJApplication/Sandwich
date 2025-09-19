package stat

import (
	"encoding/json"
	"os"
	"sandwich/config"
	"sync"
)

// 持久化存储数据到文件

var (
	lock = sync.RWMutex{}
)

func LoadStat() map[string]int64 {
	cfg := config.Get()
	lock.Lock()
	defer lock.Unlock()

	data, err := os.ReadFile(cfg.Stat.SaveFile)
	if err != nil {
		return make(map[string]int64)
	}
	var stat map[string]int64
	if err = json.Unmarshal(data, &stat); err != nil {
		return make(map[string]int64)
	}
	return stat
}

func SaveStat(f string) {
	lock.Lock()
	defer lock.Unlock()
	if _, err := os.Stat(f); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{
			"total":  0,
			"api":    0,
			"static": 0,
			"fail":   0,
			"today":  0,
		})
		_ = os.WriteFile(f, data, os.ModePerm)
	}
	m := make(map[string]int64)
	m["total"] = Get(Total)
	m["api"] = Get(API)
	m["static"] = Get(Static)
	m["fail"] = Get(Fail)
	m["today"] = Get(Today)

	data, _ := json.Marshal(m)
	_ = os.WriteFile(f, data, os.ModePerm)
}
