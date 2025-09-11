/*
Project: Sandwich header.go
Created: 2021/12/12 by Landers
*/

package utils

import (
	"net/http"
	"sandwich/config"
)

// 自定义的响应头部

func AddHeader(response *http.Response) {
	headers := config.Get().CustomHeader
	for key, value := range headers {
		if response.Header.Get(key) == "" {
			response.Header.Add(key, value)
		}
	}
}

func AddTrace(response *http.Response) {
	// 设置请求的Trace-Id
	traceId := response.Request.Header.Get(config.Get().ProxyHeader.TraceId)
	if traceId != "" {
		response.Header.Set(config.Get().ProxyHeader.TraceId, traceId)
	}
}
