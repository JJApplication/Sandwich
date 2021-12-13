/*
Project: Sandwich header.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"net/http"
)

// 自定义的响应头部
const (
	ProxyServer = "Sandwich"
	Copyright   = "renj.io"
)

func addHeader(response *http.Response) {
	response.Header.Add("Proxy-Server", ProxyServer)
	response.Header.Add("Proxy-Copyright", Copyright)
}
