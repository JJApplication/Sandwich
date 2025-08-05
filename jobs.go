/*
   Create: 2025/8/5
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"log"
	"time"
)

func InitSyncJobs() {
	go syncJob()
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
