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
	"log"
	"net/http"
	"strings"
	"sync"
)

// 静态文件的缓存
// 缓存headers支持gzip压缩

var SandwichCache []byte

const (
	Forbidden = iota
	Unavailable
)

func Cache(code int, w http.ResponseWriter, r *http.Request, resType int) {
	if StrictMode || !acceptHTML(r) {
		strictWrite(code, w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	switch resType {
	case Forbidden:
		writeResponse(w, r, ForbiddenPage)
		return
	case Unavailable:
		writeResponse(w, r, UnavailablePage)
		return
	default:
		writeResponse(w, r, []byte(ERRORSendProxy))
		return
	}
}

func acceptHTML(r *http.Request) bool {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		return true
	}
	return false
}

func strictWrite(code int, w http.ResponseWriter) {
	w.WriteHeader(code)
	w.Write([]byte(ERRORSendProxy))
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
	if len(SandwichCache) > CacheSize*1024*1024 {
		clearCache(&lock)
	}
	SandwichCache = b
	log.Println("fresh static html to cache")
}

func clearCache(lock *sync.Mutex) {
	lock.Lock()
	defer lock.Unlock()
	SandwichCache = []byte{}
	log.Println("reach cache-size limit")
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
