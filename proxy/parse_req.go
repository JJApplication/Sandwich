package proxy

import (
	"net/http"
	"net/url"
	"sandwich/breaker"
	"sandwich/constant"
	"sandwich/data"
	"sandwich/flow"
	"sandwich/log"
	"sandwich/serror"
	"strings"
)

// ParseRequest 代理从 Nginx 拿到的 host 都是带有域名的
// 直接显示为 localhost 的地址为不可信地址 直接返回错误
//
//go:inline
func ParseRequest(req *http.Request) *url.URL {
	// 优化：避免重复获取Host
	host := req.Host
	// 去除端口号（如果有），优化字符串操作
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// 断路器检查
	if !breaker.Get(host) {
		data.AddInfluxData(req, data.StatBreak)
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBucketLimit)
		log.GetLogger().Debug().
			Str("Host", host).
			Str("Remote Addr", req.RemoteAddr).
			Msg("client has been rate limited because of breakdown")
		return &url.URL{Scheme: constant.SchemeSandwich}
	}

	// 新的流控检查
	flowController := flow.GetLimiter()
	if flowController != nil {
		result := flowController.CheckRequest(req)
		if !result.Allowed {
			// 记录被限流的请求
			flowRecorder := flow.GetFlowRecorder()
			if flowRecorder != nil {
				flowRecorder.RecordBlocked(req, result)
			}

			// 添加统计数据
			data.AddInfluxData(req, data.StatAbort)
			log.GetLogger().Debug().
				Str("Remote Addr", req.RemoteAddr).
				Str("Reason", result.Reason).
				Msg("client has been rate limited")
			req.Header.Set(serror.SandwichInternalFlag, serror.SandwichReqLimit)
			return &url.URL{Scheme: constant.SchemeSandwich}
		} else {
			// 记录通过的请求（如果启用）
			flowRecorder := flow.GetFlowRecorder()
			if flowRecorder != nil {
				flowRecorder.RecordAllowed(req)
			}
		}
	}

	// 检验合法性后 判断是否为前后端转发服务
	return Resolve(req)
}
