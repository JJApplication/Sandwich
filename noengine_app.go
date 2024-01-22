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

// NoEngineAppMap 获取NoEngine服务的端口映射关系
// BlogFront -> 8888
// 需要定时刷新 默认1h
// 通过判断文件修改时间来确定是否读取文件
var (
	NoEngineAppMapLock sync.Mutex
	NoEngineAppMap     map[string]string
)

func init() {
	NoEngineAppMap = make(map[string]string)
}

func InitNoEngineAppMap() {
	NoEngineAppMapLock.Lock()
	defer NoEngineAppMapLock.Unlock()
	NoEngineAppMap = loadNoEngineAppMap()
}

func loadNoEngineAppMap() map[string]string {
	if *NoEngineApp == "" {
		log.Println("NoEngineApp config is empty")
		return nil
	}
	data, err := getContent(*NoEngineApp)
	if err != nil {
		log.Printf("NoEngineApp config read error:%s\n", err.Error())
		return nil
	}

	tmp := make(map[string]string)
	if err = json.Unmarshal(data, &tmp); err != nil {
		log.Printf("NoEngineApp config parse error:%s\n", err.Error())
		return nil
	}

	return tmp
}

func syncAppMap() {
	tick := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-tick.C:
			log.Println("reload NoEngineAPPMap active")
			InitNoEngineAppMap()
			log.Println("reload NoEngineAPPMap done")
		}
	}
}
