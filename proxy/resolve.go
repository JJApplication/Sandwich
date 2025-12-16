/*
   Create: 2024/1/14
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package proxy

import (
	"net/http"
	"net/url"
	"sandwich/constant"
	"sandwich/log"
	"sandwich/utils"
)

// 解析req判断转发逻辑

// Resolve 解析是否为前后端服务 进行分别转发
// 配置优先级 > Header的优先级
// 配置为{frontend: xx, backend: xx} 纯前后端服务时对应的另一套配置为空
//
//go:inline
func Resolve(req *http.Request) *url.URL {
	log.GetLogger().Debug().
		Str("Host", req.Host).
		Str("Url", req.RequestURI).
		Any("Header", req.Header).
		Msg("resolve request")
	switch utils.ResolveSrv(req) {
	case constant.Frontend:
		return resolveFrontend(req)
	case constant.Backend:
		return resolveBackend(req, false)
	case constant.BackendFromConf:
		return resolveBackend(req, true)
	default:
		return resolveFrontend(req)
	}
}
