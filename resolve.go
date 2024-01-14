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
)

const (
	BackendHeader = "X-Gateway-Local" // 后端服务标识
	ProxyApp      = "X-Gateway-App"   // 要转到的后端服务
)

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
		return resolveBackend(req)
	default:
		return resolveFrontend(req)
	}
}

func resolveType(req *http.Request) int {
	backHeader := req.Header.Get(BackendHeader)
	if backHeader == "yes" {
		return Backend
	}

	return Frontend
}
