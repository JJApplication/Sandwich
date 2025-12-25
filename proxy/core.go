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
//
//go:inline
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

//go:inline
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

// ProxyDirector 代理请求处理逻辑
func ProxyDirector(request *http.Request) {
	cfg := config.Get()
	log.GetLogger().Debug().
		Any("Header", request.Header).
		Str("Host", request.Host).
		Str("Trace-ID", request.Header.Get(cfg.ProxyHeader.TraceId)).
		Msg("parse request")
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
			log.Debug("detected gRPC proxy request")
			// 设置特殊的scheme来标识gRPC请求，后续在Transport中处理
			request.URL = &url.URL{Scheme: constant.SchemeGrpc}
			return
		}
	}

	request.URL = ParseRequest(request)
	log.GetLogger().Debug().Any("URL", request.URL).Msg("parse request")
}

// ProxyModifyResponse 响应修改逻辑
func ProxyModifyResponse(response *http.Response) error {
	mods := modifier.GetManager().GetModifiers()
	if config.Debug {
		start, end, sub := utils.PerformTime(func() {
			for _, mod := range mods {
				mod.Use(response)
			}
		})
		log.GetLogger().Debug().Time("start", start).Time("end", end).Dur("sub", sub).Msg("Perform time for response modifier")
		return nil
	}

	for _, mod := range mods {
		mod.Use(response)
	}
	return nil
}

// ProxyErrorHandler 错误处理逻辑
func ProxyErrorHandler(writer http.ResponseWriter, request *http.Request, err error) {
	log.GetLogger().Debug().
		Str("host", request.Host).
		Str("url", request.URL.String()).
		Str("proto", request.Proto).
		Str("method", request.Method).
		Err(err).Msg("Proxy Error")
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
	log.GetLogger().Debug().Err(err).Msg("proxy connect error")
	cache.Cache(http.StatusBadGateway, writer, request, cache.Unavailable)
}

// http转发
//
//go:inline
func newProxy() http.Handler {
	cfg := config.Get()
	mode := cfg.Proxy.ProxyMode

	switch mode {
	case "http":
		return NewHttpProxy()
	case "fasthttp":
		return NewFastProxy()
	default:
		return NewHttpProxy()
	}
}

func CreateProxy() http.Handler {
	return newProxy()
}
