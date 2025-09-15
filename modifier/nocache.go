package modifier

import (
	"net/http"
	"sandwich/constant"
	"sandwich/utils"
)

type NoCache struct {
	enabled bool
}

func NewNoCache() *NoCache {
	mod := new(NoCache)
	return mod
}

func (n NoCache) Use(response *http.Response) {
	// 首先判断请求头中的cache
	cacheHeader := response.Header.Get("Cache-Control")
	if cacheHeader != "" {
		response.Header.Add("Cache-Control", cacheHeader)
	} else {
		if utils.ResolveSrv(response.Request) == constant.Backend {
			response.Header.Add("Cache-Control", "no-cache")
		}
	}
}

func (n NoCache) ModifyResponse(response *http.Response) error {
	n.Use(response)
	return nil
}

func (n NoCache) IsEnabled() bool {
	return n.enabled
}

func (n NoCache) UpdateConfig() {
	//TODO implement me
	panic("implement me")
}

func (n NoCache) GetName() string {
	return "nocache"
}
