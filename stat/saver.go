package stat

import (
	"os"
	"sandwich/config"
	"sandwich/json"
	"sandwich/structure"
)

// 持久化存储数据到文件

func LoadStat() *structure.Map[int64] {
	cfg := config.Get()

	data, err := os.ReadFile(cfg.Stat.SaveFile)
	if err != nil {
		return structure.NewMap[int64]()
	}
	var stat = structure.NewMap[int64]()
	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return structure.NewMap[int64]()
	}

	for k, v := range tmp {
		stat.Put(k, v)
	}

	return stat
}

func SaveStat(f string) {
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
