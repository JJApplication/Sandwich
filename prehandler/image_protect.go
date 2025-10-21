package prehandler

import (
	"errors"
	"net/http"
	"net/url"
	"sandwich/config"
	"strings"
	"sync"
)

// 图片防盗链
// 对纯后端服务返回图片资源时做保护, 对于纯前端服务需要在前端服务Helios中处理

type ImageProtectModifier struct {
	enabled bool
	mu      sync.RWMutex // 读锁
	mime    []string
	allow   []string
}

var imageProtect *ImageProtectModifier

func NewImageProtectModifier() *ImageProtectModifier {
	sync.OnceFunc(func() {
		cfg := config.Get()
		imageProtect = &ImageProtectModifier{
			enabled: len(cfg.ImageProtect.ImageType) > 0,
			mime:    cfg.ImageProtect.ImageType,
			allow:   cfg.ImageProtect.AllowReferer,
		}
	})()

	return imageProtect
}

func (i *ImageProtectModifier) Access(r *http.Request) error {
	if !i.enabled {
		return nil
	}
	// 首先判断是否为图片
	referer := r.Header.Get("Referer")
	contentType := r.Header.Get("Content-Type")

	if !i.isValid(contentType, referer) {
		return errors.New("access denied")
	}

	return nil
}

func isInSlice(src string, list []string) bool {
	for _, v := range list {
		if v == src {
			return true
		}
	}

	return false
}

func (i *ImageProtectModifier) isValid(contentType, referer string) bool {
	baseType := strings.ToLower(strings.Split(contentType, ";")[0])
	if !isInSlice(baseType, i.mime) {
		return false
	}

	parseUrl, err := url.Parse(referer)
	if err != nil {
		return false
	}
	baseReferer := parseUrl.Hostname()

	if !isInSlice(baseReferer, i.allow) {
		return false
	}

	return true
}
