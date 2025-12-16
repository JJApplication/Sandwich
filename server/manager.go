package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net"
	"net/http"
	"reflect"
	autocert2 "sandwich/autocert"
	"strings"
	"sync"
	"time"

	"sandwich/config"
	"sandwich/log"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/singleflight"
)

// limitedReader 用于限制请求体读取，超出限制时直接拒绝
type limitedReader struct {
	io.ReadCloser
	limit    int64
	read     int64
	logger   *zerolog.Logger
	host     string
	exceeded bool
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.exceeded {
		return 0, fmt.Errorf("request entity too large")
	}

	n, err = lr.ReadCloser.Read(p)
	lr.read += int64(n)

	if lr.read > lr.limit {
		lr.exceeded = true
		if lr.logger != nil {
			lr.logger.Error().Str("Host", lr.host).Int64("Read", lr.read).Int64("Limit", lr.limit).Msg("请求体超出限制被拒绝")
		}
		return 0, fmt.Errorf("request entity too large")
	}

	return n, err
}

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
	logger  *zerolog.Logger            // 日志记录器

	// AutoTLS
	acmeMgr            *autocert.Manager  // autocert 管理器
	acmeMU             sync.Mutex         // 刷新证书过程的互斥锁，避免并发冲突
	autoCertTempServer *http.Server       // 刷新证书时临时占用80端口的HTTP服务器
	stoppedHTTP80      *ServerInstance    // 刷新前被停止的80端口HTTP服务器，用于刷新后恢复
	sf                 singleflight.Group // 用于合并并发的证书请求
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
func NewManager(cfg *config.Config, handler http.Handler, logger *zerolog.Logger) *Manager {
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
		ReadTimeout:       time.Second * time.Duration(m.defInt64(serverConfig.ReadTimeout, 30)),
		WriteTimeout:      time.Second * time.Duration(m.defInt64(serverConfig.WriteTimeout, 30)),
		IdleTimeout:       time.Second * time.Duration(m.defInt64(serverConfig.IdleTimeout, 60)),
		ReadHeaderTimeout: time.Second * time.Duration(m.defInt64(serverConfig.ReadHeaderTimeout, 10)),
		// 设置最大请求体大小
		MaxHeaderBytes: m.defInt(int(serverConfig.MaxHeaderBytes), 5<<20), // 1MB header limit
		Protocols:      proto,
	}

	// 配置最大请求体大小和自动重定向
	if serverConfig.MaxRequestBody > 0 || serverConfig.Protocol == "http" {
		// 通过中间件控制请求体大小和处理重定向
		originalHandler := instance.Server.Handler
		instance.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 内部请求不需要判断重定向

			// 检查是否需要自动重定向HTTP到HTTPS
			if r.Header.Get(m.config.ProxyHeader.BackendHeader) == "" && serverConfig.Protocol == "http" {
				// 添加调试日志
				m.logger.Debug().Str("Host", r.Host).Str("URI", r.RequestURI).Str("Protocol", serverConfig.Protocol).Msg("HTTP请求")

				// 检查请求的域名是否配置了自动重定向
				host := r.Host
				// 移除端口号
				if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
					host = host[:colonIndex]
				}

				m.logger.Debug().Str("域名", host).Str("原始Host", r.Host).Msg("处理域名")

				// 查找匹配的域名配置
				for i, domainConfig := range serverConfig.DomainConfig {
					m.logger.Debug().Int("域名组", i).Bool("自动重定向", domainConfig.AutoRedirect).Any("域名", domainConfig.Domains).Msg("检查域名配置")

					if domainConfig.AutoRedirect {
						// 检查当前域名是否在配置的域名列表中
						for j, configuredDomain := range domainConfig.Domains {
							m.logger.Debug().Int("域名组", j).Str("Host", host).Str("Domain", configuredDomain).Msg("检查域名匹配")

							if host == configuredDomain || (strings.HasPrefix(configuredDomain, "*.") && strings.HasSuffix(host, configuredDomain[1:])) {
								// 构建HTTPS重定向URL
								httpsURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
								m.logger.Debug().Str("源URL", r.URL.String()).Str("目标URL", httpsURL).Str("配置域名", configuredDomain).Msg("域名匹配成功，执行重定向")
								// 执行301永久重定向
								w.Header().Set("Location", httpsURL)

								// 根据配置设置HSTS头部
								if domainConfig.HSTSMaxAge > 0 {
									hstsValue := fmt.Sprintf("max-age=%d", domainConfig.HSTSMaxAge)
									if domainConfig.HSTSSubdomains {
										hstsValue += "; includeSubDomains"
									}
									if domainConfig.HSTSPreload {
										hstsValue += "; preload"
									}
									w.Header().Set("Strict-Transport-Security", hstsValue)
									m.logger.Debug().Str("HSTS头", hstsValue).Msg("设置HSTS头部")
								} else {
									m.logger.Debug().Msg("未设置HSTS头部 (HSTSMaxAge=0)")
								}

								w.WriteHeader(http.StatusMovedPermanently)
								return
							}
						}
					}
				}
			}

			// 检查Content-Length头，如果超出限制直接返回413
			if serverConfig.MaxRequestBody > 0 && r.ContentLength > serverConfig.MaxRequestBody {
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte("Request entity too large"))
				m.logger.Printf("请求体过大被拒绝: Host=%s, ContentLength=%d, Limit=%d",
					r.Host, r.ContentLength, serverConfig.MaxRequestBody)
				return
			}

			// 如果没有Content-Length头或为-1，使用limitedReader进行保护性限制
			if serverConfig.MaxRequestBody > 0 && (r.ContentLength == -1 || r.ContentLength == 0) {
				r.Body = &limitedReader{
					ReadCloser: r.Body,
					limit:      serverConfig.MaxRequestBody,
					logger:     m.logger,
					host:       r.Host,
				}
			}

			originalHandler.ServeHTTP(w, r)
		})
		m.logger.Printf("服务器 %s 设置最大请求体大小: %d bytes (%.2f MB)",
			serverConfig.Name, serverConfig.MaxRequestBody, float64(serverConfig.MaxRequestBody)/(1024*1024))
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
		// 使用 autocert 自动管理证书，返回用于标准 http.Server 的 *tls.Config
		// 构建域名白名单（必须提供域名，否则不可启动AutoTLS）
		domains := config.GetTlsDomains(m.config)
		if len(domains) == 0 {
			return fmt.Errorf("自动TLS已启用但未配置任何域名，无法申请证书")
		}

		// 初始化或复用 autocert 管理器
		if m.acmeMgr == nil {
			m.acmeMgr = autocert2.NewCertManager(domains, m.config.Features.AutoCert.Email)
		}

		// 基础TLS配置来自autocert
		base := m.acmeMgr.TLSConfig()
		// 包装 GetCertificate，在每次自动刷新前释放80端口并启用挑战处理，刷新后恢复
		origGetCert := base.GetCertificate
		base.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// 使用singleflight合并并发的证书请求
			// 避免同一域名并发握手时多次触发80端口停止/启动
			val, err, _ := m.sf.Do("cert:"+hello.ServerName, func() (interface{}, error) {
				m.acmeMU.Lock()
				defer m.acmeMU.Unlock()

				m.logger.Info().Str("域名", hello.ServerName).Msg("AutoTLS: 即将获取/刷新证书, 准备处理80端口")
				if err := m.beforeHandleAutoCert(); err != nil {
					m.logger.Error().Err(err).Msg("AutoTLS: beforeHandleAutoCert 失败")
				}

				cert, err := origGetCert(hello)

				if err2 := m.afterHandleAutoCert(); err2 != nil {
					m.logger.Error().Err(err2).Msg("AutoTLS: afterHandleAutoCert 失败")
				}

				if err != nil {
					m.logger.Error().Err(err).Msg("AutoTLS: 获取证书失败")
					return nil, err
				} else {
					m.logger.Info().Str("域名", hello.ServerName).Msg("AutoTLS: 证书获取/刷新完成")
					return cert, nil
				}
			})

			if err != nil {
				return nil, err
			}
			return val.(*tls.Certificate), nil
		}

		// 强化TLS安全参数
		base.MinVersion = tls.VersionTLS12
		base.PreferServerCipherSuites = true

		// 应用 TLS 配置
		instance.Server.TLSConfig = base
		instance.Listener = tls.NewListener(instance.Listener, base)

		return nil
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

