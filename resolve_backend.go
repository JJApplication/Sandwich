/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"log"
	"net/http"
	"net/url"
)

// 后端服务的API转发

func resolveBackend(req *http.Request, fromConf bool) *url.URL {
	host := req.Host
	var app string
	if fromConf {
		app = NoEngineDomainMap[host].Backend
	} else {
		// 获取要转发到的后端服务名
		app = req.Header.Get(ProxyApp)
	}

	if app == "" {
		log.Println("proxy -> None error: app is nil")
		req.Header.Set(SandwichInternalFlag, SandwichBackendError)
		return nil
	}
	// 获取后端服务对应的域名
	proxyApp := getDomainByApp(app)
	if proxyApp == "" {
		log.Printf("proxy -> %s error: app domain is nil", app)
		req.Header.Set(SandwichInternalFlag, SandwichBackendError)
		return nil
	}
	// 获取domain->port映射
	dst := domainReflect(proxyApp)
	if dst == nil || len(dst) == 0 {
		addInfluxData(req, StatNotFound)
		log.Printf("domain reflect failed: [%s]\n", proxyApp)
		req.Header.Set(SandwichInternalFlag, SandwichBackendError)
		return nil
	}

	addInfluxData(req, StatPass)
	log.Printf("request recv| %s |uri: %s|host: %s\n", req.Method, req.RequestURI, proxyApp)
	req.URL.Scheme = "http"
	req.URL.Host = pickOne(dst)
	log.Printf("backend -> [%s] : [%s]\n", app, req.URL.Host)

	if req.URL == nil {
		req.Header.Set(SandwichInternalFlag, SandwichBackendError)
	}
	return req.URL
}
