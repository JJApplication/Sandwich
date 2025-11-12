package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sandwich/config"
	"sandwich/log"
	"sandwich/modifier"
	"sandwich/pprof"
	"sandwich/protocols/http3"
	"sandwich/protocols/websocket"
	"sandwich/proxy"
	"sandwich/server"
	"sandwich/stat"
	"sandwich/stat/db"
	seq "sandwich/stat/sequence"
	"sandwich/utils"
	"strings"
)

// Application 应用程序结构
// 管理整个代理服务的生命周期
type Application struct {
	config        *config.Config       // 配置
	configLoader  *config.ConfigLoader // 配置加载器
	serverManager *server.Manager      // 服务器管理器
	router        *server.Router       // 路由器
	http3Server   *http3.Server        // HTTP/3 服务器
	wsServer      *websocket.Server    // WebSocket 服务器
	statServer    *stat.StatServer     // 状态统计服务器
	logger        *log.Log             // 日志记录器
	ctx           context.Context      // 应用上下文
	cancel        context.CancelFunc   // 取消函数
}

// NewApplication 创建新的应用程序实例
func NewApplication(logger *log.Log) *Application {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志记录器
	return &Application{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize 初始化应用程序
func (app *Application) Initialize(configPath string) error {
	app.logger.Printf("正在初始化 Sandwich 代理服务...")

	// 加载配置
	if err := app.loadConfig(configPath); err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 初始化组件
	if err := app.initializeComponents(); err != nil {
		return fmt.Errorf("初始化组件失败: %v", err)
	}

	// 启动配置监控
	app.startConfigWatcher()

	app.logger.Printf("Sandwich 代理服务初始化完成")
	return nil
}

// loadConfig 加载配置
func (app *Application) loadConfig(configPath string) error {
	// 创建配置加载器
	app.configLoader = config.NewConfigLoader(configPath)

	// 如果没有指定配置文件，使用默认配置
	if configPath == "" {
		app.logger.Printf("未指定配置文件，使用默认配置")
		app.config = config.GetDefaultConfig()
	} else {
		// 加载配置文件
		var err error
		app.config, err = app.configLoader.LoadConfig()
		if err != nil {
			return fmt.Errorf("加载配置文件失败: %v", err)
		}
		app.logger.Printf("已加载配置文件: %s", configPath)
	}

	// 应用环境变量覆盖
	envMapper := config.NewEnvMapper()
	if err := envMapper.ApplyEnvOverrides(app.config); err != nil {
		app.logger.Printf("应用环境变量覆盖失败: %v", err)
	}

	// 标准化配置
	config.NormalizeConfig(app.config)

	// 验证配置
	validator := config.NewConfigValidator(false)
	if validateResult := validator.ValidateConfig(app.config); validateResult != nil {
		if !validateResult.Valid {
			return fmt.Errorf("配置验证失败: %v", validateResult)
		}
		if len(validateResult.Warnings) > 0 {
			app.logger.Printf("配置存在警告: %v", validateResult.Warnings)
		}
	} else {
		app.logger.Printf("配置校验通过")
	}

	// 检查配置一致性
	warnings := config.ValidateConfigConsistency(app.config)
	for _, warning := range warnings {
		app.logger.Printf("配置警告: %s", warning)
	}

	// 注册全局配置
	app.configLoader.RegisterGlobalConfig(app.config)
	// 刷新日志配置
	log.Reload(app.config.Log)
	return nil
}

// initializeComponents 初始化组件
func (app *Application) initializeComponents() error {
	// 初始化数据库
	db.Init(app.config)
	// 初始化Sequence时序
	seq.InitSequenceManager(app.config, db.GetDB())
	// 初始化特性组件
	modifier.InitModifiers()
	// 创建原始代理处理器（复用现有的 proxy.go 逻辑）
	proxyHandler := app.createProxyHandler()

	// 创建路由器
	app.router = server.NewRouter(app.config, proxyHandler)

	// 创建服务器管理器
	app.serverManager = server.NewManager(app.config, app.router, app.logger)

	// 初始化 HTTP/3 服务器
	if app.config.Features.HTTP3.Enabled {
		app.http3Server = http3.NewServer(app.config.Features.HTTP3, app.router, app.logger)
	}

	// 初始化 WebSocket 服务器
	if app.config.Features.WebSocket.Enabled {
		app.wsServer = websocket.NewServer(app.config.Features.WebSocket, app.logger)

		// 启动 WebSocket 服务器
		if err := app.wsServer.Start(); err != nil {
			return fmt.Errorf("启动 WebSocket 服务器失败: %v", err)
		}
	}

	// 初始化状态服务器
	statServer := stat.NewStatServer(app.config.Stat, app.logger)
	app.statServer = statServer
	if err := statServer.Start(); err != nil {
		return fmt.Errorf("启动 状态统计服务器失败: %v", err)
	}

	// pprof
	pprof.InitPProf(app.config.PProf)

	return nil
}

// createProxyHandler 创建代理处理器
// 集成现有的代理逻辑，并添加 WebSocket 支持
func (app *Application) createProxyHandler() http.Handler {
	// 创建基于现有逻辑的反向代理
	rp := app.createReverseProxy()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否为 WebSocket 升级请求
		if app.wsServer != nil && app.isWebSocketRequest(r) {
			// 处理 WebSocket 升级
			if err := app.wsServer.HandleUpgrade(w, r); err != nil {
				app.logger.Printf("WebSocket 升级失败: %v", err)
				http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
			}
			return
		}

		// 对于普通 HTTP 请求，使用反向代理处理
		rp.ServeHTTP(w, r)
	})
}

