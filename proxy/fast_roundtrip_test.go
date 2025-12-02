package proxy

import (
	"net/http"
	"net/url"
	"testing"
)

func TestFastRoundTripper_RoundTrip_MissingPort(t *testing.T) {
	rt := NewFastRoundTripper()

	// 构造模拟日志中的请求
	// URL Host 带端口
	u := &url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:9090",
		Path:   "/favicon.ico",
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// 模拟日志中的 req.Host
	req.Host = "service.renj.io"

	// 这是一个真实的请求，需要一个监听 9090 的服务，或者我们只看是否报 "missing port" 错误
	// 如果没有服务监听，应该报 connection refused，而不是 missing port。

	// 为了测试，我们可以不用启动服务，只要错误不是 missing port 就行。
	// 或者我们可以启动一个临时的 httptest Server。

	_, err = rt.RoundTrip(req)
	if err != nil {
		t.Logf("Error: %v", err)
		// 检查 fasthttp 的 URI 状态（由于 RoundTrip 内部没有暴露 fr，我们无法直接看，只能推测）
		// 但我们可以断言错误类型
		if err.Error() == "missing port in address" {
			t.Fatal("Reproduced: missing port in address")
		}
	}
}
