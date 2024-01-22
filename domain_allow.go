/*
   Create: 2024/1/22
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

// 域名允许列表
// 不在列表中的域名直接拒绝请求

var (
	domainAllowListLock sync.Mutex
	DomainAllowList     map[string]struct{}
)

func init() {
	DomainAllowList = make(map[string]struct{})
}

func InitDomainAllowList() {
	domainAllowListLock.Lock()
	defer domainAllowListLock.Unlock()
	DomainAllowList = loadDomainAllowList()
}

func loadDomainAllowList() map[string]struct{} {
	data, err := getContent(*DomainList)
	if err != nil {
		log.Printf("DomainList config read error:%s\n", err.Error())
		return nil
	}
	var dl []string
	if err = json.Unmarshal(data, &dl); err != nil {
		log.Printf("DomainList config parse error:%s\n", err.Error())
		return nil
	}

	tmp := make(map[string]struct{})
	for _, domain := range dl {
		tmp[domain] = struct{}{}
	}

	return tmp
}

// 校验域名是否绑定
func validateDomain(domain string) bool {
	_, ok := DomainAllowList[domain]
	return ok
}

func syncDomainList() {
	tick := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-tick.C:
			log.Println("reload DomainList active")
			InitDomainAllowList()
			log.Println("reload DomainList done")
		}
	}
}
