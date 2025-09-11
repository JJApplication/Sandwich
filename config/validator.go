/*
Project: Sandwich validator.go
Created: 2025/1/15 by Landers
Copyright Renj
*/

package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError 配置验证错误
type ValidationError struct {
	Field   string // 字段名
	Value   string // 字段值
	Message string // 错误信息
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("字段 '%s' 验证失败: %s (值: %s)", e.Field, e.Message, e.Value)
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool               // 是否有效
	Errors   []*ValidationError // 错误列表
	Warnings []string           // 警告列表
}

// AddError 添加错误
func (vr *ValidationResult) AddError(field, value, message string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// AddWarning 添加警告
func (vr *ValidationResult) AddWarning(message string) {
	vr.Warnings = append(vr.Warnings, message)
}

// ConfigValidator 配置验证器
type ConfigValidator struct {
	strict bool // 是否启用严格模式
}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator(strict bool) *ConfigValidator {
	return &ConfigValidator{
		strict: strict,
	}
}

// ValidateConfig 验证完整配置
func (cv *ConfigValidator) ValidateConfig(config *Config) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// 验证服务器配置
	cv.validateServers(config.Servers, result)

	// 验证域名配置
	cv.validateDomains(config, result)

	// 验证中间件配置
	cv.validateMiddleware(config.Middleware, result)

	// 验证功能特性配置
	cv.validateFeatures(config.Features, result)

	// 验证数据库配置
	cv.validateDatabase(config.Database, result)

	// 验证监控配置
	cv.validateMonitor(config.Monitor, result)

	// 验证安全配置
	cv.validateSecurity(config.Security, result)

	// 验证配置间的依赖关系
	cv.validateDependencies(config, result)

	return result
}

// validateServers 验证服务器配置
func (cv *ConfigValidator) validateServers(servers []ServerConfig, result *ValidationResult) {
	if len(servers) == 0 {
		result.AddError("servers", "[]", "至少需要配置一个服务器")
		return
	}

	serverNames := make(map[string]bool)
	portProtocolMap := make(map[string]string) // port:protocol -> server_name

	for i, server := range servers {
		prefix := fmt.Sprintf("servers[%d]", i)

		// 验证服务器名称
		if server.Name == "" {
			result.AddError(prefix+".name", "", "服务器名称不能为空")
		} else if serverNames[server.Name] {
			result.AddError(prefix+".name", server.Name, "服务器名称重复")
		} else {
			serverNames[server.Name] = true
		}

		// 验证主机地址
		if server.Host == "" {
			result.AddError(prefix+".host", "", "主机地址不能为空")
		} else if !cv.isValidHost(server.Host) {
			result.AddError(prefix+".host", server.Host, "主机地址格式无效")
		}

		// 验证端口
		if server.Port <= 0 || server.Port > 65535 {
			result.AddError(prefix+".port", strconv.Itoa(server.Port), "端口号必须在1-65535之间")
		} else {
			// 检查端口冲突（同一端口不能同时用于不同协议）
			portKey := fmt.Sprintf("%s:%d", server.Host, server.Port)
			if existingServer, exists := portProtocolMap[portKey]; exists {
				if server.Protocol != servers[i].Protocol {
					result.AddError(prefix+".port", strconv.Itoa(server.Port),
						fmt.Sprintf("端口冲突，已被服务器 '%s' 使用", existingServer))
				}
			} else {
				portProtocolMap[portKey] = server.Name
			}
		}

		// 验证协议
		if !cv.isValidProtocol(server.Protocol) {
			result.AddError(prefix+".protocol", server.Protocol, "不支持的协议类型")
		}

		// 验证TLS配置
		if server.Protocol == "https" || server.Protocol == "http3" {
			if server.TLS == nil {
				result.AddError(prefix+".tls", "null", fmt.Sprintf("%s协议需要TLS配置", server.Protocol))
			} else {
				cv.validateTLS(*server.TLS, prefix+".tls", result)
			}
		}

		// 检查特殊端口使用
		if server.Port < 1024 && cv.strict {
			result.AddWarning(fmt.Sprintf("服务器 '%s' 使用特权端口 %d，需要管理员权限", server.Name, server.Port))
		}
	}
}

