package test

import (
	"fmt"
	"net/http"
	"sandwich/config"
	"sandwich/flow"
	"time"
)

// 简单的流控测试程序
func main() {
	// 创建测试配置
	testConfig := &config.FlowControlConfig{
		Enabled: true,
		GlobalLimit: config.RateLimit{
			Requests: 5,
			Window:   "10s",
			Unit:     "s",
		},
		Rules: []config.FlowControlRule{
			{
				Name:       "test-host",
				Enabled:    true,
				Priority:   1,
				MatchType:  "host",
				MatchValue: "test.example.com",
				Limits: []config.RateLimit{
					{Requests: 2, Window: "5s", Unit: "s"},
				},
				Action:      "block",
				Description: "测试主机限流",
			},
			{
				Name:       "test-user-agent",
				Enabled:    true,
				Priority:   2,
				MatchType:  "header",
				HeaderKey:  "User-Agent",
				MatchValue: "TestBot",
				Limits: []config.RateLimit{
					{Requests: 1, Window: "10s", Unit: "s"},
				},
				Action:      "block",
				Description: "测试User-Agent限流",
			},
		},
		Recording: config.FlowRecordConfig{
			Enabled:       true,
			RecordBlocked: true,
			RecordAllowed: true,
			StorageType:   "file",
		},
	}

	// 模拟config.Get()的返回值
	mockConfig := &config.Config{
		FlowControl: *testConfig,
	}

	// 这里应该设置全局配置，但为了测试，我们直接创建控制器
	fmt.Println("=== 流控功能测试 ===")

	// 测试请求
	testRequests := []*http.Request{
		createTestRequest("test.example.com", "NormalUser", "192.168.1.1"),
		createTestRequest("test.example.com", "NormalUser", "192.168.1.1"),
		createTestRequest("test.example.com", "NormalUser", "192.168.1.1"), // 这个应该被限流
		createTestRequest("other.example.com", "TestBot", "192.168.1.2"),
		createTestRequest("other.example.com", "TestBot", "192.168.1.2"), // 这个应该被限流
		createTestRequest("normal.example.com", "Chrome", "192.168.1.3"),
	}

	fmt.Printf("测试配置:\n- 全局限制: %d请求/%s\n", testConfig.GlobalLimit.Requests, testConfig.GlobalLimit.Window)
	fmt.Printf("- Host规则: %s限制%d请求/5s\n", testConfig.Rules[0].MatchValue, testConfig.Rules[0].Limits[0].Requests)
	fmt.Printf("- User-Agent规则: %s限制%d请求/10s\n", testConfig.Rules[1].MatchValue, testConfig.Rules[1].Limits[0].Requests)
	fmt.Println()

	for i, req := range testRequests {
		fmt.Printf("测试请求 %d: Host=%s, User-Agent=%s, IP=%s\n",
			i+1, req.Host, req.Header.Get("User-Agent"), req.RemoteAddr)

		// 这里需要实际的流控检查逻辑，但由于依赖完整的配置系统，
		// 我们只是展示测试框架
		fmt.Printf("  结果: 需要完整配置系统支持才能实际测试\n")
		fmt.Println()

		// 添加小延迟来模拟实际请求间隔
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("=== 测试完成 ===")
	fmt.Println("注意: 完整测试需要:")
	fmt.Println("1. 正确的配置文件 (config.json)")
	fmt.Println("2. 启动完整的Sandwich服务")
	fmt.Println("3. 发送实际的HTTP请求进行测试")
}

func createTestRequest(host, userAgent, remoteAddr string) *http.Request {
	req, _ := http.NewRequest("GET", "http://"+host+"/test", nil)
	req.Host = host
	req.Header.Set("User-Agent", userAgent)
	req.RemoteAddr = remoteAddr + ":12345"
	return req
}
