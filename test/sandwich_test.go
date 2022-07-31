/*
Create: 2022/7/31
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package test

import (
	"compress/gzip"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"
)

func TestSandwich(t *testing.T) {

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")

		s, _ := ioutil.ReadFile("sandwich.html")
		if useGzip(request) {
			minify(writer, s)
			return
		}
		writer.Write(s)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		t.Log("err", err)
	}
}

// 压缩html文件
func minify(w http.ResponseWriter, b []byte) {
	w.Header().Add("Content-Encoding", "gzip")
	w.Header().Add("Accept-Encoding", "gzip")
	gzw := gzip.NewWriter(w)
	_, _ = gzw.Write(b)
	defer func() {
		if e := gzw.Close(); e != nil {
			log.Printf("gzip write error: %v", e)
		}
	}()
}

func useGzip(request *http.Request) bool {
	accept := request.Header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}
