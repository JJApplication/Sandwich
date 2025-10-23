package test

import (
	"fmt"
	"os"
	"sandwich/config"
	"sandwich/json"
)

// 测试最大请求体配置的加载和使用
func main() {
	// 创建测试配置
	testConfig := &config.Config{
		Servers: []config.ServerConfig{
			{
				Name:           "test-http",
				Host:           "127.0.0.1",
				Port:           8080,
				Protocol:       "http",
				Enabled:        true,
				MaxRequestBody: 10 * 1024 * 1024, // 10MB
				UseHttp2:       false,
			},
			{
				Name:           "test-https",
				Host:           "127.0.0.1",
				Port:           8443,
				Protocol:       "https",
				Enabled:        true,
				MaxRequestBody: 50 * 1024 * 1024, // 50MB
				UseHttp2:       true,
				TLS: &config.TLSConfig{
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
					AutoTLS:  false,
				},
			},
			{
				Name:           "test-upload",
				Host:           "127.0.0.1",
				Port:           9090,
				Protocol:       "http",
				Enabled:        true,
				MaxRequestBody: 100 * 1024 * 1024, // 100MB
				UseHttp2:       false,
			},
		},
	}

	fmt.Println("=== Sandwich 最大请求体配置测试 ===")
	fmt.Println()

	// 显示配置信息
	fmt.Println("服务器配置:")
	for i, server := range testConfig.Servers {
		fmt.Printf("  [%d] %s\n", i+1, server.Name)
		fmt.Printf("      地址: %s:%d\n", server.Host, server.Port)
		fmt.Printf("      协议: %s\n", server.Protocol)
		fmt.Printf("      启用: %t\n", server.Enabled)
		fmt.Printf("      最大请求体: %d bytes (%.2f MB)\n",
			server.MaxRequestBody, float64(server.MaxRequestBody)/(1024*1024))
		fmt.Printf("      HTTP/2: %t\n", server.UseHttp2)
		fmt.Println()
	}

	// 测试获取启用的服务器
	enabledServers := config.GetEnabledServers(testConfig)
	fmt.Printf("启用的服务器数量: %d\n", len(enabledServers))
	fmt.Println()

	// 生成配置文件
	configFile := "test_max_body_config.json"
	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		fmt.Printf("序列化配置失败: %v\n", err)
		return
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		fmt.Printf("写入配置文件失败: %v\n", err)
		return
	}

	fmt.Printf("测试配置已保存到: %s\n", configFile)
	fmt.Println()

	// 测试不同大小的配置值
	fmt.Println("常用配置值参考:")
	sizes := map[string]int64{
		"1MB":   1 * 1024 * 1024,
		"10MB":  10 * 1024 * 1024,
		"32MB":  32 * 1024 * 1024,
		"64MB":  64 * 1024 * 1024,
		"100MB": 100 * 1024 * 1024,
		"512MB": 512 * 1024 * 1024,
	}

	for name, size := range sizes {
		fmt.Printf("  %s: %d bytes\n", name, size)
	}

	fmt.Println()
	fmt.Println("=== 测试完成 ===")
	fmt.Println("注意: 在实际配置中使用这些数值时，请根据业务需求选择合适的大小")
}
