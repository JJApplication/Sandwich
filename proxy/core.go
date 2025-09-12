/*
Package proxy
核心代理转发逻辑 实现自定义域名转发
*/
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sandwich/breaker"
	"sandwich/cache"
	"sandwich/config"
	"sandwich/constant"
	"sandwich/log"
	"sandwich/serror"
	"sandwich/utils"
	"sync"
	"time"
)

const (
	FlushInterval = 500 * time.Millisecond
)

var (
	// 全局共享的Transport实例，避免重复创建
	sharedTransport *http.Transport
	transportOnce   sync.Once
)

// getOptimizedTransport 获取优化的HTTP传输层配置
func getOptimizedTransport() *http.Transport {
	transportOnce.Do(func() {
		sharedTransport = &http.Transport{
			// 连接池配置
			MaxIdleConns:        100,              // 最大空闲连接数
			MaxIdleConnsPerHost: 20,               // 每个主机最大空闲连接数
			MaxConnsPerHost:     50,               // 每个主机最大连接数
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			// 超时配置
			ResponseHeaderTimeout: 30 * time.Second, // 响应头超时
			ExpectContinueTimeout: 1 * time.Second,  // 100-continue超时
			// 启用TCP keep-alive
			DisableKeepAlives: false,
			// 启用HTTP/2支持
			ForceAttemptHTTP2: true,
			// 禁用压缩以减少CPU开销（如果不需要）
			DisableCompression: false,
		}
	})
	return sharedTransport
}

// http转发
func newProxy() *httputil.ReverseProxy {
	cfg := config.Get()

	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			log.DebugF("parse request Header: %#v\n", request.Header)
			log.DebugF("parse request Host: %#v\n", request.Host)
			log.DebugF("parse request Trace-Id: %s\n", request.Header.Get(cfg.ProxyHeader.TraceId))
			if !cache.ValidateDomain(request) {
				request.Header.Set(serror.SandwichInternalFlag, serror.SandwichDomainNotAllow)
				request.URL = &url.URL{Scheme: constant.Sandwich}
				return
			}
			request.URL = ParseRequest(request)
			log.DebugF("parse request, URL: %#v\n", request.URL)
		},
		Transport:     getOptimizedTransport(),
		FlushInterval: FlushInterval,
		ErrorLog:      nil,
		BufferPool:    nil,
		ModifyResponse: func(response *http.Response) error {
			NoCache(response)
			utils.AddHeader(response)
			utils.AddTrace(response)
			utils.AddSecureHeader(response)
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			log.DebugF("host: %s, url: %#v, proto: %s, method: %s\n",
				request.Host, request.URL, request.Proto, request.Method)
			// 熔断判断
			switch request.Header.Get(serror.SandwichInternalFlag) {
			case serror.SandwichBucketLimit:
				log.Debug("reach breaker limit")
				writer.WriteHeader(http.StatusGatewayTimeout)
				return
			case serror.SandwichReqLimit:
				log.Debug("reach flow control limit")
				cache.Cache(http.StatusTooManyRequests, writer, request, cache.Forbidden)
				return
			case serror.SandwichDomainNotAllow:
				log.Debug("http: no Host in request URL")
				cache.Cache(http.StatusForbidden, writer, request, cache.Forbidden)
				return
			case serror.SandwichBackendError:
				log.Debug("backend: service is down")
				cache.Cache(http.StatusBadGateway, writer, request, cache.Unavailable)
			}
			breaker.Set(request.Host)
			log.ErrorF("proxy connect error: %s\n", err.Error())
			cache.Cache(http.StatusBadGateway, writer, request, cache.Unavailable)
		},
	}

	return proxy
}

func CreateProxy() *httputil.ReverseProxy {
	return newProxy()
}
