package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"sandwich/config"
	"sandwich/log"
)

// Manager 服务器管理器
// 负责管理多个服务器实例的启动、停止和监控
type Manager struct {
	config  *config.Config
	servers map[string]*ServerInstance // 服务器实例映射
	handler http.Handler               // 请求处理器
	mu      sync.RWMutex               // 读写锁
	ctx     context.Context            // 上下文
	cancel  context.CancelFunc         // 取消函数
	wg      sync.WaitGroup             // 等待组
	started bool                       // 是否已启动
	logger  *log.Log                   // 日志记录器
}

// ServerInstance 服务器实例
// 表示单个监听端口的服务器
type ServerInstance struct {
	Name     string              // 服务器名称
	Config   config.ServerConfig // 服务器配置
	Server   *http.Server        // HTTP 服务器
	Listener net.Listener        // 网络监听器
	TLS      bool                // 是否启用 TLS
	Started  bool                // 是否已启动
	Error    error               // 启动错误
}

// NewManager 创建新的服务器管理器
func NewManager(cfg *config.Config, handler http.Handler, logger *log.Log) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	if logger == nil {
		logger = log.GetLogger()
	}

	return &Manager{
		config:  cfg,
		servers: make(map[string]*ServerInstance),
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

// Start 启动所有配置的服务器
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("服务器管理器已经启动")
	}

	// 获取启用的服务器配置
	enabledServers := config.GetEnabledServers(m.config)
	if len(enabledServers) == 0 {
		return fmt.Errorf("没有启用的服务器配置")
	}

	m.logger.Printf("开始启动 %d 个服务器实例", len(enabledServers))

	// 启动每个服务器
	for _, serverConfig := range enabledServers {
		if err := m.startServer(serverConfig); err != nil {
			m.logger.Printf("启动服务器 %s 失败: %v", serverConfig.Name, err)
			// 继续启动其他服务器，不因为一个失败而全部停止
			continue
		}
	}

	// 检查是否至少有一个服务器启动成功
	if len(m.servers) == 0 {
		return fmt.Errorf("没有服务器启动成功")
	}

	m.started = true
	m.logger.Printf("服务器管理器启动完成，成功启动 %d 个服务器", len(m.servers))

	return nil
}

// startServer 启动单个服务器
func (m *Manager) startServer(serverConfig config.ServerConfig) error {
	// 创建服务器实例
	instance := &ServerInstance{
		Name:   serverConfig.Name,
		Config: serverConfig,
		TLS:    serverConfig.Protocol == "https" || serverConfig.Protocol == "http3",
	}

	// 创建监听地址
	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)

	// 设置支持协议
	proto := &http.Protocols{}
	if serverConfig.UseHttp2 {
		proto.SetHTTP2(true)
		proto.SetHTTP1(true)
		proto.SetUnencryptedHTTP2(true)
	} else {
		proto.SetHTTP2(false)
		proto.SetUnencryptedHTTP2(false)
		proto.SetHTTP1(true)
	}

	// 创建 HTTP 服务器
	instance.Server = &http.Server{
		Addr:    addr,
		Handler: m.handler,
		// 设置合理的超时时间
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		Protocols:         proto,
	}

	// 创建监听器
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %v", err)
	}
	instance.Listener = listener

	// 如果是 HTTPS，配置 TLS
	if instance.TLS {
		if err := m.configureTLS(instance); err != nil {
			listener.Close()
			return fmt.Errorf("配置 TLS 失败: %v", err)
		}
	}

	// 保存服务器实例
	m.servers[serverConfig.Name] = instance

	// 在 goroutine 中启动服务器
	m.wg.Add(1)
	go m.runServer(instance)

	m.logger.Printf("服务器 %s 开始监听 %s (协议: %s)",
		serverConfig.Name, addr, serverConfig.Protocol)

	return nil
}

// configureTLS 配置 TLS
func (m *Manager) configureTLS(instance *ServerInstance) error {
	tlsConfig := instance.Config.TLS
	if tlsConfig == nil {
		return fmt.Errorf("HTTPS 服务器缺少 TLS 配置")
	}

	if tlsConfig.AutoTLS {
		// TODO: 实现自动 TLS (Let's Encrypt)
		return fmt.Errorf("自动 TLS 功能尚未实现")
	}

	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
	if err != nil {
		return fmt.Errorf("加载 TLS 证书失败: %v", err)
	}

	// 配置 TLS
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// 设置最低 TLS 版本
		MinVersion: tls.VersionTLS12,
		// 优先使用服务器的密码套件顺序
		PreferServerCipherSuites: true,
	}

	// 应用 TLS 配置
	instance.Server.TLSConfig = tlsCfg
	instance.Listener = tls.NewListener(instance.Listener, tlsCfg)

	return nil
}

