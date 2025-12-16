/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package cache

import (
	"sandwich/log"
	"sandwich/structure"
)

// 域名允许列表
// 不在列表中的域名直接拒绝请求

var (
	DomainAllowList = structure.NewMapStruct(100)
)

func InitDomainAllowList(domainData *DomainMap) {
	loadDomainAllowList(domainData)
}

func loadDomainAllowList(domainData *DomainMap) {
	if domainData == nil {
		log.Info("DomainList config read empty")
		return
	}

	for _, domain := range domainData.Domains {
		DomainAllowList.Put(domain)
	}
}