// beforeHandleAutoCert 在autocert发起证书申请/刷新前，确保80端口可用于HTTP-01挑战
func (m *Manager) beforeHandleAutoCert() error {
	// 查找当前占用80端口的HTTP服务器
	var http80 *ServerInstance
	m.mu.RLock()
	for _, inst := range m.servers {
		if inst.Config.Protocol == "http" && inst.Config.Port == 80 && inst.Started {
			http80 = inst
			break
		}
	}
	m.mu.RUnlock()

	// 如有占用则先停止
	if http80 != nil {
		m.logger.Info().Str("服务", http80.Name).Msg("AutoTLS: 检测到80端口被服务器占用, 先停止该HTTP服务器")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := http80.Server.Shutdown(ctx); err != nil {
			m.logger.Error().Err(err).Msg("AutoTLS: 停止80端口HTTP服务器失败")
		}
		m.stoppedHTTP80 = http80
		// 从映射移除，避免状态混淆
		m.mu.Lock()
		delete(m.servers, http80.Name)
		m.mu.Unlock()
	}

	// 启动临时挑战服务器，监听80端口，仅用于处理 /.well-known/acme-challenge/
	addr := fmt.Sprintf("0.0.0.0:%d", 80)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("AutoTLS: 创建80端口挑战监听失败: %v", err)
	}

	tempSrv := &http.Server{
		Addr:              addr,
		Handler:           m.acmeMgr.HTTPHandler(nil),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	m.autoCertTempServer = tempSrv
	go func() {
		if err := tempSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			m.logger.Error().Err(err).Msg("AutoTLS: 挑战服务器运行错误")
		}
	}()
	m.logger.Info().Str("地址", addr).Msg("AutoTLS: 临时挑战服务器已开启")
	return nil
}

