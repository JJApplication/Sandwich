/*
Project: Sandwich domain_pool.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// 域名端口映射表
var domainPool map[string][]int

// 更加安全的端口映射表
var domainPoolSync sync.Mutex

func InitPool() {
	domainPoolSync = sync.Mutex{}
	domainPool = make(map[string][]int, 10)
	InitLog()
	getDataFromMongo()

	go syncJob()
}

func getDomainPort(host string) []int {
	if d, ok := domainPool[host]; ok {
		return d
	}
	return []int{}
}

// 将端口转换为ip地址 单机的ip都是127.0.0.1
func domainReflect(host string) []string {
	group := getDomainPort(host)
	if len(group) == 0 {
		return nil
	}
	var dGroup []string
	for _, v := range group {
		dGroup = append(dGroup, fmt.Sprintf("127.0.0.1:%d", v))
	}

	return dGroup
}

func getDataFromMongo() {
	data := getAppFromMongo()
	for _, v := range data {
		log.Printf("find app [%s] from mongo, domain: [%s], ports: [%+v]\n",
			v.Meta.Name, v.Meta.Meta.Domain, v.Meta.RunData.Ports)
	}

	// 托管随机端口服务和固定端口服务
	for _, d := range data {
		domainPoolSync.Lock()
		log.Printf("load [%s] to pool\n", d.Meta.Name)
		if d.Meta.Meta.Domain != "" && d.Meta.RunData.RandomPort {
			domainPool[d.Meta.Meta.Domain] = d.Meta.RunData.Ports
		} else if d.Meta.Meta.Domain != "" && len(d.Meta.RunData.Ports) > 0 && !d.Meta.RunData.RandomPort {
			domainPool[d.Meta.Meta.Domain] = d.Meta.RunData.Ports
		}
		domainPoolSync.Unlock()
	}

	log.Println("domainPool is:")
	for k, v := range domainPool {
		log.Printf("[%s]: %+v\n", k, v)
	}
}

// 异步从数据库同步端口数据
func syncJob() {
	tick := time.NewTicker(SyncTime)
	for {
		select {
		case <-tick.C:
			log.Println("sync job active")
			getDataFromMongo()
		}
	}
}
