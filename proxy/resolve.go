/*
   Create: 2024/1/14
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package proxy

import (
	"net/http"
	"net/url"
	"sandwich/cache"
	"sandwich/config"
	"sandwich/log"
)

// 解析req判断转发逻辑

const (
	Backend = iota
	Frontend
	FrontendFromConf
	BackendFromConf
)

// Resolve 解析是否为前后端服务 进行分别转发
// 配置优先级 > Header的优先级
// 配置为{frontend: xx, backend: xx} 纯前后端服务时对应的另一套配置为空
func Resolve(req *http.Request) *url.URL {
	log.DebugF("resolve url: %s\n", req.RequestURI)
	log.DebugF("resolve host: %s\n", req.Host)
	log.DebugF("resolve headers: %#v\n", req.Header)
	switch resolveType(req) {
	case Frontend:
		return resolveFrontend(req)
	case Backend:
		return resolveBackend(req, false)
	case BackendFromConf:
		return resolveBackend(req, true)
	default:
		return resolveFrontend(req)
	}
}

// ResolveSrv 为修改响应头识别请求的服务是否属于后端
func ResolveSrv(r *http.Request) int {
	return resolveType(r)
}

func resolveType(req *http.Request) int {
	host := req.Host
	app := cache.AppDomainMap[host]
	if app.Frontend != "" && app.Backend != "" {
		backHeader := req.Header.Get(config.Get().ProxyHeader.BackendHeader)
		if backHeader == "yes" {
			return Backend
		}

		return Frontend
	}

	if app.Frontend != "" {
		return FrontendFromConf
	} else {
		return BackendFromConf
	}
}