// afterHandleAutoCert 在证书申请/刷新完成后，关闭挑战服务器并恢复原80端口HTTP服务
func (m *Manager) afterHandleAutoCert() error {
	// 关闭临时挑战服务器
	if m.autoCertTempServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.autoCertTempServer.Shutdown(ctx)
		m.autoCertTempServer = nil
		m.logger.Info().Msg("AutoTLS: 临时挑战服务器已关闭")
	}

	// 恢复原HTTP服务器（如存在）
	if m.stoppedHTTP80 != nil {
		m.logger.Info().Str("服务", m.stoppedHTTP80.Name).Msg("AutoTLS: 重新启动原HTTP服务器 (80端口)")
		if err := m.startServer(m.stoppedHTTP80.Config); err != nil {
			m.logger.Error().Err(err).Msg("AutoTLS: 重启80端口HTTP服务器失败")
		}
		m.stoppedHTTP80 = nil
	}
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

// 为基础类型提供判空处理
func (m *Manager) defInt(v, d int) int {
	if reflect.ValueOf(v).IsZero() {
		return d
	}
	return v
}

func (m *Manager) defInt64(v, d int64) int64 {
	if reflect.ValueOf(v).IsZero() {
		return d
	}
	return v
}

func (m *Manager) defString(v, d string) string {
	if reflect.ValueOf(v).IsZero() {
		return d
	}
	return v
}
