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

	if !breaker.Get(host) {
		data.AddInfluxData(req, data.StatBreak)
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBucketLimit)
		return &url.URL{Scheme: constant.Sandwich}
	}
	if !flow.GetLimiter().GetConn() {
		data.AddInfluxData(req, data.StatAbort)
		log.InfoF("client %s has been limit to request\n", req.RemoteAddr)
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichReqLimit)
		return &url.URL{Scheme: constant.Sandwich}
	}
	defer flow.GetLimiter().ReleaseConn()

	// 检验合法性后 判断是否为前后端转发服务
	return Resolve(req)
}
