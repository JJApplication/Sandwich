/*
Project: Sandwich config.go
Created: 2025/1/15 by Landers
Copyright Renj
*/

package config

import (
	"encoding/json"
	"os"
	"sandwich/constant"
	"strings"
)

// Config 主配置结构体，包含所有服务配置信息
type Config struct {
	Servers      []ServerConfig     `yaml:"servers" json:"servers"`             // 服务器配置列表
	Middleware   []MiddlewareConfig `yaml:"middleware" json:"middleware"`       // 中间件配置列表
	Features     FeatureConfig      `yaml:"features" json:"features"`           // 功能特性配置
	Database     DatabaseConfig     `yaml:"database" json:"database"`           // 数据库配置
	Monitor      MonitorConfig      `yaml:"monitor" json:"monitor"`             // 监控配置
	Security     SecurityConfig     `yaml:"security" json:"security"`           // 安全配置
	FrontProxy   FrontProxyConfig   `yaml:"front_proxy" json:"front_proxy"`     // 前端代理配置
	ProxyHeader  ProxyHeader        `yaml:"proxy_header" json:"proxy_header"`   // 内置的代理头配置
	CustomHeader map[string]string  `yaml:"custom_header" json:"custom_header"` // 自定义Header
	DomainMap    string             `yaml:"domain_map" json:"domain_map"`       // 域名映射文件
	JobSyncTime  int                `yaml:"job_sync_time" json:"job_sync_time"` // 同步时间
	Debug        bool               `yaml:"debug" json:"debug"`                 // 调试模式
}

// ServerConfig 服务器配置结构体
type ServerConfig struct {
	Name         string         `yaml:"name" json:"name"`                   // 服务器名称
	Host         string         `yaml:"host" json:"host"`                   // 监听主机地址
	Port         int            `yaml:"port" json:"port"`                   // 监听端口
	UseHttp2     bool           `yaml:"use_http2" json:"use_http2"`         // 使用HTTP2
	Protocol     string         `yaml:"protocol" json:"protocol"`           // 协议类型: http, https, http3
	Enabled      bool           `yaml:"enabled" json:"enabled"`             // 是否启用
	TLS          *TLSConfig     `yaml:"tls,omitempty" json:"tls,omitempty"` // TLS配置
	DomainConfig []DomainConfig `yaml:"domains" json:"domains"`             // 域名绑定配置
}

// TLSConfig TLS证书配置结构体
type TLSConfig struct {
	CertFile string `yaml:"cert_file" json:"cert_file"` // 证书文件路径
	KeyFile  string `yaml:"key_file" json:"key_file"`   // 私钥文件路径
	AutoTLS  bool   `yaml:"auto_tls" json:"auto_tls"`   // 是否启用自动TLS
}

// DomainConfig 域名配置结构体
type DomainConfig struct {
	Domains []string `yaml:"domains" json:"domains"`   // 域名
	UseTLS  bool     `yaml:"use_tls" json:"use_tls"`   // 监听在https
	AutoTLS bool     `yaml:"auto_tls" json:"auto_tls"` // 自动重定向
}

// MiddlewareConfig 中间件配置结构体
type MiddlewareConfig struct {
	Name    string                 `yaml:"name" json:"name"`       // 中间件名称
	Enabled bool                   `yaml:"enabled" json:"enabled"` // 是否启用
	Order   int                    `yaml:"order" json:"order"`     // 执行顺序
	Config  map[string]interface{} `yaml:"config" json:"config"`   // 中间件配置参数
}

func (m *MiddlewareConfig) GetInt(key string) int {
	v, ok := m.Config[key]
	if !ok {
		return 0
	}
	value, ok := v.(float64)
	if !ok {
		return 0
	}
	return int(value)
}

func (m *MiddlewareConfig) GetString(key string) string {
	v, ok := m.Config[key]
	if !ok {
		return ""
	}
	value, ok := v.(string)
	if !ok {
		return ""
	}
	return value
}

