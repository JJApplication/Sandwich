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
	"sandwich/grpc_proxy"
	"sandwich/log"
	"sandwich/modifier"
	"sandwich/prehandler"
	"sandwich/serror"
	"sandwich/stat"
	seq "sandwich/stat/sequence"
	"sandwich/utils"
	"sync"
	"time"
)

const (
	FlushInterval = 100
	BufferSize    = 32 * 1024
)

var (
	// 全局共享的Transport实例，避免重复创建
	sharedTransport *sandwichTransport
	transportOnce   sync.Once
)

// getOptimizedTransport 获取优化的HTTP传输层配置
func getOptimizedTransport() *sandwichTransport {
	transportOnce.Do(func() {
		sharedTransport = &sandwichTransport{
			Transport: &http.Transport{
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
			},
		}
	})
	return sharedTransport
}

type syncBufferPool struct {
	pool    sync.Pool
	bufSize int
}

func (s *syncBufferPool) Get() []byte {
	return s.pool.Get().([]byte)
}

func (s *syncBufferPool) Put(buf []byte) {
	if cap(buf) == s.bufSize {
		s.pool.Put(buf)
	}
}

func getBufferPool(bufSize int) httputil.BufferPool {
	return &syncBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, bufSize)
			},
		},
		bufSize: bufSize,
	}
}

// http转发
func newProxy() *httputil.ReverseProxy {
	cfg := config.Get()
	mods := modifier.GetManager().GetModifiers()

	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			log.DebugF("parse request Header: %#v\n", request.Header)
			log.DebugF("parse request Host: %#v\n", request.Host)
			log.DebugF("parse request Trace-Id: %s\n", request.Header.Get(cfg.ProxyHeader.TraceId))
			stat.Add(stat.Total)
			stat.AddGeo(request.RemoteAddr)
			if !prehandler.ValidateDomain(request) {
				request.Header.Set(serror.SandwichInternalFlag, serror.SandwichDomainNotAllow)
				request.URL = &url.URL{Scheme: constant.SchemeSandwich}
				return
			}

			// 记录时序数据（域名、路径、方法）
			if seq.SeqMgt().IsEnabled() {
				path := request.URL.Path
				if path == "" {
					path = "/"
				}
				seq.SeqMgt().RecordRequest(request.Host, path, request.Method)
			}

			// 检查是否为gRPC代理请求
			if grpc_proxy.IsEnabled() {
				proxy := grpc_proxy.GetGrpcProxy()
				if proxy != nil && proxy.IsGrpcRequest(request) {
					log.DebugF("detected gRPC proxy request")
					// 设置特殊的scheme来标识gRPC请求，后续在Transport中处理
					request.URL = &url.URL{Scheme: constant.SchemeGrpc}
					return
				}
			}

			request.URL = ParseRequest(request)
			log.DebugF("parse request, URL: %#v\n", request.URL)
		},
		Transport:     getOptimizedTransport(),
		FlushInterval: time.Duration(utils.DefaultInt64(cfg.Proxy.FlushInterval, FlushInterval)) * time.Millisecond,
		ErrorLog:      nil,
		BufferPool:    getBufferPool(utils.DefaultInt(cfg.Proxy.BufSize, BufferSize)),
		ModifyResponse: func(response *http.Response) error {
			if config.Debug {
				start, end, sub := utils.PerformTime(func() {
					for _, mod := range mods {
						mod.Use(response)
					}
				})
				log.DebugF("Perform time for response modifier: start - %v end - %v - sub: %v\n", start, end, sub)
				return nil
			}

			for _, mod := range mods {
				mod.Use(response)
			}
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			log.DebugF("host: %s, url: %#v, proto: %s, method: %s, error: %v\n",
				request.Host, request.URL, request.Proto, request.Method, err)
			stat.Add(stat.Fail)
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
				breaker.Set(request.Host)
				log.Debug("backend: service is down")
				cache.Cache(http.StatusBadGateway, writer, request, cache.Unavailable)
			}
			log.ErrorF("proxy connect error: %s\n", err.Error())
			cache.Cache(http.StatusBadGateway, writer, request, cache.Unavailable)
		},
	}

	return proxy
}

func CreateProxy() *httputil.ReverseProxy {
	return newProxy()
}
