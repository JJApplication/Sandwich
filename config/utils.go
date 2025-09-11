package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// MergeConfigs 合并多个配置，后面的配置会覆盖前面的配置
// 支持深度合并，不会完全替换嵌套结构
func MergeConfigs(configs ...*Config) (*Config, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("至少需要一个配置")
	}

	// 深拷贝第一个配置作为基础
	result, err := DeepCopyConfig(configs[0])
	if err != nil {
		return nil, fmt.Errorf("复制基础配置失败: %v", err)
	}

	// 依次合并其他配置
	for i := 1; i < len(configs); i++ {
		if err := mergeConfigInto(result, configs[i]); err != nil {
			return nil, fmt.Errorf("合并第 %d 个配置失败: %v", i+1, err)
		}
	}

	return result, nil
}

// DeepCopyConfig 深拷贝配置结构
func DeepCopyConfig(src *Config) (*Config, error) {
	if src == nil {
		return nil, nil
	}

	// 使用 JSON 序列化/反序列化进行深拷贝
	data, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %v", err)
	}

	var dst Config
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, fmt.Errorf("反序列化配置失败: %v", err)
	}

	return &dst, nil
}

// mergeConfigInto 将 src 配置合并到 dst 配置中
func mergeConfigInto(dst, src *Config) error {
	if src == nil {
		return nil
	}

	// 合并服务器配置
	if len(src.Servers) > 0 {
		dst.Servers = append(dst.Servers, src.Servers...)
	}

	// 合并中间件配置
	if len(src.Middleware) > 0 {
		dst.Middleware = append(dst.Middleware, src.Middleware...)
	}

	// 合并功能配置
	mergeFeatures(&dst.Features, &src.Features)

	// 合并数据库配置
	mergeDatabaseConfig(&dst.Database, &src.Database)

	// 合并监控配置
	mergeMonitorConfig(&dst.Monitor, &src.Monitor)

	// 合并安全配置
	mergeSecurityConfig(&dst.Security, &src.Security)

	// 覆盖简单字段
	if src.Debug {
		dst.Debug = src.Debug
	}

	return nil
}

// mergeFeatures 合并功能配置
func mergeFeatures(dst, src *FeatureConfig) {
	if src.HTTP3.Enabled {
		dst.HTTP3 = src.HTTP3
	}
	if src.WebSocket.Enabled {
		dst.WebSocket = src.WebSocket
	}
	if src.Gzip.Enabled {
		dst.Gzip = src.Gzip
	}
	if src.Cache.Enabled {
		dst.Cache = src.Cache
	}
}

// mergeDatabaseConfig 合并数据库配置
func mergeDatabaseConfig(dst, src *DatabaseConfig) {
	if src.Mongo.URL != "" {
		dst.Mongo = src.Mongo
	}
	if src.Influx.Enabled {
		dst.Influx = src.Influx
	}
}

// mergeMonitorConfig 合并监控配置
func mergeMonitorConfig(dst, src *MonitorConfig) {
	if src.Enabled {
		*dst = *src
	}
}

// mergeSecurityConfig 合并安全配置
func mergeSecurityConfig(dst, src *SecurityConfig) {
	if src.StrictMode {
		dst.StrictMode = src.StrictMode
	}
	if len(src.AllowIPs) > 0 {
		dst.AllowIPs = src.AllowIPs
	}
	if len(src.DenyIPs) > 0 {
		dst.DenyIPs = src.DenyIPs
	}
	if src.RateLimit > 0 {
		dst.RateLimit = src.RateLimit
	}
}

// FindServerByName 根据名称查找服务器配置
func FindServerByName(config *Config, name string) *ServerConfig {
	for i := range config.Servers {
		if config.Servers[i].Name == name {
			return &config.Servers[i]
		}
	}
	return nil
}

// FindMiddlewareByName 根据名称查找中间件配置
func FindMiddlewareByName(config *Config, name string) *MiddlewareConfig {
	for i := range config.Middleware {
		if config.Middleware[i].Name == name {
			return &config.Middleware[i]
		}
	}
	return nil
}

// GetEnabledServers 获取所有启用的服务器配置
func GetEnabledServers(config *Config) []ServerConfig {
	var enabled []ServerConfig
	for _, server := range config.Servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}