// runServer 运行服务器
func (m *Manager) runServer(instance *ServerInstance) {
	defer m.wg.Done()

	// 标记服务器已启动
	instance.Started = true

	// 启动服务器
	err := instance.Server.Serve(instance.Listener)
	if err != nil && err != http.ErrServerClosed {
		instance.Error = err
		m.logger.Printf("服务器 %s 运行错误: %v", instance.Name, err)
	} else {
		m.logger.Printf("服务器 %s 已停止", instance.Name)
	}

	// 标记服务器已停止
	instance.Started = false
}

// Stop 停止所有服务器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("服务器管理器未启动")
	}

	m.logger.Printf("开始停止 %d 个服务器实例", len(m.servers))

	// 创建停止超时上下文
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	// 停止所有服务器
	var wg sync.WaitGroup
	for name, instance := range m.servers {
		wg.Add(1)
		go func(name string, instance *ServerInstance) {
			defer wg.Done()
			if err := instance.Server.Shutdown(stopCtx); err != nil {
				m.logger.Printf("停止服务器 %s 失败: %v", name, err)
				// 强制关闭
				instance.Server.Close()
			} else {
				m.logger.Printf("服务器 %s 已优雅停止", name)
			}
		}(name, instance)
	}

	// 等待所有服务器停止
	wg.Wait()

	// 取消上下文
	m.cancel()

	// 等待所有 goroutine 结束
	m.wg.Wait()

	// 清理状态
	m.servers = make(map[string]*ServerInstance)
	m.started = false

	m.logger.Printf("所有服务器已停止")
	return nil
}

// Restart 重启所有服务器
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return fmt.Errorf("停止服务器失败: %v", err)
	}

	// 重新创建上下文
	m.ctx, m.cancel = context.WithCancel(context.Background())

	return m.Start()
}

// GetServerStatus 获取服务器状态
func (m *Manager) GetServerStatus() map[string]*ServerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本以避免并发访问问题
	status := make(map[string]*ServerInstance)
	for name, instance := range m.servers {
		// 创建实例副本
		statusCopy := &ServerInstance{
			Name:    instance.Name,
			Config:  instance.Config,
			TLS:     instance.TLS,
			Started: instance.Started,
			Error:   instance.Error,
		}
		status[name] = statusCopy
	}

	return status
}

// GetServer 根据名称获取服务器实例
func (m *Manager) GetServer(name string) (*ServerInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.servers[name]
	return instance, exists
}

// IsStarted 检查管理器是否已启动
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// GetRunningServers 获取正在运行的服务器列表
func (m *Manager) GetRunningServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var running []string
	for name, instance := range m.servers {
		if instance.Started && instance.Error == nil {
			running = append(running, name)
		}
	}

	return running
}

// GetFailedServers 获取启动失败的服务器列表
func (m *Manager) GetFailedServers() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	failed := make(map[string]error)
	for name, instance := range m.servers {
		if instance.Error != nil {
			failed[name] = instance.Error
		}
	}

	return failed
}

// UpdateConfig 更新配置并重启受影响的服务器
func (m *Manager) UpdateConfig(newConfig *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 比较配置差异
	diff := config.CompareConfigs(m.config, newConfig)
	if len(diff) == 0 {
		m.logger.Printf("配置无变化，跳过重启")
		return nil
	}

	m.logger.Printf("检测到配置变化，准备更新服务器")

	// 更新配置
	m.config = newConfig

	// 如果管理器已启动，需要重启
	if m.started {
		return m.Restart()
	}

	return nil
}

// GetListenAddresses 获取所有监听地址
func (m *Manager) GetListenAddresses() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var addresses []string
	for _, instance := range m.servers {
		if instance.Started {
			addr := fmt.Sprintf("%s://%s:%d",
				instance.Config.Protocol,
				instance.Config.Host,
				instance.Config.Port)
			addresses = append(addresses, addr)
		}
	}

	return addresses
}
