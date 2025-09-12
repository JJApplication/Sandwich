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
	"time"
)

const (
	FlushInterval = 500 * time.Millisecond
)

// http转发
func newProxy() *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			log.DebugF("parse request Header: %#v\n", request.Header)
			log.DebugF("parse request Host: %#v\n", request.Host)
			log.DebugF("parse request Trace-Id: %s\n", request.Header.Get(config.Get().ProxyHeader.TraceId))
			if !cache.ValidateDomain(request) {
				request.Header.Set(serror.SandwichInternalFlag, serror.SandwichDomainNotAllow)
				request.URL = &url.URL{Scheme: constant.Sandwich}
				return
			}
			request.URL = ParseRequest(request)
			log.DebugF("parse request, URL: %#v\n", request.URL)
		},
		Transport:     nil,
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