// validateTLS 验证TLS配置
func (cv *ConfigValidator) validateTLS(tls TLSConfig, prefix string, result *ValidationResult) {
	if !tls.AutoTLS {
		// 验证证书文件
		if tls.CertFile == "" {
			result.AddError(prefix+".cert_file", "", "证书文件路径不能为空")
		} else if !cv.fileExists(tls.CertFile) {
			result.AddError(prefix+".cert_file", tls.CertFile, "证书文件不存在")
		}

		// 验证私钥文件
		if tls.KeyFile == "" {
			result.AddError(prefix+".key_file", "", "私钥文件路径不能为空")
		} else if !cv.fileExists(tls.KeyFile) {
			result.AddError(prefix+".key_file", tls.KeyFile, "私钥文件不存在")
		}
	}
}

// validateDomains 验证域名配置
func (cv *ConfigValidator) validateDomains(config *Config, result *ValidationResult) {
	domains := GetDomains(config)
	if len(domains) == 0 {
		result.AddWarning("未配置任何域名，服务器将无法处理请求")
		return
	}

	serverNames := make(map[string]bool)
	for _, server := range config.Servers {
		serverNames[server.Name] = true
	}

	domainNames := make(map[string]bool)

	for i, domain := range domains {
		prefix := fmt.Sprintf("domains[%d]", i)
		// 验证域名
		if domain == "" {
			result.AddError(prefix+".domain", "", "域名不能为空")
		} else if domainNames[domain] {
			result.AddError(prefix+".domain", domain, "域名重复")
		} else if !cv.isValidDomain(domain) {
			result.AddError(prefix+".domain", domain, "域名格式无效")
		} else {
			domainNames[domain] = true
		}
	}
}

// validateMiddleware 验证中间件配置
func (cv *ConfigValidator) validateMiddleware(middleware []MiddlewareConfig, result *ValidationResult) {
	middlewareNames := make(map[string]bool)
	orders := make(map[int]string)

	for i, mw := range middleware {
		prefix := fmt.Sprintf("middleware[%d]", i)

		// 验证中间件名称
		if mw.Name == "" {
			result.AddError(prefix+".name", "", "中间件名称不能为空")
		} else if middlewareNames[mw.Name] {
			result.AddError(prefix+".name", mw.Name, "中间件名称重复")
		} else {
			middlewareNames[mw.Name] = true
		}

		// 验证执行顺序
		if existingMw, exists := orders[mw.Order]; exists {
			result.AddError(prefix+".order", strconv.Itoa(mw.Order),
				fmt.Sprintf("执行顺序冲突，已被中间件 '%s' 使用", existingMw))
		} else {
			orders[mw.Order] = mw.Name
		}

		// 验证特定中间件的配置
		cv.validateSpecificMiddleware(mw, prefix, result)
	}
}

// validateSpecificMiddleware 验证特定中间件的配置
func (cv *ConfigValidator) validateSpecificMiddleware(mw MiddlewareConfig, prefix string, result *ValidationResult) {
	switch mw.Name {
	case "rate_limit":
		if rps, ok := mw.Config["requests_per_second"]; ok {
			if _, ok := rps.(float64); ok {
				rpsInt := int(rps.(float64))
				if rpsInt <= 0 {
					result.AddError(prefix+".config.requests_per_second", strconv.Itoa(rpsInt), "请求速率必须大于0")
				}
			} else {
				result.AddError(prefix+".config.requests_per_second", fmt.Sprintf("%v", rps), "请求速率必须是整数")
			}
		}

	case "circuit_breaker":
		if threshold, ok := mw.Config["failure_threshold"]; ok {
			if _, ok := threshold.(float64); ok {
				thresholdInt := int(threshold.(float64))
				if thresholdInt <= 0 {
					result.AddError(prefix+".config.failure_threshold", strconv.Itoa(thresholdInt), "失败阈值必须大于0")
				}
			} else {
				result.AddError(prefix+".config.failure_threshold", fmt.Sprintf("%v", threshold), "失败阈值必须是整数")
			}
		}
	}
}

