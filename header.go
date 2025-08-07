/*
Project: Sandwich header.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"net/http"
)

// 自定义的响应头部

func addHeader(response *http.Response) {
	if response.Header.Get("Proxy-Server") == "" {
		response.Header.Add("Proxy-Server", Sandwich)
	}
	if response.Header.Get("Proxy-Copyright") == "" {
		response.Header.Add("Proxy-Copyright", Copyright)
	}
	// 设置请求的Trace-Id
	traceId := response.Request.Header.Get(TraceID)
	if traceId != "" {
		response.Header.Set(TraceID, traceId)
	}
}

func nocache(response *http.Response) {
	// 首先判断请求头中的cache
	cacheHeader := response.Header.Get("Cache-Control")
	if cacheHeader != "" {
		response.Header.Add("Cache-Control", cacheHeader)
	} else {
		if ResolveSrv(response.Request) == Backend {
			response.Header.Add("Cache-Control", "no-cache")
		}
	}
}
