/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"sandwich/cache"
	"sandwich/config"
	"sandwich/data"
	"sandwich/log"
	"sandwich/stat"
	"strings"
)

// 前台NoEngine的服务转发

func resolveFrontend(req *http.Request) *url.URL {
	host := req.Host
	if !resolveDomain(host) {
		data.AddInfluxData(req, data.StatNotFound)
		log.GetLogger().Error().Str("Host", host).Msg("domain resolved failed")
		return nil
	}
	// 根据域名获取前端app
	app := cache.AppDomainMap.MustGet(host)
	if app.Frontend != "" {
		log.GetLogger().Debug().Str("Host", host).Any("app", app).Msg("domain resolved")
		if config.Get().FrontProxy.FrontendPort <= 0 {
			log.GetLogger().Error().Str("front", app.Frontend).Msg("app port not found")
			return nil
		}
		data.AddInfluxData(req, data.StatPass)
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s:%d", config.Get().FrontProxy.FrontendHost, config.Get().FrontProxy.FrontendPort)
		req.Header.Set("Host", host)
		req.Header.Set(config.Get().FrontProxy.FrontendFlag, app.Frontend)
		// 转发请求必须携带实际HOST信息
		req.Header.Set(config.Get().ProxyHeader.FrontendHostHeader, host)
		log.GetLogger().Debug().Str("front", app.Frontend).Int("port", config.Get().FrontProxy.FrontendPort).Msg("proxy -> frontend")
		stat.Add(stat.Static)
		return req.URL
	}
	log.GetLogger().Error().Str("domain", host).Msg("domain resolved failed")
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
