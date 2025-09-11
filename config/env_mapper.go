package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvMapper 环境变量映射器
// 负责将环境变量映射到配置结构体字段
type EnvMapper struct {
	mappings map[string]string // 环境变量名到配置路径的映射
}

// NewEnvMapper 创建新的环境变量映射器
func NewEnvMapper() *EnvMapper {
	return &EnvMapper{
		mappings: getDefaultMappings(),
	}
}

// getDefaultMappings 获取默认的环境变量映射关系
func getDefaultMappings() map[string]string {
	return map[string]string{
		// 服务器配置
		"SANDWICH_HOST":     "servers.0.host",
		"SANDWICH_PORT":     "servers.0.port",
		"SANDWICH_PROTOCOL": "servers.0.protocol",

		// HTTPS 配置
		"SANDWICH_TLS_CERT": "servers.1.tls.cert_file",
		"SANDWICH_TLS_KEY":  "servers.1.tls.key_file",
		"SANDWICH_AUTO_TLS": "servers.1.tls.auto_tls",

		// 数据库配置
		"SANDWICH_MONGO_URL":     "database.mongo.url",
		"SANDWICH_DB_NAME":       "database.mongo.database",
		"SANDWICH_MONGO_TIMEOUT": "database.mongo.timeout",
		"SANDWICH_INFLUX_URL":    "database.influx.url",
		"SANDWICH_INFLUX_TOKEN":  "database.influx.token",
		"SANDWICH_INFLUX_ORG":    "database.influx.org",
		"SANDWICH_INFLUX_BUCKET": "database.influx.bucket",
		"SANDWICH_INFLUX_PWD":    "database.influx.password",
		"SANDWICH_ENABLE_INFLUX": "database.influx.enabled",

		// 功能配置
		"SANDWICH_DEBUG":       "debug",
		"SANDWICH_STRICT_MODE": "security.strict_mode",
		"SANDWICH_GZIP":        "features.gzip.enabled",
		"SANDWICH_CACHE_SIZE":  "features.cache.size",

		// HTTP/3 配置
		"SANDWICH_HTTP3_ENABLED":         "features.http3.enabled",
		"SANDWICH_HTTP3_MAX_CONNECTIONS": "features.http3.max_connections",
		"SANDWICH_HTTP3_IDLE_TIMEOUT":    "features.http3.idle_timeout",
		"SANDWICH_HTTP3_KEEP_ALIVE":      "features.http3.keep_alive",

		// WebSocket 配置
		"SANDWICH_WS_ENABLED":          "features.websocket.enabled",
		"SANDWICH_WS_PING_INTERVAL":    "features.websocket.ping_interval",
		"SANDWICH_WS_PONG_TIMEOUT":     "features.websocket.pong_timeout",
		"SANDWICH_WS_MAX_MESSAGE_SIZE": "features.websocket.max_message_size",
		"SANDWICH_WS_BUFFER_SIZE":      "features.websocket.buffer_size",

		// 限流和熔断
		"SANDWICH_LIMIT":         "security.rate_limit",
		"SANDWICH_BREAKER_LIMIT": "middleware.circuit_breaker.config.failure_threshold",

		// 监控配置
		"SANDWICH_MONITOR_ENABLED": "monitor.enabled",
		"SANDWICH_MONITOR_PORT":    "monitor.port",
		"SANDWICH_MONITOR_PATH":    "monitor.path",
	}
}

// ApplyEnvOverrides 应用环境变量覆盖到配置
func (em *EnvMapper) ApplyEnvOverrides(config *Config) error {
	for envKey, configPath := range em.mappings {
		envValue := os.Getenv(envKey)
		if envValue == "" {
			continue
		}

		if err := em.setConfigValue(config, configPath, envValue); err != nil {
			return err
		}
	}

	return nil
}

// setConfigValue 根据路径设置配置值
func (em *EnvMapper) setConfigValue(config *Config, path, value string) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "servers":
		return em.setServerValue(config, parts[1:], value)
	case "database":
		return em.setDatabaseValue(config, parts[1:], value)
	case "features":
		return em.setFeaturesValue(config, parts[1:], value)
	case "security":
		return em.setSecurityValue(config, parts[1:], value)
	case "monitor":
		return em.setMonitorValue(config, parts[1:], value)
	case "middleware":
		return em.setMiddlewareValue(config, parts[1:], value)
	case "debug":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			config.Debug = boolVal
		}
	}

	return nil
}

