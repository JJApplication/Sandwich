package utils

import (
	"net/http"
	"sandwich/cache"
	"sandwich/config"
	"sandwich/constant"
	"sandwich/stat"
)

// ResolveSrv 为修改响应头识别请求的服务是否属于后端
//
//go:inline
func ResolveSrv(r *http.Request) int {
	return resolveType(r)
}

//go:inline
func resolveType(req *http.Request) int {
	host := req.Host
	stat.AddDomainStat(host)
	app, ok := cache.AppDomainMap.Get(host)
	if !ok {
		return constant.Frontend
	}
	if app.Frontend != "" && app.Backend != "" {
		backHeader := req.Header.Get(config.Get().ProxyHeader.BackendHeader)
		if backHeader == "yes" {
			return constant.Backend
		}

		return constant.Frontend
	}

	if app.Frontend != "" {
		return constant.FrontendFromConf
	} else {
		return constant.BackendFromConf
	}
}
