package proxy

import "net/http"

func NoCache(response *http.Response) {
	// 首先判断请求头中的cache
	cacheHeader := response.Header.Get("Cache-Control")
	if cacheHeader != "" {
		response.Header.Add("Cache-Control", cacheHeader)
	} else {
		if ResolveSrv(response.Request) == Backend {
			response.Header.Add("Cache-Control", "no-cache")
		}
	}
}
