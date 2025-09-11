/*
   Create: 2025/8/5
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package jobs

import (
	"sandwich/cache"
	"sandwich/config"
	"sandwich/constant"
	"sandwich/data"
	"sandwich/log"
	"time"
)

func InitSyncJobs() {
	go SyncJob()
	go syncDomainMap()
}

func syncDomainMap() {
	t := config.Get().JobSyncTime
	if t <= 0 {
		t = constant.SyncTime
	}
	tick := time.NewTicker(time.Duration(t) * time.Second)
	for {
		select {
		case <-tick.C:
			log.Info("reload NoEngineDomainMap active")
			cache.InitNoEngineDomainMap()
			log.Info("reload NoEngineDomainMap done")
		}
	}
}

// SyncJob 异步从数据库同步端口数据
func SyncJob() {
	t := config.Get().JobSyncTime
	if t <= 0 {
		t = constant.SyncTime
	}
	tick := time.NewTicker(time.Duration(t) * time.Second)
	for {
		select {
		case <-tick.C:
			log.Info("sync job active")
			data.GetDataFromMongo()
		}
	}
}
