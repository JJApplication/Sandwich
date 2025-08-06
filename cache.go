/*
Create: 2022/7/31
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package main

import (
	"bytes"
	"compress/gzip"
	"log"
	"net/http"
	"strings"
)

// 静态文件的缓存
// 缓存headers支持gzip压缩

// gzip压缩后的静态页面缓存
var (
	ForbiddenPageGzip   []byte
	UnavailablePageGzip []byte
)

const (
	Forbidden = iota
	Unavailable
	Other
)

var CodeMap = map[int][]byte{
	Forbidden:   ForbiddenPage,
	Unavailable: UnavailablePage,
	Other:       []byte(ERRORSendProxy),
}

func Cache(code int, w http.ResponseWriter, r *http.Request, resType int) {
	if StrictMode || !acceptHTML(r) {
		strictWrite(code, w)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	switch resType {
	case Forbidden:
		writeResponse(w, r, Forbidden)
		return
	case Unavailable:
		writeResponse(w, r, Unavailable)
		return
	default:
		writeResponse(w, r, Other)
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

func writeResponse(w http.ResponseWriter, request *http.Request, t int) {
	if useGzip(request) {
		// 检查是否有预压缩的缓存
		if gzipData := getGzipCache(t); gzipData != nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(gzipData)
		} else {
			minify(w, CodeMap[t])
		}
	} else {
		w.Write(CodeMap[t])
	}
}

// 初始化gzip缓存
func initGzipCache() {
	var err error
	ForbiddenPageGzip, err = compressData(ForbiddenPage)
	if err != nil {
		log.Printf("compress ForbiddenPage error: %s\n", err.Error())
	}
	UnavailablePageGzip, err = compressData(UnavailablePage)
	if err != nil {
		log.Printf("compress UnavailablePage error: %s\n", err.Error())
	}
	log.Println("gzip cache initialized")
}

// 压缩数据到字节数组
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(data)
	if err != nil {
		return nil, err
	}
	err = gzw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 获取对应的gzip缓存数据
func getGzipCache(t int) []byte {
	// 通过比较字节数组来确定是哪个页面
	if t == Forbidden {
		return ForbiddenPageGzip
	}
	if t == Unavailable {
		return UnavailablePageGzip
	}
	return nil
}

// 压缩html文件
func minify(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Encoding", "gzip")
	gzw := gzip.NewWriter(w)
	defer func() {
		if e := gzw.Close(); e != nil {
			log.Printf("gzip write error: %s\n", e.Error())
		}
	}()
	_, _ = gzw.Write(b)
}

func useGzip(request *http.Request) bool {
	if !Gzip {
		return false
	}
	accept := request.Header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}
