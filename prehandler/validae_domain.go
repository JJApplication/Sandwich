package prehandler

import (
	"net/http"
	"sandwich/cache"
	"sandwich/config"
	"strings"
)

// ValidateDomain 校验域名是否绑定
// 内部请求无需校验
func ValidateDomain(req *http.Request) bool {
	domain := req.Host
	cf := config.Get()
	if strings.HasPrefix(domain, "127.0.0.1") ||
		req.Header.Get(cf.ProxyHeader.BackendHeader) != "" {
		return true
	}
	_, ok := cache.DomainAllowList[domain]
	return ok
}
