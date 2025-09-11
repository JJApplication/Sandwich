/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package cache

import (
	"net/http"
	"sandwich/config"
	"sandwich/log"
	"strings"
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

// ValidateDomain 校验域名是否绑定
// 内部请求无需校验
func ValidateDomain(req *http.Request) bool {
	domain := req.Host
	cf := config.Get()
	if strings.HasPrefix(domain, "127.0.0.1") ||
		req.Header.Get(cf.ProxyHeader.BackendHeader) != "" {
		return true
	}
	_, ok := DomainAllowList[domain]
	return ok
}