// isWebSocketRequest 检查是否为 WebSocket 请求
func (app *Application) isWebSocketRequest(r *http.Request) bool {
	// 检查Connection和Upgrade头是否存在且值正确
	connection := strings.ToLower(r.Header.Get("Connection"))
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))

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

// createReverseProxy 创建反向代理
// 基于现有的 proxy.go 逻辑，集成到新架构中
func (app *Application) createReverseProxy() *httputil.ReverseProxy {
	return proxy.CreateProxy()
}

// startConfigWatcher 启动配置文件监控
func (app *Application) startConfigWatcher() {
	if app.configLoader == nil {
		return
	}

	// 添加配置变化监听器
	app.configLoader.AddWatcher(func(newConfig *config.Config) {
		app.logger.Printf("检测到配置变化，正在重新加载...")

		// 更新配置
		app.config = newConfig

		// 更新路由器配置
		if app.router != nil {
			app.router.UpdateConfig(newConfig)
		}

		// 更新服务器管理器配置
		if app.serverManager != nil {
			if err := app.serverManager.UpdateConfig(newConfig); err != nil {
				app.logger.Printf("更新服务器配置失败: %v", err)
			}
		}

		app.logger.Printf("配置重新加载完成")
	})

	// 启动配置监控
	app.configLoader.StartWatching(utils.ToSecond(app.config.Monitor.Interval))
}

// Start 启动应用程序
func (app *Application) Start() error {
	app.logger.Printf("正在启动 Sandwich 代理服务...")

	// 启动服务器管理器
	if err := app.serverManager.Start(); err != nil {
		return fmt.Errorf("启动服务器管理器失败: %v", err)
	}

	// 启动 HTTP/3 服务器（如果启用）
	if app.http3Server != nil {
		// 查找 HTTPS 服务器配置来获取 TLS 配置
		httpsServers := config.GetServersByProtocol(app.config, "https")
		if len(httpsServers) > 0 {
			server := httpsServers[0]
			addr := fmt.Sprintf("%s:%d", server.Host, server.Port)

			// 创建 TLS 配置
			tlsConfig, err := app.createTLSConfig(server.TLS)
			if err != nil {
				app.logger.Printf("创建 TLS 配置失败: %v", err)
			} else {
				if err := app.http3Server.Start(addr, tlsConfig); err != nil {
					app.logger.Printf("启动 HTTP/3 服务器失败: %v", err)
				}
			}
		}
	}

	// 打印启动信息
	app.printStartupInfo()

	return nil
}

// createTLSConfig 创建 TLS 配置
func (app *Application) createTLSConfig(tlsConfig *config.TLSConfig) (*tls.Config, error) {
	if tlsConfig == nil {
		return nil, fmt.Errorf("TLS 配置为空")
	}

	// 这里应该实现 TLS 配置的创建逻辑
	// 为了简化，返回一个基本的配置
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}, nil
}

// printStartupInfo 打印启动信息
func (app *Application) printStartupInfo() {
	app.logger.Printf("========================================")
	app.logger.Printf("Sandwich 代理服务 v2 启动成功")
	app.logger.Printf("========================================")

	// 打印监听地址
	addresses := app.serverManager.GetListenAddresses()
	for _, addr := range addresses {
		app.logger.Printf("监听地址: %s", addr)
	}

	// 打印功能状态
	if app.config.Features.HTTP3.Enabled {
		app.logger.Printf("HTTP/3 支持: 已启用")
	}
	if app.config.Features.WebSocket.Enabled {
		app.logger.Printf("WebSocket 支持: 已启用")
	}
	if app.config.Features.Gzip.Enabled {
		app.logger.Printf("Gzip 压缩: 已启用")
	}
	if app.config.Features.Cache.Enabled {
		app.logger.Printf("内存缓存: 已启用")
	}

	// 打印域名配置
	//domains := app.router.GetSupportedDomains()
	//if len(domains) > 0 {
	//	app.logger.Printf("支持的域名: %v", domains)
	//}

	app.logger.Printf("========================================")
	printModifiers(&app.config.Features)
}

