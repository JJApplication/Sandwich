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
			addHeader(response)
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			if *Debug {
				log.Printf("[DEBUG] host: %s, url: %s, proto: %s, method: %s\n",
					request.Host, request.URL, request.Proto, request.Method)
			}
			// 熔断判断
			if err.Error() == "unsupported protocol scheme \"Sandwich\"" {
				if *Debug {
					log.Printf("[DEBUG] unsupported protocol scheme Sandwich")
				}
				writer.WriteHeader(http.StatusBadGateway)
				return
			}
			if err.Error() == "http: no Host in request URL" {
				if *Debug {
					log.Printf("[DEBUG] http: no Host in request URL")
				}
				writer.WriteHeader(http.StatusTooManyRequests)
				Cache(writer, request)
				return
			}
			breaker.Set(request.Host)
			log.Printf("proxy connect error: %s\n", err.Error())
			Cache(writer, request)
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
		return &url.URL{Host: "localhost", Scheme: Sandwich}
	}
	if !limiter.GetConn() {
		addInfluxData(req, StatAbort)
		log.Printf("client %s has been limit to request\n", req.RemoteAddr)
		return &url.URL{Host: "", Scheme: "http"}
	}
	defer limiter.ReleaseConn()

	// 检验合法性后 判断是否为前后端转发服务
	return Resolve(req)
}
