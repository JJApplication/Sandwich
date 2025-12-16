/*
   Create: 2024/1/13
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package proxy

import (
	"net/http"
	"net/url"
	"sandwich/balancer"
	"sandwich/cache"
	"sandwich/config"
	"sandwich/data"
	"sandwich/log"
	"sandwich/serror"
	"sandwich/stat"

	"golang.org/x/sync/singleflight"
)

var resolveSF singleflight.Group

// 后端服务的API转发

func resolveBackend(req *http.Request, fromConf bool) *url.URL {
	host := req.Host
	var app string
	if fromConf {
		app = cache.AppDomainMap.MustGet(host).Backend
	} else {
		// 获取要转发到的后端服务名
		app = req.Header.Get(config.Get().ProxyHeader.ProxyApp)
	}

	if app == "" {
		log.Debug("proxy -> None error: app is nil")
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
		return nil
	}
	// 获取后端服务对应的域名
	// 使用singleflight优化高并发下的线性查找
	val, _, _ := resolveSF.Do("app:"+app, func() (interface{}, error) {
		return cache.GetDomainByApp(app), nil
	})
	proxyApp := val.(string)

	if proxyApp == "" {
		log.GetLogger().Debug().Str("app", app).Msg("proxy -> app error: app domain is nil")
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
		return nil
	}
	// 获取domain->port映射
	dst := data.DomainReflect(proxyApp)
	if dst == nil || len(dst) == 0 {
		data.AddInfluxData(req, data.StatNotFound)
		log.GetLogger().Debug().Str("app", proxyApp).Msg("domain reflect failed")
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
		return nil
	}

	data.AddInfluxData(req, data.StatPass)
	log.GetLogger().Info().
		Str("app", proxyApp).
		Str("method", req.Method).
		Str("uri", req.RequestURI).
		Str("host", host).Msg("proxy request recv")
	req.URL.Scheme = "http"
	req.URL.Host = balancer.PickOne(dst)
	log.GetLogger().Debug().Str("app", proxyApp).Str("Host", req.URL.Host).Msg("proxy -> backend success")

	if req.URL == nil {
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
	}
	stat.Add(stat.API)
	return req.URL
}
