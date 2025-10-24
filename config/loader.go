/*
Project: Sandwich loader.go
Created: 2025/1/15 by Landers
Copyright Renj
*/

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sandwich/constant"
	"sandwich/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	cf    *Config
	Debug bool
)

func Get() *Config {
	if cf == nil {
		return new(Config)
	}
	return cf
}

func (c *Config) GetMiddle(name string) *MiddlewareConfig {
	for _, m := range c.Middleware {
		if m.Name == name {
			return &m
		}
	}
	return new(MiddlewareConfig)
}

// ConfigLoader 配置加载器结构体
type ConfigLoader struct {
	configType   string          // 配置类型
	configPath   string          // 配置文件路径
	config       *Config         // 当前配置
	mu           sync.RWMutex    // 读写锁
	watchers     []func(*Config) // 配置变更监听器列表
	lastModTime  time.Time       // 最后修改时间
	reloadTicker *time.Ticker    // 重载定时器
	stopChan     chan struct{}   // 停止信号
}

// NewConfigLoader 创建新的配置加载器
func NewConfigLoader(configPath string) *ConfigLoader {
	var configType string
	if filepath.Ext(configPath) == "" {
		configType = constant.JSON
	} else if strings.Contains(filepath.Ext(configPath), constant.YAML) {
		configType = constant.YAML
	}
	return &ConfigLoader{
		configType: configType,
		configPath: configPath,
		watchers:   make([]func(*Config), 0),
		stopChan:   make(chan struct{}),
	}
}

// LoadConfig 加载配置文件
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		cl.config = GetDefaultConfig()
		// 应用环境变量覆盖
		cl.applyEnvOverrides(cl.config)
		return cl.config, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	switch cl.configType {
	case constant.JSON:
		// 解析JSON配置
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	case constant.YAML:
		// 解析YAML配置
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	// 应用环境变量覆盖
	cl.applyEnvOverrides(&config)

	// 验证配置
	if err := cl.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	cl.config = &config

	// 更新文件修改时间
	if stat, err := os.Stat(cl.configPath); err == nil {
		cl.lastModTime = stat.ModTime()
	}

	Debug = config.Debug
	return cl.config, nil
}

func (cl *ConfigLoader) RegisterGlobalConfig(config *Config) {
	cl.mu.Lock()
	cf = config
	cl.mu.Unlock()
}

// GetConfig 获取当前配置（线程安全）
func (cl *ConfigLoader) GetConfig() *Config {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.config
}

// StartWatching 开始监控配置文件变化
func (cl *ConfigLoader) StartWatching(interval time.Duration) {
	cl.reloadTicker = time.NewTicker(interval)
	go cl.watchConfigFile()
}

// StopWatching 停止监控配置文件变化
func (cl *ConfigLoader) StopWatching() {
	if cl.reloadTicker != nil {
		cl.reloadTicker.Stop()
	}
	close(cl.stopChan)
}

// AddWatcher 添加配置变更监听器
func (cl *ConfigLoader) AddWatcher(watcher func(*Config)) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.watchers = append(cl.watchers, watcher)
}

// watchConfigFile 监控配置文件变化
func (cl *ConfigLoader) watchConfigFile() {
	for {
		select {
		case <-cl.reloadTicker.C:
			if cl.shouldReload() {
				if newConfig, err := cl.LoadConfig(); err == nil {
					cl.notifyWatchers(newConfig)
				}
			}
		case <-cl.stopChan:
			return
		}
	}
}

// shouldReload 检查是否需要重新加载配置
func (cl *ConfigLoader) shouldReload() bool {
	stat, err := os.Stat(cl.configPath)
	if err != nil {
		return false
	}
	return stat.ModTime().After(cl.lastModTime)
}

// notifyWatchers 通知所有监听器配置已变更
func (cl *ConfigLoader) notifyWatchers(config *Config) {
	cl.mu.RLock()
	watchers := make([]func(*Config), len(cl.watchers))
	copy(watchers, cl.watchers)
	cl.mu.RUnlock()

	for _, watcher := range watchers {
		watcher(config)
	}
}

