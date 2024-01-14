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
	"os"
)

// NoEngineDomainMap 获取NoEngine服务的域名映射
// 仅针对前端服务
// blog.renj.io -> {front: BlogFront, back: Blog}
var NoEngineDomainMap map[string]domainMap

type domainMap struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

func init() {
	NoEngineDomainMap = make(map[string]domainMap)
}

func loadNoEngineDomainMap() {
	if *NoEngineDomain == "" {
		log.Println("NoEngineDomain config is empty")
		return
	}
	data, err := getContent(*NoEngineDomain)
	if err != nil {
		log.Printf("NoEngineDomain config read error:%s\n", err.Error())
		return
	}
	if err = json.Unmarshal(data, &NoEngineDomainMap); err != nil {
		log.Printf("NoEngineDomain config parse error:%s\n", err.Error())
		return
	}
}

func getContent(file string) ([]byte, error) {
	return os.ReadFile(file)
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