// setServerValue 设置服务器配置值
func (em *EnvMapper) setServerValue(config *Config, parts []string, value string) error {
	if len(parts) < 2 {
		return nil
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil || index >= len(config.Servers) {
		return nil
	}

	server := &config.Servers[index]
	switch parts[1] {
	case "host":
		server.Host = value
	case "port":
		if port, err := strconv.Atoi(value); err == nil {
			server.Port = port
		}
	case "protocol":
		server.Protocol = value
	case "tls":
		if len(parts) >= 3 {
			if server.TLS == nil {
				server.TLS = &TLSConfig{}
			}
			switch parts[2] {
			case "cert_file":
				server.TLS.CertFile = value
			case "key_file":
				server.TLS.KeyFile = value
			case "auto_tls":
				if boolVal, err := strconv.ParseBool(value); err == nil {
					server.TLS.AutoTLS = boolVal
				}
			}
		}
	}

	return nil
}

// setDatabaseValue 设置数据库配置值
func (em *EnvMapper) setDatabaseValue(config *Config, parts []string, value string) error {
	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "mongo":
		switch parts[1] {
		case "url":
			config.Database.Mongo.URL = value
		case "database":
			config.Database.Mongo.Database = value
		case "timeout":
			if duration, err := time.ParseDuration(value); err == nil {
				config.Database.Mongo.Timeout = int(duration)
			}
		}
	case "influx":
		switch parts[1] {
		case "enabled":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				config.Database.Influx.Enabled = boolVal
			}
		case "url":
			config.Database.Influx.URL = value
		case "token":
			config.Database.Influx.Token = value
		case "org":
			config.Database.Influx.Org = value
		case "bucket":
			config.Database.Influx.Bucket = value
		case "password":
			config.Database.Influx.Password = value
		}
	}

	return nil
}

// setFeaturesValue 设置功能配置值
func (em *EnvMapper) setFeaturesValue(config *Config, parts []string, value string) error {
	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "http3":
		switch parts[1] {
		case "enabled":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				config.Features.HTTP3.Enabled = boolVal
			}
		case "max_connections":
			if intVal, err := strconv.Atoi(value); err == nil {
				config.Features.HTTP3.MaxConnections = intVal
			}
		case "idle_timeout":
			if duration, err := time.ParseDuration(value); err == nil {
				config.Features.HTTP3.IdleTimeout = int(duration)
			}
		case "keep_alive":
			if duration, err := time.ParseDuration(value); err == nil {
				config.Features.HTTP3.KeepAlive = int(duration)
			}
		}
	case "websocket":
		switch parts[1] {
		case "enabled":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				config.Features.WebSocket.Enabled = boolVal
			}
		case "ping_interval":
			if duration, err := time.ParseDuration(value); err == nil {
				config.Features.WebSocket.PingInterval = int(duration)
			}
		case "pong_timeout":
			if duration, err := time.ParseDuration(value); err == nil {
				config.Features.WebSocket.PongTimeout = int(duration)
			}
		case "max_message_size":
			if intVal, err := strconv.Atoi(value); err == nil {
				config.Features.WebSocket.MaxMessageSize = int64(intVal)
			}
		case "buffer_size":
			if intVal, err := strconv.Atoi(value); err == nil {
				config.Features.WebSocket.BufferSize = intVal
			}
		}
	case "gzip":
		switch parts[1] {
		case "enabled":
			if boolVal, err := strconv.ParseBool(value); err == nil {
				config.Features.Gzip.Enabled = boolVal
			}
		}
	case "cache":
		switch parts[1] {
		case "size":
			if intVal, err := strconv.Atoi(value); err == nil {
				config.Features.Cache.Size = intVal
			}
		}
	}

	return nil
}

// setSecurityValue 设置安全配置值
func (em *EnvMapper) setSecurityValue(config *Config, parts []string, value string) error {
	if len(parts) < 1 {
		return nil
	}

	switch parts[0] {
	case "strict_mode":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			config.Security.StrictMode = boolVal
		}
	case "rate_limit":
		if intVal, err := strconv.Atoi(value); err == nil {
			config.Security.RateLimit = intVal
		}
	}

	return nil
}

// setMonitorValue 设置监控配置值
func (em *EnvMapper) setMonitorValue(config *Config, parts []string, value string) error {
	if len(parts) < 1 {
		return nil
	}

	switch parts[0] {
	case "enabled":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			config.Monitor.Enabled = boolVal
		}
	case "port":
		if intVal, err := strconv.Atoi(value); err == nil {
			config.Monitor.Port = intVal
		}
	case "path":
		config.Monitor.Path = value
	}

	return nil
}

// setMiddlewareValue 设置中间件配置值
func (em *EnvMapper) setMiddlewareValue(config *Config, parts []string, value string) error {
	if len(parts) < 1 {
		return nil
	}

	// 查找对应的中间件
	for i := range config.Middleware {
		if config.Middleware[i].Name == parts[0] {
			if len(parts) >= 3 && parts[1] == "config" {
				// 设置中间件配置项
				if config.Middleware[i].Config == nil {
					config.Middleware[i].Config = make(map[string]interface{})
				}
				// 尝试解析为不同类型
				if intVal, err := strconv.Atoi(value); err == nil {
					config.Middleware[i].Config[parts[2]] = intVal
				} else if boolVal, err := strconv.ParseBool(value); err == nil {
					config.Middleware[i].Config[parts[2]] = boolVal
				} else {
					config.Middleware[i].Config[parts[2]] = value
				}
			}
			break
		}
	}

	return nil
}

// AddMapping 添加自定义环境变量映射
func (em *EnvMapper) AddMapping(envKey, configPath string) {
	em.mappings[envKey] = configPath
}

// RemoveMapping 移除环境变量映射
func (em *EnvMapper) RemoveMapping(envKey string) {
	delete(em.mappings, envKey)
}

// GetMappings 获取所有映射关系
func (em *EnvMapper) GetMappings() map[string]string {
	result := make(map[string]string)
	for k, v := range em.mappings {
		result[k] = v
	}
	return result
}
