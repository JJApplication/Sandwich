/*
Project: Sandwich proxy.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const (
	FlushInterval = 60 * time.Second
)

// http转发
func newProxy() *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Director: func(request *http.Request) {
			if !validateDomain(request.Host) {
				request.Header.Set(SandwichInternalFlag, SandwichDomainNotAllow)
				request.URL = &url.URL{Scheme: Sandwich}
				return
			}
			request.URL = ParseRequest(request)
			if *Debug {
				log.Printf("[DEBUG] parse request, URL: %+v\n",
					request.URL)
			}
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
			if *Debug {
				log.Printf("[DEBUG] host: %s, url: %s, proto: %s, method: %s\n",
					request.Host, request.URL, request.Proto, request.Method)
			}
			// 熔断判断
			switch request.Header.Get(SandwichInternalFlag) {
			case SandwichBucketLimit:
				if *Debug {
					log.Printf("[DEBUG] reach breaker limit")
				}
				writer.WriteHeader(http.StatusTooManyRequests)
				return
			case SandwichReqLimit:
				if *Debug {
					log.Printf("[DEBUG] reach flow control limit")
				}
				Cache(http.StatusTooManyRequests, writer, request, Forbidden)
				return
			case SandwichDomainNotAllow:
				if *Debug {
					log.Printf("[DEBUG] http: no Host in request URL")
				}
				Cache(http.StatusForbidden, writer, request, Forbidden)
				return
			case SandwichBackendError:
				if *Debug {
					log.Printf("[DEBUG] backend: service is down")
				}
				Cache(http.StatusBadGateway, writer, request, Unavailable)
			}
			breaker.Set(request.Host)
			log.Printf("proxy connect error: %s\n", err.Error())
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
		return &url.URL{Scheme: Sandwich}
	}
	if !limiter.GetConn() {
		addInfluxData(req, StatAbort)
		log.Printf("client %s has been limit to request\n", req.RemoteAddr)
		req.Header.Set(SandwichInternalFlag, SandwichReqLimit)
		return &url.URL{Scheme: Sandwich}
	}
	defer limiter.ReleaseConn()

	// 检验合法性后 判断是否为前后端转发服务
	return Resolve(req)
}
