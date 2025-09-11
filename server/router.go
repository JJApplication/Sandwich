package server

import (
	"net/http"
	"strings"
	"sync"

	"sandwich/config"
)

// Router 服务器路由器
// 负责根据域名和协议将请求路由到正确的后端服务
type Router struct {
	config        *config.Config
	defaultDomain *config.DomainConfig // 默认域名配置
	mu            sync.RWMutex         // 读写锁
	proxy         http.Handler         // 代理处理器
}

// NewRouter 创建新的路由器
func NewRouter(cfg *config.Config, proxyHandler http.Handler) *Router {
	router := &Router{
		config: cfg,
		proxy:  proxyHandler,
	}

	// 构建域名映射
	router.buildDomainMap()

	return router
}

// buildDomainMap 构建域名映射表
func (r *Router) buildDomainMap() {
}

// isDefaultDomain 判断是否为默认域名
func isDefaultDomain(domain string) bool {
	// 简单的默认域名判断逻辑
	// 可以根据实际需求调整
	return !strings.Contains(domain, ".") ||
		strings.HasPrefix(domain, "localhost") ||
		strings.HasPrefix(domain, "127.0.0.1") ||
		strings.HasPrefix(domain, "0.0.0.0")
}

// ServeHTTP 实现 http.Handler 接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 检查是否为WebSocket升级请求
	if r.isWebSocketRequest(req) {
		// 对于WebSocket请求，需要特殊处理
		// 这里直接转发到代理处理器，让它来处理WebSocket升级
		r.proxy.ServeHTTP(w, req)
		return
	}

	// 对于普通HTTP请求，转发到代理处理器
	r.proxy.ServeHTTP(w, req)
}

// extractHost 从请求中提取主机名
func (r *Router) extractHost(req *http.Request) string {
	host := req.Host

	// 移除端口号
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// 标准化主机名
	return strings.ToLower(strings.TrimSpace(host))
}

// matchWildcard 通配符匹配
func (r *Router) matchWildcard(pattern, host string) bool {
	// 简单的通配符匹配实现
	// 支持 *.example.com 格式
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}

	suffix := pattern[2:] // 移除 "*."
	return strings.HasSuffix(host, "."+suffix) || host == suffix
}

// detectProtocol 检测请求协议
func (r *Router) detectProtocol(req *http.Request) string {
	// 检查是否为 WebSocket 升级请求
	if r.isWebSocketRequest(req) {
		return "websocket"
	}

	// 检查 TLS
	if req.TLS != nil {
		return "https"
	}

	// 检查 HTTP/3 (通过特定头部)
	if req.Header.Get("Alt-Svc") != "" || req.ProtoMajor == 3 {
		return "http3"
	}

	return "http"
}

// isWebSocketRequest 检查是否为 WebSocket 请求
func (r *Router) isWebSocketRequest(req *http.Request) bool {
	// 检查Connection和Upgrade头是否存在且值正确
	connection := strings.ToLower(req.Header.Get("Connection"))
	upgrade := strings.ToLower(req.Header.Get("Upgrade"))
	
	// 检查Connection头是否包含"upgrade"（可能有多个值）
	hasUpgradeConnection := false
	for _, part := range strings.Split(connection, ",") {
		if strings.TrimSpace(part) == "upgrade" {
			hasUpgradeConnection = true
			break
		}
	}
	
	return hasUpgradeConnection && upgrade == "websocket"
}

// setDomainContext 在请求上下文中设置域名配置
func (r *Router) setDomainContext(req *http.Request, domainConfig *config.DomainConfig) {
	// 通过请求头传递域名配置信息
	// 这样代理处理器就可以获取到相关配置
	//req.Header.Set("X-Sandwich-Domain", domainConfig.Domain)
	//req.Header.Set("X-Sandwich-LoadBalance", domainConfig.LoadBalance)
	//req.Header.Set("X-Sandwich-Sticky", fmt.Sprintf("%t", domainConfig.Sticky))
	//req.Header.Set("X-Sandwich-HealthCheck", fmt.Sprintf("%t", domainConfig.HealthCheck))
	//req.Header.Set("X-Sandwich-Timeout", domainConfig.Timeout.String())
	//req.Header.Set("X-Sandwich-Retries", fmt.Sprintf("%d", domainConfig.Retries))
}

// UpdateConfig 更新配置
func (r *Router) UpdateConfig(newConfig *config.Config) {
	r.config = newConfig
	r.buildDomainMap()
}

// GetDomainConfig 根据域名获取配置

// GetSupportedDomains 获取支持的域名列表

// GetDomainStats 获取域名统计信息

// DomainStats 域名统计信息
type DomainStats struct {
	Domain       string   `json:"domain"`        // 域名
	BackendCount int      `json:"backend_count"` // 后端服务数量
	LoadBalance  string   `json:"load_balance"`  // 负载均衡算法
	HealthCheck  bool     `json:"health_check"`  // 是否启用健康检查
	Sticky       bool     `json:"sticky"`        // 是否启用会话保持
	Protocols    []string `json:"protocols"`     // 支持的协议
}

// MiddlewareRouter 中间件路由器
// 负责应用中间件链
type MiddlewareRouter struct {
	router      *Router
	middlewares []MiddlewareHandler
}

// MiddlewareHandler 中间件处理器接口
type MiddlewareHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

// NewMiddlewareRouter 创建中间件路由器
func NewMiddlewareRouter(router *Router, middlewares []MiddlewareHandler) *MiddlewareRouter {
	return &MiddlewareRouter{
		router:      router,
		middlewares: middlewares,
	}
}

// ServeHTTP 实现 http.Handler 接口
func (mr *MiddlewareRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 构建中间件链
	handler := mr.buildChain()
	handler.ServeHTTP(w, r)
}

// buildChain 构建中间件链
func (mr *MiddlewareRouter) buildChain() http.Handler {
	// 最终处理器是路由器
	final := http.HandlerFunc(mr.router.ServeHTTP)

	// 从后往前构建中间件链
	for i := len(mr.middlewares) - 1; i >= 0; i-- {
		middleware := mr.middlewares[i]
		next := final
		final = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middleware.ServeHTTP(w, r, next.ServeHTTP)
		})
	}

	return final
}

// AddMiddleware 添加中间件
func (mr *MiddlewareRouter) AddMiddleware(middleware MiddlewareHandler) {
	mr.middlewares = append(mr.middlewares, middleware)
}

// RemoveMiddleware 移除中间件
func (mr *MiddlewareRouter) RemoveMiddleware(index int) {
	if index >= 0 && index < len(mr.middlewares) {
		mr.middlewares = append(mr.middlewares[:index], mr.middlewares[index+1:]...)
	}
}

// GetRouter 获取底层路由器
func (mr *MiddlewareRouter) GetRouter() *Router {
	return mr.router
}
