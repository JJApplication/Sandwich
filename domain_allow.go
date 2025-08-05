/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"fmt"
	"log"
	"net/http"
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

func InitDomainAllowList(domainData *noengineDomainMap) {
	domainAllowListLock.Lock()
	defer domainAllowListLock.Unlock()
	DomainAllowList = loadDomainAllowList(domainData)
}

func loadDomainAllowList(domainData *noengineDomainMap) map[string]struct{} {
	if domainData == nil {
		log.Println("DomainList config read empty")
		return nil
	}

	tmp := make(map[string]struct{})
	for _, domain := range domainData.Domains {
		tmp[domain] = struct{}{}
	}

	return tmp
}

// 校验域名是否绑定
// 内部请求无需校验
func validateDomain(req *http.Request) bool {
	domain := req.Host
	if (domain == fmt.Sprintf(":%s", Port) || domain == fmt.Sprintf("127.0.0.1:%s", Port)) &&
		req.Header.Get(BackendHeader) != "" {
		return true
	}
	_, ok := DomainAllowList[domain]
	return ok
}