// GetEnabledMiddleware 获取所有启用的中间件配置，按执行顺序排序
func GetEnabledMiddleware(config *Config) []MiddlewareConfig {
	var enabled []MiddlewareConfig
	for _, middleware := range config.Middleware {
		if middleware.Enabled {
			enabled = append(enabled, middleware)
		}
	}

	// 按执行顺序排序
	for i := 0; i < len(enabled)-1; i++ {
		for j := i + 1; j < len(enabled); j++ {
			if enabled[i].Order > enabled[j].Order {
				enabled[i], enabled[j] = enabled[j], enabled[i]
			}
		}
	}

	return enabled
}

// GetServersByProtocol 根据协议类型获取服务器配置
func GetServersByProtocol(config *Config, protocol string) []ServerConfig {
	var servers []ServerConfig
	for _, server := range config.Servers {
		if server.Enabled && strings.EqualFold(server.Protocol, protocol) {
			servers = append(servers, server)
		}
	}
	return servers
}

// ValidateConfigConsistency 验证配置的一致性
func ValidateConfigConsistency(config *Config) []string {
	var warnings []string

	// 检查域名配置中引用的服务器是否存在
	serverNames := make(map[string]bool)
	for _, server := range config.Servers {
		serverNames[server.Name] = true
	}

	// 检查端口冲突
	portMap := make(map[string][]string) // host:port -> server names
	for _, server := range config.Servers {
		if !server.Enabled {
			continue
		}
		key := fmt.Sprintf("%s:%d", server.Host, server.Port)
		portMap[key] = append(portMap[key], server.Name)
	}

	for hostPort, servers := range portMap {
		if len(servers) > 1 {
			warnings = append(warnings, fmt.Sprintf("端口冲突 %s: %v", hostPort, servers))
		}
	}

	// 检查 HTTPS 服务器的 TLS 配置
	for _, server := range config.Servers {
		if !server.Enabled {
			continue
		}
		if strings.EqualFold(server.Protocol, "https") || strings.EqualFold(server.Protocol, "http3") {
			if server.TLS == nil {
				warnings = append(warnings, fmt.Sprintf("HTTPS/HTTP3 服务器 %s 缺少 TLS 配置", server.Name))
			} else if !server.TLS.AutoTLS && (server.TLS.CertFile == "" || server.TLS.KeyFile == "") {
				warnings = append(warnings, fmt.Sprintf("HTTPS/HTTP3 服务器 %s 缺少证书文件配置", server.Name))
			}
		}
	}

	// 检查中间件顺序
	orders := make(map[int][]string)
	for _, middleware := range config.Middleware {
		if middleware.Enabled {
			orders[middleware.Order] = append(orders[middleware.Order], middleware.Name)
		}
	}

	for order, middlewares := range orders {
		if len(middlewares) > 1 {
			warnings = append(warnings, fmt.Sprintf("中间件执行顺序冲突 %d: %v", order, middlewares))
		}
	}

	return warnings
}

// NormalizeConfig 标准化配置，设置默认值和修正不合理的配置
func NormalizeConfig(config *Config) {
	// 标准化服务器配置
	for i := range config.Servers {
		server := &config.Servers[i]

		// 设置默认名称
		if server.Name == "" {
			server.Name = fmt.Sprintf("server-%d", i+1)
		}

		// 标准化协议名称
		server.Protocol = strings.ToLower(server.Protocol)

		// 设置默认端口
		if server.Port == 0 {
			switch server.Protocol {
			case "http":
				server.Port = 80
			case "https", "http3":
				server.Port = 443
			default:
				server.Port = 8080
			}
		}

		// 设置默认主机
		if server.Host == "" {
			server.Host = "0.0.0.0"
		}
	}

	// 标准化中间件配置
	for i := range config.Middleware {
		middleware := &config.Middleware[i]

		// 设置默认执行顺序
		if middleware.Order == 0 {
			middleware.Order = i + 1
		}
	}

	// 标准化功能配置
	normalizeFeatures(&config.Features)

	// 标准化监控配置
	if config.Monitor.Port == 0 {
		config.Monitor.Port = 9090
	}
	if config.Monitor.Path == "" {
		config.Monitor.Path = "/metrics"
	}
	if config.Monitor.Interval == 0 {
		config.Monitor.Interval = 15
	}
}

