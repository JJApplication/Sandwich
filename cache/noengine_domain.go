/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package cache

import (
	"sandwich/config"
	"sandwich/json"
	"sandwich/log"
	"sync"
)

type DomainMap struct {
	Domains   []string             `json:"domains"`   // 允许域名
	DomainMap map[string]domainMap `json:"domainMap"` // 域名映射
}

// AppDomainMap 获取微服务的域名映射
// 仅针对前端服务
// blog.renj.io -> {front: BlogFront, back: Blog}
var (
	AppDomainMapLock sync.Mutex
	AppDomainMap     map[string]domainMap
)

type domainMap struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

func init() {
	AppDomainMap = make(map[string]domainMap)
}

func InitNoEngineDomainMap() {
	AppDomainMapLock.Lock()
	defer AppDomainMapLock.Unlock()
	appData := loadAppDomainMap()
	if appData == nil {
		AppDomainMap = make(map[string]domainMap)
	} else {
		AppDomainMap = appData.DomainMap
	}
	InitDomainAllowList(appData)
}

func loadAppDomainMap() *DomainMap {
	cf := config.Get()
	if cf.DomainMap == "" {
		log.Info("AppDomain config is empty")
		return nil
	}
	data, err := getContent(cf.DomainMap)
	if err != nil {
		log.ErrorF("AppDomain config read error:%s\n", err.Error())
		return nil
	}

	var tmp *DomainMap
	if err = json.Unmarshal(data, &tmp); err != nil {
		log.ErrorF("AppDomain config parse error:%s\n", err.Error())
		return nil
	}

	return tmp
}

// GetDomainByApp 通过app查找域名
func GetDomainByApp(app string) string {
	for domain, appMap := range AppDomainMap {
		if appMap.Frontend == app || appMap.Backend == app {
			return domain
		}
		continue
	}

	return ""
}
