package proxy

import (
	"net/http"
	"net/http/httputil"
	"sandwich/config"
	"sandwich/utils"
	"time"
)

// NewFastProxy 创建基于 fasthttp 的反向代理
// 使用 httputil.ReverseProxy 结构，但 Transport 替换为 fasthttp 实现
func NewFastProxy() http.Handler {
	cfg := config.Get()

	proxy := &httputil.ReverseProxy{
		Director:       ProxyDirector,
		Transport:      getOptimizedTransport("fasthttp"), // 使用fasthttp transport
		FlushInterval:  time.Duration(utils.DefaultInt64(cfg.Proxy.FlushInterval, FlushInterval)) * time.Millisecond,
		ErrorLog:       nil,
		BufferPool:     getBufferPool(utils.DefaultInt(cfg.Proxy.BufSize, BufferSize)),
		ModifyResponse: ProxyModifyResponse,
		ErrorHandler:   ProxyErrorHandler,
	}

	return proxy
}
