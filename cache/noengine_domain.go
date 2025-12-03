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
	"sandwich/structure"
)

type DomainMap struct {
	Domains   []string                  `json:"domains"`   // 允许域名
	DomainMap *structure.Map[domainMap] `json:"domainMap"` // 域名映射
}

// AppDomainMap 获取微服务的域名映射
// 仅针对前端服务
// blog.renj.io -> {front: BlogFront, back: Blog}
var (
	AppDomainMap = structure.NewMap[domainMap]()
)

type domainMap struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

func InitNoEngineDomainMap() {
	appData := loadAppDomainMap()
	if appData != nil {
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

	var tmp map[string]domainMap
	if err = json.Unmarshal(data, &tmp); err != nil {
		log.ErrorF("AppDomain config parse error:%s\n", err.Error())
		return nil
	}

	dmap := structure.NewMap[domainMap]()
	for key, domain := range tmp {
		dmap.Put(key, domain)
	}

	return &DomainMap{
		Domains:   dmap.Keys(),
		DomainMap: dmap,
	}
}

// GetDomainByApp 通过app查找域名
func GetDomainByApp(app string) string {
	key, _, ok := AppDomainMap.Find(func(domain string, appMap domainMap) bool {
		if appMap.Frontend == app || appMap.Backend == app {
			return true
		}
		return false
	})

	if ok {
		return key
	}

	return ""
}