// validateFeatures 验证功能特性配置
func (cv *ConfigValidator) validateFeatures(features FeatureConfig, result *ValidationResult) {
	// 验证HTTP/3配置
	if features.HTTP3.Enabled {
		if features.HTTP3.MaxConnections <= 0 {
			result.AddError("features.http3.max_connections", strconv.Itoa(features.HTTP3.MaxConnections), "最大连接数必须大于0")
		}
	}

	// 验证WebSocket配置
	if features.WebSocket.Enabled {
		if features.WebSocket.MaxMessageSize <= 0 {
			result.AddError("features.websocket.max_message_size", strconv.FormatInt(features.WebSocket.MaxMessageSize, 10), "最大消息大小必须大于0")
		}
		if features.WebSocket.BufferSize <= 0 {
			result.AddError("features.websocket.buffer_size", strconv.Itoa(features.WebSocket.BufferSize), "缓冲区大小必须大于0")
		}
	}

	// 验证Gzip配置
	if features.Gzip.Enabled {
		if features.Gzip.Level < 1 || features.Gzip.Level > 9 {
			result.AddError("features.gzip.level", strconv.Itoa(features.Gzip.Level), "压缩级别必须在1-9之间")
		}
	}

	// 验证缓存配置
	if features.Cache.Enabled {
		if features.Cache.Size <= 0 {
			result.AddError("features.cache.size", strconv.Itoa(features.Cache.Size), "缓存大小必须大于0")
		}
		if !cv.isValidCacheStrategy(features.Cache.Strategy) {
			result.AddError("features.cache.strategy", features.Cache.Strategy, "不支持的缓存策略")
		}
	}
}

// validateDatabase 验证数据库配置
func (cv *ConfigValidator) validateDatabase(database DatabaseConfig, result *ValidationResult) {
	// 验证MongoDB配置
	if database.Mongo.URL == "" {
		result.AddError("database.mongo.url", "", "MongoDB连接URL不能为空")
	} else if !cv.isValidMongoURL(database.Mongo.URL) {
		result.AddError("database.mongo.url", database.Mongo.URL, "MongoDB连接URL格式无效")
	}

	if database.Mongo.Database == "" {
		result.AddError("database.mongo.database", "", "MongoDB数据库名称不能为空")
	}

	// 验证InfluxDB配置
	if database.Influx.Enabled {
		if database.Influx.URL == "" {
			result.AddError("database.influx.url", "", "InfluxDB连接URL不能为空")
		} else if !cv.isValidURL(database.Influx.URL) {
			result.AddError("database.influx.url", database.Influx.URL, "InfluxDB连接URL格式无效")
		}

		if database.Influx.Org == "" {
			result.AddError("database.influx.org", "", "InfluxDB组织名称不能为空")
		}

		if database.Influx.Bucket == "" {
			result.AddError("database.influx.bucket", "", "InfluxDB存储桶名称不能为空")
		}
	}
}

// validateMonitor 验证监控配置
func (cv *ConfigValidator) validateMonitor(monitor MonitorConfig, result *ValidationResult) {
	if monitor.Enabled {
		if monitor.Port <= 0 || monitor.Port > 65535 {
			result.AddError("monitor.port", strconv.Itoa(monitor.Port), "监控端口必须在1-65535之间")
		}

		if monitor.Path == "" {
			result.AddError("monitor.path", "", "监控路径不能为空")
		} else if !strings.HasPrefix(monitor.Path, "/") {
			result.AddError("monitor.path", monitor.Path, "监控路径必须以/开头")
		}
	}
}

// validateSecurity 验证安全配置
func (cv *ConfigValidator) validateSecurity(security SecurityConfig, result *ValidationResult) {
	// 验证IP白名单
	for i, ip := range security.AllowIPs {
		if !cv.isValidIP(ip) {
			result.AddError(fmt.Sprintf("security.allow_ips[%d]", i), ip, "IP地址格式无效")
		}
	}

	// 验证IP黑名单
	for i, ip := range security.DenyIPs {
		if !cv.isValidIP(ip) {
			result.AddError(fmt.Sprintf("security.deny_ips[%d]", i), ip, "IP地址格式无效")
		}
	}

	// 验证速率限制
	if security.RateLimit < 0 {
		result.AddError("security.rate_limit", strconv.Itoa(security.RateLimit), "速率限制不能为负数")
	}
}

