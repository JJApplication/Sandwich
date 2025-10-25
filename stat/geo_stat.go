package stat

import (
	"net"
	"os"
	"sandwich/config"
	geo2 "sandwich/geo"
	"sandwich/json"
	"sandwich/log"
	"sandwich/structure"
	"sync/atomic"
)

// geo数据

const (
	GeoSet = "ip2country"
)

func LoadGeoStat() *structure.Map[*int64] {
	var geoStat = structure.NewMap[*int64]()
	cfg := config.Get()

	data, err := os.ReadFile(cfg.Stat.GeoFile)
	if err != nil {
		return geoStat
	}

	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return geoStat
	}
	for k, v := range tmp {
		geoStat.Put(k, &v)
	}

	return geoStat
}

func SaveGeoStat() {
	cfg := config.Get()
	if _, err := os.Stat(cfg.Stat.GeoFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.GeoFile, data, os.ModePerm)
	}
	geoStatByte, err := C().Get(GeoSet)
	if err != nil {
		log.ErrorF("Get GeoSet failed: %v\n", err)
		return
	}
	_ = os.WriteFile(cfg.Stat.GeoFile, geoStatByte, os.ModePerm)
}

// 同步数据到缓存中
func syncGEOStat() {
	// 将临时的geo指针转换为数据
	geoDataMap := make(map[string]int64)

	geoIp.Range(func(key string, value *int64) bool {
		geoDataMap[key] = *value
		return true
	})

	data, err := json.Marshal(geoDataMap)
	if err != nil {
		log.ErrorF("sync geoIp failed: %v\n", err)
	}
	C().Set(GeoSet, data)
}

// AddGeo 使用协程处理 减少耗时
func AddGeo(addr string) {
	cfg := config.Get()
	if !cfg.Stat.EnableStat {
		return
	}
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	isoCode := geo2.GeoLookUp(ip)
	if isoCode == "" {
		return
	}

	// 原子操作geo指针时 只需要读锁
	geo, ok := geoIp.Get(isoCode)
	if !ok {
		geoIp.Put(isoCode, new(int64))
	} else {
		atomic.AddInt64(geo, 1)
	}
}

func GetGeoData() []byte {
	data, err := C().Get(GeoSet)
	if err != nil {
		return nil
	}
	return data
}
