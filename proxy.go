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
	"strings"
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
			if err != nil {
				if err.Error() == "http: no Host in request URL" {
					writer.Header().Add("Proxy-Server", ProxyServer)
					writer.Header().Add("Proxy-Copyright", Copyright)
					writer.WriteHeader(http.StatusTooManyRequests)
					writer.Write([]byte(ERRORTooMany))
					return
				}
				log.Printf("proxy connect error: %s\n", err.Error())
				writer.Header().Add("Proxy-Server", ProxyServer)
				writer.Header().Add("Proxy-Copyright", Copyright)
				writer.Write([]byte(ERRORSendProxy))
			}
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
	uri := req.RequestURI
	host := req.Host

	if !limiter.GetConn() {
		log.Printf("client %s has been limit to request\n", req.RemoteAddr)
		return &url.URL{Host: "", Scheme: "http"}
	}
	defer limiter.ReleaseConn()

	if !resolveDomain(host) {
		log.Printf("domain resolved failed: [%s]\n", host)
		return nil
	}

	dst := domainReflect(host)
	if dst == nil || len(dst) == 0 {
		log.Printf("domain reflect failed: [%s]\n", host)
		return nil
	}

	log.Printf("request recv| %s |uri: %s|host: %s\n", req.Method, uri, host)
	// log.Printf("request header: %v\n", req.Header)
	req.URL.Scheme = "http"
	req.URL.Host = pickOne(dst)
	return req.URL
}

func resolveDomain(host string) bool {
	if host == "" {
		return false
	}
	if strings.Contains(host, "localhost:") || strings.Contains(host, "127.0.0.1:") ||
		strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		return false
	}
	return true
}
