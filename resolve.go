/*
   Create: 2024/1/14
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"log"
	"net/http"
	"net/url"
)

// 解析req判断转发逻辑

const (
	Backend = iota
	Frontend
	FrontendFromConf
	BackendFromConf
)

const (
	BackendHeader = "X-Gateway-Local" // 后端服务标识
	ProxyApp      = "X-Gateway-App"   // 要转到的后端服务
)

// Resolve 解析是否为前后端服务 进行分别转发
// 配置优先级 > Header的优先级
// 配置为{frontend: xx, backend: xx} 纯前后端服务时对应的另一套配置为空
func Resolve(req *http.Request) *url.URL {
	if *Debug {
		log.Printf("[DEBUG] resolve url: %s\n", req.RequestURI)
		log.Printf("[DEBUG] resolve host: %s\n", req.Host)
		log.Printf("[DEBUG] resolve headers: %+v\n", req.Header)
	}
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
	app := NoEngineDomainMap[host]
	if app.Frontend != "" && app.Backend != "" {
		backHeader := req.Header.Get(BackendHeader)
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