// applyEnvOverrides 应用环境变量覆盖配置
func (cl *ConfigLoader) applyEnvOverrides(config *Config) {
	// 服务器配置覆盖
	if host := os.Getenv("SANDWICH_HOST"); host != "" {
		for i := range config.Servers {
			if config.Servers[i].Protocol == "http" {
				config.Servers[i].Host = host
				break
			}
		}
	}

	if port := os.Getenv("SANDWICH_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			for i := range config.Servers {
				if config.Servers[i].Protocol == "http" {
					config.Servers[i].Port = p
					break
				}
			}
		}
	}

	// 数据库配置覆盖
	if mongoUrl := os.Getenv("SANDWICH_MONGO_URL"); mongoUrl != "" {
		config.Database.Mongo.URL = mongoUrl
	}

	if dbName := os.Getenv("SANDWICH_DB_NAME"); dbName != "" {
		config.Database.Mongo.Database = dbName
	}

	// InfluxDB配置覆盖
	if influxUrl := os.Getenv("SANDWICH_INFLUX_URL"); influxUrl != "" {
		config.Database.Influx.URL = influxUrl
	}

	if influxToken := os.Getenv("SANDWICH_INFLUX_TOKEN"); influxToken != "" {
		config.Database.Influx.Token = influxToken
	}

	if influxOrg := os.Getenv("SANDWICH_INFLUX_ORG"); influxOrg != "" {
		config.Database.Influx.Org = influxOrg
	}

	if influxBucket := os.Getenv("SANDWICH_INFLUX_BUCKET"); influxBucket != "" {
		config.Database.Influx.Bucket = influxBucket
	}

	if influxPwd := os.Getenv("SANDWICH_INFLUX_PWD"); influxPwd != "" {
		config.Database.Influx.Password = influxPwd
	}

	if enableInflux := os.Getenv("SANDWICH_ENABLE_INFLUX"); enableInflux != "" {
		if enabled, err := strconv.ParseBool(enableInflux); err == nil {
			config.Database.Influx.Enabled = enabled
		}
	}

	// 功能特性配置覆盖
	if debug := os.Getenv("SANDWICH_DEBUG"); debug != "" {
		if enabled, err := strconv.ParseBool(debug); err == nil {
			// 可以添加debug配置到config结构体中
			_ = enabled
		}
	}

	if strictMode := os.Getenv("SANDWICH_STRICT_MODE"); strictMode != "" {
		if enabled, err := strconv.ParseBool(strictMode); err == nil {
			config.Security.StrictMode = enabled
		}
	}

	if gzip := os.Getenv("SANDWICH_GZIP"); gzip != "" {
		if enabled, err := strconv.ParseBool(gzip); err == nil {
			config.Features.Gzip.Enabled = enabled
		}
	}

	if cacheSize := os.Getenv("SANDWICH_CACHE_SIZE"); cacheSize != "" {
		if size, err := strconv.Atoi(cacheSize); err == nil {
			config.Features.Cache.Size = size
		}
	}

	// Helios配置覆盖
	if heliosAddr := os.Getenv("SANDWICH_HELIOS_ADDRESS"); heliosAddr != "" {
		// 可以添加helios配置到config结构体中
		_ = heliosAddr
	}

	if frontendFlag := os.Getenv("SANDWICH_FRONTEND_FLAG"); frontendFlag != "" {
		// 可以添加frontend配置到config结构体中
		_ = frontendFlag
	}

	if frontendHost := os.Getenv("SANDWICH_FRONTEND_HOST"); frontendHost != "" {
		// 可以添加frontend配置到config结构体中
		_ = frontendHost
	}

	if frontendPort := os.Getenv("SANDWICH_FRONTEND_PORT"); frontendPort != "" {
		if port, err := strconv.Atoi(frontendPort); err == nil {
			// 可以添加frontend配置到config结构体中
			_ = port
		}
	}

	// 限流配置覆盖
	if limit := os.Getenv("SANDWICH_LIMIT"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			config.Security.RateLimit = l
		}
	}

	// 熔断配置覆盖
	if breakerLimit := os.Getenv("SANDWICH_BREAKER_LIMIT"); breakerLimit != "" {
		if limit, err := strconv.Atoi(breakerLimit); err == nil {
			// 更新熔断中间件配置
			for i := range config.Middleware {
				if config.Middleware[i].Name == "circuit_breaker" {
					config.Middleware[i].Config["failure_threshold"] = limit
					break
				}
			}
		}
	}
}

