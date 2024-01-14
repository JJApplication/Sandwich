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
	"time"
)

// NoEngineAppMap 获取NoEngine服务的端口映射关系
// BlogFront -> 8888
// 需要定时刷新 默认1h
// 通过判断文件修改时间来确定是否读取文件
var NoEngineAppMap map[string]string

const (
	refreshTime = 1 * 60 * 60
)

func init() {
	NoEngineAppMap = make(map[string]string)
}

func loadNoEngineAppMap() {
	if *NoEngineApp == "" {
		log.Println("NoEngineApp config is empty")
		return
	}
	data, err := getContent(*NoEngineApp)
	if err != nil {
		log.Printf("NoEngineApp config read error:%s\n", err.Error())
		return
	}
	if err = json.Unmarshal(data, &NoEngineAppMap); err != nil {
		log.Printf("NoEngineApp config parse error:%s\n", err.Error())
		return
	}
}

func syncAppMap() {
	tick := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-tick.C:
			log.Println("reload NoEngineAPPMap active")
			loadNoEngineAppMap()
			log.Println("reload NoEngineAPPMap done")
		}
	}
}
