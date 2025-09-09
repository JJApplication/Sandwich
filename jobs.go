/*
   Create: 2025/8/5
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"sandwich/constant"
	"sandwich/log"
	"time"
)

func InitSyncJobs() {
	go syncJob()
	go syncDomainMap()
}

func syncDomainMap() {
	tick := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-tick.C:
			log.Info("reload NoEngineDomainMap active")
			InitNoEngineDomainMap()
			log.Info("reload NoEngineDomainMap done")
		}
	}
}

// 异步从数据库同步端口数据
func syncJob() {
	tick := time.NewTicker(constant.SyncTime)
	for {
		select {
		case <-tick.C:
			log.Info("sync job active")
			getDataFromMongo()
		}
	}
}