func (m *MiddlewareConfig) GetBool(key string) bool {
	v, ok := m.Config[key]
	if !ok {
		return false
	}
	value, ok := v.(bool)
	if ok {
		return value
	}
	switch v.(type) {
	case float64:
		return v.(float64) > 0
	case bool:
		return v.(bool)
	case string:
		vs := v.(string)
		if strings.ToLower(vs) == "true" || strings.ToLower(vs) == "yes" {
			return true
		}
		return false
	default:
		return false
	}
}

// FeatureConfig 功能特性配置结构体
type FeatureConfig struct {
	HTTP3     HTTP3Config     `yaml:"http3" json:"http3"`         // HTTP/3配置
	WebSocket WebSocketConfig `yaml:"websocket" json:"websocket"` // WebSocket配置
	Gzip      GzipConfig      `yaml:"gzip" json:"gzip"`           // Gzip压缩配置
	Cache     CacheConfig     `yaml:"cache" json:"cache"`         // 缓存配置
}

// HTTP3Config HTTP/3协议配置结构体
type HTTP3Config struct {
	Enabled        bool `yaml:"enabled" json:"enabled"`                 // 是否启用HTTP/3
	MaxConnections int  `yaml:"max_connections" json:"max_connections"` // 最大连接数
	IdleTimeout    int  `yaml:"idle_timeout" json:"idle_timeout"`       // 空闲超时时间
	KeepAlive      int  `yaml:"keep_alive" json:"keep_alive"`           // 保活时间
}

// WebSocketConfig WebSocket协议配置结构体
type WebSocketConfig struct {
	Enabled        bool  `yaml:"enabled" json:"enabled"`                   // 是否启用WebSocket
	PingInterval   int   `yaml:"ping_interval" json:"ping_interval"`       // 心跳间隔
	PongTimeout    int   `yaml:"pong_timeout" json:"pong_timeout"`         // 心跳响应超时
	MaxMessageSize int64 `yaml:"max_message_size" json:"max_message_size"` // 最大消息大小
	BufferSize     int   `yaml:"buffer_size" json:"buffer_size"`           // 缓冲区大小
}

// GzipConfig Gzip压缩配置结构体
type GzipConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"` // 是否启用Gzip压缩
	Level   int      `yaml:"level" json:"level"`     // 压缩级别 1-9
	Types   []string `yaml:"types" json:"types"`     // 压缩的MIME类型列表
}

// CacheConfig 缓存配置结构体
type CacheConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用缓存
	Size     int    `yaml:"size" json:"size"`         // 缓存大小
	TTL      int    `yaml:"ttl" json:"ttl"`           // 缓存过期时间
	Strategy string `yaml:"strategy" json:"strategy"` // 缓存策略: lru, lfu, fifo
}

// DatabaseConfig 数据库配置结构体
type DatabaseConfig struct {
	Mongo  MongoConfig  `yaml:"mongo" json:"mongo"`   // MongoDB配置
	Influx InfluxConfig `yaml:"influx" json:"influx"` // InfluxDB配置
}

// MongoConfig MongoDB配置结构体
type MongoConfig struct {
	URL      string `yaml:"url" json:"url"`           // MongoDB连接URL
	Database string `yaml:"database" json:"database"` // 数据库名称
	Timeout  int    `yaml:"timeout" json:"timeout"`   // 连接超时时间
}

// InfluxConfig InfluxDB配置结构体
type InfluxConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`   // 是否启用InfluxDB
	URL      string `yaml:"url" json:"url"`           // InfluxDB连接URL
	Token    string `yaml:"token" json:"token"`       // 访问令牌
	Org      string `yaml:"org" json:"org"`           // 组织名称
	Bucket   string `yaml:"bucket" json:"bucket"`     // 存储桶名称
	Password string `yaml:"password" json:"password"` // 密码
}

// MonitorConfig 监控配置结构体
type MonitorConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`       // 是否启用监控
	Port       int    `yaml:"port" json:"port"`             // 监控端口
	Path       string `yaml:"path" json:"path"`             // 监控路径
	Interval   int    `yaml:"interval" json:"interval"`     // 监控间隔
	Prometheus bool   `yaml:"prometheus" json:"prometheus"` // 是否启用Prometheus
}

