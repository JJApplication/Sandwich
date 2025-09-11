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
	traceIdHeader := config.Get().ProxyHeader.TraceId
	traceId := response.Request.Header.Get(traceIdHeader)

	// 仅当traceID不存在时才生成并设置新的TraceID
	if traceId == "" {
		// 生成唯一的TraceID
		newTraceId := generateTraceId()
		response.Header.Set(traceIdHeader, newTraceId)
	} else {
		// 如果请求中已有TraceID，传递到响应中
		response.Header.Set(traceIdHeader, traceId)
	}
}
