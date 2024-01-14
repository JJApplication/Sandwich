/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// 前台NoEngine的服务转发

func resolveFrontend(req *http.Request) *url.URL {
	host := req.Host
	if !resolveDomain(host) {
		addInfluxData(req, StatNotFound)
		log.Printf("domain resolved failed: [%s]\n", host)
		return nil
	}
	// 根据域名获取前端app
	app := NoEngineDomainMap[host]
	if app.Frontend != "" {
		log.Printf("doamin resolved -> [%s] : [%s]\n", host, app)
		port := NoEngineAppMap[app.Frontend]
		addInfluxData(req, StatPass)
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("127.0.0.1:%s", port)
		log.Printf("frontend -> [%s] : [%s]\n", app, port)
		return req.URL
	}
	log.Printf("domain resolved failed: [%s]\n", host)
	return nil
}

// 判断是否为域名访问
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
