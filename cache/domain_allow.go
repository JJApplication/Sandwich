/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package cache

import (
	"sandwich/log"
	"sync"
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

func InitDomainAllowList(domainData *DomainMap) {
	domainAllowListLock.Lock()
	defer domainAllowListLock.Unlock()
	DomainAllowList = loadDomainAllowList(domainData)
}

func loadDomainAllowList(domainData *DomainMap) map[string]struct{} {
	if domainData == nil {
		log.Info("DomainList config read empty")
		return nil
	}

	tmp := make(map[string]struct{})
	for _, domain := range domainData.Domains {
		tmp[domain] = struct{}{}
	}

	return tmp
}
