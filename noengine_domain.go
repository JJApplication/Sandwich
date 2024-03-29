/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// NoEngineDomainMap 获取NoEngine服务的域名映射
// 仅针对前端服务
// blog.renj.io -> {front: BlogFront, back: Blog}
var (
	NoEngineDomainMapLock sync.Mutex
	NoEngineDomainMap     map[string]domainMap
)

type domainMap struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

func init() {
	NoEngineDomainMap = make(map[string]domainMap)
}

func InitNoEngineDomainMap() {
	NoEngineDomainMapLock.Lock()
	defer NoEngineDomainMapLock.Unlock()
	NoEngineDomainMap = loadNoEngineDomainMap()
}

func loadNoEngineDomainMap() map[string]domainMap {
	if *NoEngineDomain == "" {
		log.Println("NoEngineDomain config is empty")
		return nil
	}
	data, err := getContent(*NoEngineDomain)
	if err != nil {
		log.Printf("NoEngineDomain config read error:%s\n", err.Error())
		return nil
	}

	var tmp map[string]domainMap
	if err = json.Unmarshal(data, &tmp); err != nil {
		log.Printf("NoEngineDomain config parse error:%s\n", err.Error())
		return nil
	}

	return tmp
}

// 通过app查找域名
func getDomainByApp(app string) string {
	for domain, appMap := range NoEngineDomainMap {
		if appMap.Frontend == app || appMap.Backend == app {
			return domain
		}
		continue
	}

	return ""
}

func syncDomainMap() {
	tick := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-tick.C:
			log.Println("reload NoEngineDomainMap active")
			InitNoEngineDomainMap()
			log.Println("reload NoEngineDomainMap done")
		}
	}
}