// normalizeFeatures 标准化功能配置
func normalizeFeatures(features *FeatureConfig) {
	// HTTP/3 默认配置
	if features.HTTP3.MaxConnections == 0 {
		features.HTTP3.MaxConnections = 10000
	}
	if features.HTTP3.IdleTimeout == 0 {
		features.HTTP3.IdleTimeout = 60
	}
	if features.HTTP3.KeepAlive == 0 {
		features.HTTP3.KeepAlive = 30
	}

	// WebSocket 默认配置
	if features.WebSocket.PingInterval == 0 {
		features.WebSocket.PingInterval = 30
	}
	if features.WebSocket.PongTimeout == 0 {
		features.WebSocket.PongTimeout = 10
	}
	if features.WebSocket.MaxMessageSize == 0 {
		features.WebSocket.MaxMessageSize = 1048576 // 1MB
	}
	if features.WebSocket.BufferSize == 0 {
		features.WebSocket.BufferSize = 4096
	}

	// Gzip 默认配置
	if features.Gzip.Level == 0 {
		features.Gzip.Level = 6
	}
	if len(features.Gzip.Types) == 0 {
		features.Gzip.Types = []string{
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
		}
	}

	// Cache 默认配置
	if features.Cache.Size == 0 {
		features.Cache.Size = 10000
	}
	if features.Cache.TTL == 0 {
		features.Cache.TTL = 60
	}
	if features.Cache.Strategy == "" {
		features.Cache.Strategy = "lru"
	}
}

// CompareConfigs 比较两个配置的差异
func CompareConfigs(old, new *Config) map[string]interface{} {
	diff := make(map[string]interface{})

	// 使用反射比较结构体
	oldVal := reflect.ValueOf(old).Elem()
	newVal := reflect.ValueOf(new).Elem()

	compareValues(oldVal, newVal, "", diff)

	return diff
}

// compareValues 递归比较两个反射值
func compareValues(oldVal, newVal reflect.Value, path string, diff map[string]interface{}) {
	if oldVal.Type() != newVal.Type() {
		diff[path] = map[string]interface{}{
			"old": oldVal.Interface(),
			"new": newVal.Interface(),
		}
		return
	}

	switch oldVal.Kind() {
	case reflect.Struct:
		for i := 0; i < oldVal.NumField(); i++ {
			fieldName := oldVal.Type().Field(i).Name
			fieldPath := path
			if fieldPath != "" {
				fieldPath += "."
			}
			fieldPath += fieldName

			compareValues(oldVal.Field(i), newVal.Field(i), fieldPath, diff)
		}
	case reflect.Slice:
		if oldVal.Len() != newVal.Len() {
			diff[path+".length"] = map[string]interface{}{
				"old": oldVal.Len(),
				"new": newVal.Len(),
			}
		}
		minLen := oldVal.Len()
		if newVal.Len() < minLen {
			minLen = newVal.Len()
		}
		for i := 0; i < minLen; i++ {
			compareValues(oldVal.Index(i), newVal.Index(i), fmt.Sprintf("%s[%d]", path, i), diff)
		}
	default:
		if !reflect.DeepEqual(oldVal.Interface(), newVal.Interface()) {
			diff[path] = map[string]interface{}{
				"old": oldVal.Interface(),
				"new": newVal.Interface(),
			}
		}
	}
}

// GetDomains 使用Set存储域名列表
func GetDomains(config *Config) []string {
	var domains []string
	for _, serverConfig := range config.Servers {
		for _, domainConfig := range serverConfig.DomainConfig {
			for _, domain := range domainConfig.Domains {
				domains = append(domains, domain)
			}
		}
	}

	return domains
}

func GetWsPort(config *Config) string {
	for _, serverConfig := range config.Servers {
		for _, domainConfig := range serverConfig.DomainConfig {
			if domainConfig.UseWebsocket {
				return strconv.Itoa(serverConfig.Port)
			}
		}
	}

	return ""
}