// SecurityConfig 安全配置结构体
type SecurityConfig struct {
	StrictMode bool     `yaml:"strict_mode" json:"strict_mode"` // 严格模式
	AllowIPs   []string `yaml:"allow_ips" json:"allow_ips"`     // 允许的IP列表
	DenyIPs    []string `yaml:"deny_ips" json:"deny_ips"`       // 拒绝的IP列表
	RateLimit  int      `yaml:"rate_limit" json:"rate_limit"`   // 速率限制
}

type FrontProxyConfig struct {
	GrpcAddr     string `yaml:"grpc_addr" json:"grpc_addr"`
	FrontendFlag string `yaml:"frontend_flag" json:"frontend_flag"`
	FrontendHost string `yaml:"frontend_host" json:"frontend_host"`
	FrontendPort int    `yaml:"frontend_port" json:"frontend_port"`
}

type ProxyHeader struct {
	TraceId            string `yaml:"trace_id" json:"trace_id"`                         // traceId头
	FrontendHostHeader string `yaml:"frontend_host_header" json:"frontend_host_header"` // 前端服务真实HOST
	BackendHeader      string `yaml:"backend_header" json:"backend_header"`             // 区分后端服务标识
	ProxyApp           string `yaml:"proxy_app" json:"proxy_app"`                       // 要转到的后端服务标识
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		Servers: []ServerConfig{
			{
				Name:     "http-server",
				Host:     "0.0.0.0",
				Port:     80,
				Protocol: "http",
				Enabled:  true,
			},
			{
				Name:     "https-server",
				Host:     "0.0.0.0",
				Port:     443,
				Protocol: "https",
				Enabled:  false,
				TLS: &TLSConfig{
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
					AutoTLS:  false,
				},
			},
		},
		Middleware: []MiddlewareConfig{
			{
				Name:    "limiter",
				Enabled: true,
				Order:   1,
				Config: map[string]interface{}{
					"limit": 100,
					"reset": 60,
				},
			},
			{
				Name:    "breaker",
				Enabled: true,
				Order:   2,
				Config: map[string]interface{}{
					"limit": 20,
					"reset": 10,
				},
			},
		},
		Features: FeatureConfig{
			HTTP3: HTTP3Config{
				Enabled:        false,
				MaxConnections: 1000,
				IdleTimeout:    60,
				KeepAlive:      30,
			},
			WebSocket: WebSocketConfig{
				Enabled:        false,
				PingInterval:   10,
				PongTimeout:    10,
				MaxMessageSize: 1048576, // 1MB
				BufferSize:     1024,
			},
			Gzip: GzipConfig{
				Enabled: true,
				Level:   6,
				Types:   []string{"text/html", "text/css", "text/javascript", "application/json"},
			},
			Cache: CacheConfig{
				Enabled:  true,
				Size:     1000,
				TTL:      60,
				Strategy: "lru",
			},
		},
		Database: DatabaseConfig{
			Mongo: MongoConfig{
				URL:      "mongodb://localhost:27017",
				Database: "sandwich",
				Timeout:  10,
			},
			Influx: InfluxConfig{
				Enabled:  false,
				URL:      "http://localhost:8086",
				Token:    "",
				Org:      "sandwich",
				Bucket:   "metrics",
				Password: "",
			},
		},
		Monitor: MonitorConfig{
			Enabled:    true,
			Port:       9090,
			Path:       "/metrics",
			Interval:   15,
			Prometheus: true,
		},
		Security: SecurityConfig{
			StrictMode: false,
			AllowIPs:   []string{},
			DenyIPs:    []string{},
			RateLimit:  1000,
		},
		ProxyHeader: ProxyHeader{
			TraceId:            "X-Gateway-Trace-Id",
			FrontendHostHeader: "X-Proxy-Internal-Host",
			BackendHeader:      "X-Proxy-Internal-Local",
			ProxyApp:           "X-Proxy-Backend",
		},
		CustomHeader: map[string]string{
			"Proxy-Server":    constant.Sandwich,
			"Proxy-Copyright": constant.Copyright,
		},
	}
}

// CreateConfig 生成默认配置文件
func CreateConfig() error {
	config := GetDefaultConfig()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("config.default.json", data, 0644)
}
