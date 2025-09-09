/*
Project: Sandwich proxy.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sandwich/constant"
	"sandwich/log"
	"time"
)

const (
	FlushInterval = 60 * time.Second
)

// http转发
func newProxy() *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			log.DebugF("parse request Header: %#v\n", request.Header)
			log.DebugF("parse request Host: %#v\n", request.Host)
			log.DebugF("parse request Trace-Id: %s\n", request.Header.Get(constant.TraceID))
			if !validateDomain(request) {
				request.Header.Set(SandwichInternalFlag, SandwichDomainNotAllow)
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
			nocache(response)
			addHeader(response)
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			log.DebugF("host: %s, url: %#v, proto: %s, method: %s\n",
				request.Host, request.URL, request.Proto, request.Method)
			// 熔断判断
			switch request.Header.Get(SandwichInternalFlag) {
			case SandwichBucketLimit:
				log.Debug("reach breaker limit")
				writer.WriteHeader(http.StatusTooManyRequests)
				return
			case SandwichReqLimit:
				log.Debug("reach flow control limit")
				Cache(http.StatusTooManyRequests, writer, request, Forbidden)
				return
			case SandwichDomainNotAllow:
				log.Debug("http: no Host in request URL")
				Cache(http.StatusForbidden, writer, request, Forbidden)
				return
			case SandwichBackendError:
				log.Debug("backend: service is down")
				Cache(http.StatusBadGateway, writer, request, Unavailable)
			}
			breaker.Set(request.Host)
			log.ErrorF("proxy connect error: %s\n", err.Error())
			Cache(http.StatusBadGateway, writer, request, Unavailable)
		},
	}

	return proxy
}

func Proxy() *httputil.ReverseProxy {
	return newProxy()
}

// ParseRequest 代理从Nginx拿到的host 都是带有域名的
// 直接显示为localhost的地址为不可信地址 直接返回错误
func ParseRequest(req *http.Request) *url.URL {
	host := req.Host

	if !breaker.Get(host) {
		addInfluxData(req, StatBreak)
		req.Header.Set(SandwichInternalFlag, SandwichBucketLimit)
		return &url.URL{Scheme: constant.Sandwich}
	}
	if !limiter.GetConn() {
		addInfluxData(req, StatAbort)
		log.InfoF("client %s has been limit to request\n", req.RemoteAddr)
		req.Header.Set(SandwichInternalFlag, SandwichReqLimit)
		return &url.URL{Scheme: constant.Sandwich}
	}
	defer limiter.ReleaseConn()

	// 检验合法性后 判断是否为前后端转发服务
	return Resolve(req)
}
