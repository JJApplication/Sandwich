package proxy

import (
	"net/http"
	"net/http/httputil"
	"sandwich/config"
	"sandwich/utils"
	"time"
)

// NewHttpProxy 创建基于 net/http 的反向代理
func NewHttpProxy() http.Handler {
	cfg := config.Get()

	// 使用默认的Director, ModifyResponse, ErrorHandler
	proxy := &httputil.ReverseProxy{
		Director:       ProxyDirector,
		Transport:      getOptimizedTransport("http"), // 使用http transport
		FlushInterval:  time.Duration(utils.DefaultInt64(cfg.Proxy.FlushInterval, FlushInterval)) * time.Millisecond,
		ErrorLog:       nil,
		BufferPool:     getBufferPool(utils.DefaultInt(cfg.Proxy.BufSize, BufferSize)),
		ModifyResponse: ProxyModifyResponse,
		ErrorHandler:   ProxyErrorHandler,
	}

	return proxy
}