// validateConfig 验证配置的有效性
func (cl *ConfigLoader) validateConfig(config *Config) error {
	// 验证服务器配置
	serverNames := make(map[string]bool)
	for i, server := range config.Servers {
		if server.Name == "" {
			return fmt.Errorf("服务器配置[%d]: 服务器名称不能为空", i)
		}

		if serverNames[server.Name] {
			return fmt.Errorf("服务器配置[%d]: 服务器名称 '%s' 重复", i, server.Name)
		}
		serverNames[server.Name] = true

		if server.Port <= 0 || server.Port > 65535 {
			return fmt.Errorf("服务器配置[%d]: 端口号 %d 无效", i, server.Port)
		}

		if !isValidProtocol(server.Protocol) {
			return fmt.Errorf("服务器配置[%d]: 协议 '%s' 不支持", i, server.Protocol)
		}

		// 验证HTTPS和HTTP3需要TLS配置
		if (server.Protocol == "https" || server.Protocol == "http3") && server.TLS == nil {
			return fmt.Errorf("服务器配置[%d]: %s协议需要TLS配置", i, server.Protocol)
		}

		// 验证TLS证书文件
		if server.TLS != nil && !server.TLS.AutoTLS {
			if server.TLS.CertFile == "" || server.TLS.KeyFile == "" {
				return fmt.Errorf("服务器配置[%d]: TLS证书文件路径不能为空", i)
			}
		}
	}

	// TODO 验证域名配置
	// 验证中间件配置
	middlewareNames := make(map[string]bool)
	for i, middleware := range config.Middleware {
		if middleware.Name == "" {
			return fmt.Errorf("中间件配置[%d]: 中间件名称不能为空", i)
		}

		if middlewareNames[middleware.Name] {
			return fmt.Errorf("中间件配置[%d]: 中间件名称 '%s' 重复", i, middleware.Name)
		}
		middlewareNames[middleware.Name] = true
	}

	return nil
}

// isValidProtocol 检查协议是否有效
func isValidProtocol(protocol string) bool {
	validProtocols := []string{"http", "https", "http3"}
	for _, p := range validProtocols {
		if p == protocol {
			return true
		}
	}
	return false
}

// isValidLoadBalanceAlgorithm 检查负载均衡算法是否有效
func isValidLoadBalanceAlgorithm(algorithm string) bool {
	validAlgorithms := []string{"random", "round_robin", "weighted_round_robin", "least_connections"}
	for _, a := range validAlgorithms {
		if a == algorithm {
			return true
		}
	}
	return false
}

// SaveConfig 保存配置到文件
func (cl *ConfigLoader) SaveConfig(config *Config) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// 验证配置
	if err := cl.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(cl.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(cl.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	cl.config = config
	return nil
}

// MergeConfig 合并配置（用于部分更新）
func (cl *ConfigLoader) MergeConfig(updates map[string]interface{}) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.config == nil {
		return fmt.Errorf("当前配置为空，无法合并")
	}

	// 这里可以实现更复杂的配置合并逻辑
	// 目前简单实现，实际使用时可以根据需要扩展
	for key, value := range updates {
		switch key {
		case "debug":
			if debug, ok := value.(bool); ok {
				// 可以添加debug字段到config结构体
				_ = debug
			}
		case "strict_mode":
			if strictMode, ok := value.(bool); ok {
				cl.config.Security.StrictMode = strictMode
			}
			// 可以添加更多字段的合并逻辑
		}
	}

	return nil
}

// GetConfigSummary 获取配置摘要信息
func (cl *ConfigLoader) GetConfigSummary() map[string]interface{} {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if cl.config == nil {
		return nil
	}

	summary := map[string]interface{}{
		"servers_count":     len(cl.config.Servers),
		"domains_count":     len(GetDomains(cl.config)),
		"middleware_count":  len(cl.config.Middleware),
		"http3_enabled":     cl.config.Features.HTTP3.Enabled,
		"websocket_enabled": cl.config.Features.WebSocket.Enabled,
		"influx_enabled":    cl.config.Database.Influx.Enabled,
		"monitor_enabled":   cl.config.Monitor.Enabled,
	}

	// 添加服务器信息
	servers := make([]map[string]interface{}, len(cl.config.Servers))
	for i, server := range cl.config.Servers {
		servers[i] = map[string]interface{}{
			"name":     server.Name,
			"protocol": server.Protocol,
			"port":     server.Port,
			"enabled":  server.Enabled,
		}
	}
	summary["servers"] = servers

	return summary
}