// Stop 停止应用程序
func (app *Application) Stop() error {
	app.logger.Printf("正在停止 Sandwich 代理服务...")

	// 停止配置监控
	if app.configLoader != nil {
		app.configLoader.StopWatching()
	}

	// 停止 HTTP/3 服务器
	if app.http3Server != nil {
		if err := app.http3Server.Stop(); err != nil {
			app.logger.Printf("停止 HTTP/3 服务器失败: %v", err)
		}
	}

	// 停止 WebSocket 服务器
	if app.wsServer != nil {
		if err := app.wsServer.Stop(); err != nil {
			app.logger.Printf("停止 WebSocket 服务器失败: %v", err)
		}
	}

	// 停止服务器管理器
	if app.serverManager != nil {
		if err := app.serverManager.Stop(); err != nil {
			app.logger.Printf("停止服务器管理器失败: %v", err)
		}
	}

	// 停止状态统计服务器
	if app.statServer != nil {
		if err := app.statServer.Stop(); err != nil {
			app.logger.Printf("停止状态服务器失败: %v", err)
		}
	}

	// 取消上下文
	app.cancel()

	app.logger.Printf("Sandwich 代理服务已停止")
	return nil
}

func (app *Application) getAllDomains() []string {
	return config.GetDomains(app.config)
}

// validateDomain 验证域名
// 集成现有的域名验证逻辑
func (app *Application) validateDomain(request *http.Request) bool {
	host := request.Host
	if host == "" {
		return false
	}

	// 检查是否在配置的域名列表中
	for _, domain := range app.getAllDomains() {
		if domain == host || domain == "*" {
			return true
		}
	}

	// 如果没有配置域名，允许所有请求（向后兼容）
	if len(app.getAllDomains()) == 0 {
		return true
	}

	return false
}

// parseRequest 解析请求
// 集成现有的请求解析逻辑
func (app *Application) parseRequest(request *http.Request) *url.URL {
	return nil
}

// addResponseHeaders 添加响应头
func (app *Application) addResponseHeaders(response *http.Response) {
	// 添加缓存控制头
	response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	response.Header.Set("Pragma", "no-cache")
	response.Header.Set("Expires", "0")

	// 添加服务器标识
	response.Header.Set("X-Powered-By", "Sandwich-v2")
	response.Header.Set("Server", "Sandwich/2.0")
}

// handleProxyError 处理代理错误
func (app *Application) handleProxyError(writer http.ResponseWriter, request *http.Request, err error) {
	app.logger.Printf("代理错误 - Host: %s, URL: %s, Error: %v", request.Host, request.URL.String(), err)

	// 检查内部错误标志
	errorType := request.Header.Get("X-Sandwich-Error")
	switch errorType {
	case "domain-not-allowed":
		app.logger.Printf("域名不被允许: %s", request.Host)
		writer.WriteHeader(http.StatusForbidden)
		writer.Write([]byte("Domain not allowed"))
		return
	case "no-backend":
		app.logger.Printf("没有可用的后端服务: %s", request.Host)
		writer.WriteHeader(http.StatusBadGateway)
		writer.Write([]byte("No backend available"))
		return
	}

	// 通用错误处理
	writer.WriteHeader(http.StatusBadGateway)
	writer.Write([]byte("Proxy error occurred"))
}

// GetStats 获取服务统计信息
func (app *Application) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 服务器状态
	if app.serverManager != nil {
		stats["servers"] = app.serverManager.GetServerStatus()
		stats["running_servers"] = app.serverManager.GetRunningServers()
		stats["failed_servers"] = app.serverManager.GetFailedServers()
	}

	// HTTP/3 统计
	if app.http3Server != nil {
		stats["http3"] = app.http3Server.GetStats()
	}

	// WebSocket 统计
	if app.wsServer != nil {
		stats["websocket"] = app.wsServer.GetStats()
	}

	// 域名统计
	if app.router != nil {
		//stats["domains"] = app.router.GetDomainStats()
	}

	return stats
}
