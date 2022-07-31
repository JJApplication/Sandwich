/*
Create: 2022/7/31
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package main

import (
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

// 静态文件的缓存

var SandwichCache []byte

func Cache(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Proxy-Server", ProxyServer)
	w.Header().Add("Proxy-Copyright", Copyright)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if checkCache() {
		writeResponse(w, r, SandwichCache)
		return
	}
	s, err := ioutil.ReadFile("sandwich.html")
	if err != nil {
		writeResponse(w, r, []byte(ERRORSendProxy))
		return
	}
	freshCache(s)
	writeResponse(w, r, s)
}

func writeResponse(w http.ResponseWriter, request *http.Request, data []byte) {
	if useGzip(request) {
		minify(w, data)
	} else {
		w.Write(data)
	}
}

func checkCache() bool {
	if len(SandwichCache) > 0 {
		return true
	}
	return false
}

func freshCache(b []byte) {
	lock := sync.Mutex{}
	lock.Lock()
	defer lock.Unlock()
	SandwichCache = b
	log.Println("fresh static html to cache")
}

// 压缩html文件
func minify(w http.ResponseWriter, b []byte) {
	w.Header().Add("Content-Encoding", "gzip")
	w.Header().Add("Accept-Encoding", "gzip")
	gzw := gzip.NewWriter(w)
	_, _ = gzw.Write(b)
	defer func() {
		if e := gzw.Close(); e != nil {
			log.Printf("gzip write error: %s\n", e.Error())
		}
	}()
}

func useGzip(request *http.Request) bool {
	accept := request.Header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}
