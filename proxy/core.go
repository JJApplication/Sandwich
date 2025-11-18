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
func getOptimizedTransport(transport string) *sandwichTransport {
	transportOnce.Do(func() {
		switch transport {
		case "http":
			sharedTransport = &sandwichTransport{
				Transport: OriginRoundTrip(),
			}
		case "fasthttp":
			sharedTransport = &sandwichTransport{
				Transport: NewFastRoundTripper(),
			}
		default:
			sharedTransport = &sandwichTransport{
				Transport: OriginRoundTrip(),
			}
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
			// 转发前安全清理敏感请求头
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
		Transport:     getOptimizedTransport(cfg.Proxy.Transport),
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
