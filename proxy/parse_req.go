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
)

// ParseRequest 代理从Nginx拿到的host 都是带有域名的
// 直接显示为localhost的地址为不可信地址 直接返回错误
func ParseRequest(req *http.Request) *url.URL {
	host := req.Host

	// 断路器检查
	if !breaker.Get(host) {
		data.AddInfluxData(req, data.StatBreak)
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBucketLimit)
		return &url.URL{Scheme: constant.Sandwich}
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
			log.InfoF("client %s has been rate limited: %s", req.RemoteAddr, result.Reason)
			req.Header.Set(serror.SandwichInternalFlag, serror.SandwichReqLimit)
			return &url.URL{Scheme: constant.Sandwich}
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