// validateDependencies 验证配置间的依赖关系
func (cv *ConfigValidator) validateDependencies(config *Config, result *ValidationResult) {
	// 检查HTTP/3功能是否有对应的服务器
	if config.Features.HTTP3.Enabled {
		hasHTTP3Server := false
		for _, server := range config.Servers {
			if server.Protocol == "http3" && server.Enabled {
				hasHTTP3Server = true
				break
			}
		}
		if !hasHTTP3Server {
			result.AddWarning("启用了HTTP/3功能但没有配置HTTP/3服务器")
		}
	}

	// 检查WebSocket功能是否有对应的域名配置
	if config.Features.WebSocket.Enabled {
		hasWebSocketDomain := false
		for _, server := range config.Servers {
			if hasWebSocketDomain {
				break
			}
			if server.Protocol == "websocket" {
				hasWebSocketDomain = true
				break
			}
		}
		if !hasWebSocketDomain {
			result.AddWarning("启用了WebSocket功能但没有配置WebSocket域名")
		}
	}

	// 检查监控端口是否与服务器端口冲突
	if config.Monitor.Enabled {
		for _, server := range config.Servers {
			if server.Port == config.Monitor.Port {
				result.AddError("monitor.port", strconv.Itoa(config.Monitor.Port),
					fmt.Sprintf("监控端口与服务器 '%s' 端口冲突", server.Name))
			}
		}
	}
}

// 辅助验证方法

// isValidHost 验证主机地址是否有效
func (cv *ConfigValidator) isValidHost(host string) bool {
	if host == "localhost" || host == "0.0.0.0" {
		return true
	}
	return net.ParseIP(host) != nil
}

// isValidProtocol 验证协议是否有效
func (cv *ConfigValidator) isValidProtocol(protocol string) bool {
	validProtocols := []string{"http", "https", "http3"}
	for _, p := range validProtocols {
		if p == protocol {
			return true
		}
	}
	return false
}

// isValidDomain 验证域名格式是否有效
func (cv *ConfigValidator) isValidDomain(domain string) bool {
	// 简单的域名格式验证
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?))*$`)
	return domainRegex.MatchString(domain)
}

// isValidBackend 验证后端服务地址是否有效
func (cv *ConfigValidator) isValidBackend(backend string) bool {
	// 格式: host:port
	parts := strings.Split(backend, ":")
	if len(parts) != 2 {
		return false
	}

	host, portStr := parts[0], parts[1]

	// 验证主机
	if host == "" || (!cv.isValidHost(host) && !cv.isValidDomain(host)) {
		return false
	}

	// 验证端口
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return false
	}

	return true
}

// isValidLoadBalanceAlgorithm 验证负载均衡算法是否有效
func (cv *ConfigValidator) isValidLoadBalanceAlgorithm(algorithm string) bool {
	validAlgorithms := []string{"random", "round_robin", "weighted_round_robin", "least_connections"}
	for _, a := range validAlgorithms {
		if a == algorithm {
			return true
		}
	}
	return false
}

// isValidCacheStrategy 验证缓存策略是否有效
func (cv *ConfigValidator) isValidCacheStrategy(strategy string) bool {
	validStrategies := []string{"lru", "lfu", "fifo"}
	for _, s := range validStrategies {
		if s == strategy {
			return true
		}
	}
	return false
}

// isValidMongoURL 验证MongoDB连接URL是否有效
func (cv *ConfigValidator) isValidMongoURL(mongoURL string) bool {
	return strings.HasPrefix(mongoURL, "mongodb://") || strings.HasPrefix(mongoURL, "mongodb+srv://")
}

// isValidURL 验证URL格式是否有效
func (cv *ConfigValidator) isValidURL(urlStr string) bool {
	_, err := url.Parse(urlStr)
	return err == nil
}

// isValidIP 验证IP地址是否有效
func (cv *ConfigValidator) isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// fileExists 检查文件是否存在
func (cv *ConfigValidator) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
