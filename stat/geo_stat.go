package stat

import (
	"net"
	"os"
	"sandwich/config"
	geo2 "sandwich/geo"
	"sandwich/json"
	"sandwich/log"
	"sync/atomic"
)

// geo数据

const (
	GeoSet = "ip2country"
)

func LoadGeoStat() map[string]int64 {
	cfg := config.Get()
	geoLock.Lock()
	defer geoLock.Unlock()

	data, err := os.ReadFile(cfg.Stat.GeoFile)
	if err != nil {
		return make(map[string]int64)
	}
	var stat map[string]int64
	if err = json.Unmarshal(data, &stat); err != nil {
		return make(map[string]int64)
	}
	return stat
}

func SaveGeoStat() {
	cfg := config.Get()
	geoLock.Lock()
	defer geoLock.Unlock()
	if _, err := os.Stat(cfg.Stat.GeoFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.GeoFile, data, os.ModePerm)
	}
	geoStatByte, err := C().Get(GeoSet)
	if err != nil {
		log.ErrorF("Get GeoSet1 failed: %v\n", err)
		return
	}
	_ = os.WriteFile(cfg.Stat.GeoFile, geoStatByte, os.ModePerm)
}

// 同步数据到缓存中
func syncGEOStat() {
	geoLock.Lock()
	defer geoLock.Unlock()
	// 将临时的geo指针转换为数据
	geoDataMap := make(map[string]int64)
	for key, v := range geoIp {
		geoDataMap[key] = *v
	}

	data, err := json.Marshal(geoDataMap)
	if err != nil {
		log.ErrorF("sync geoIp failed: %v\n", err)
	}
	C().Set(GeoSet, data)
}

// AddGeo 使用协程处理 减少耗时
func AddGeo(addr string) {
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	geoLock.RLock()
	isoCode := geo2.GeoLookUp(ip)
	geoLock.RUnlock()
	if isoCode == "" {
		return
	}

	// 原子操作geo指针时 只需要读锁
	geo, ok := geoIp[isoCode]
	if !ok {
		geoLock.Lock()
		geoIp[isoCode] = new(int64)
		geoLock.Unlock()
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
