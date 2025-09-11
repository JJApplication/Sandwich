package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sandwich/log"
	"sandwich/utils"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"sandwich/config"
)

// Server WebSocket 服务器
// 负责处理 WebSocket 连接的升级和转发
type Server struct {
	config    config.WebSocketConfig // WebSocket 配置
	upgrader  websocket.Upgrader     // WebSocket 升级器
	conns     map[string]*Connection // 活跃连接映射
	connsMu   sync.RWMutex           // 连接映射锁
	logger    *log.Log               // 日志记录器
	ctx       context.Context        // 上下文
	cancel    context.CancelFunc     // 取消函数
	started   bool                   // 是否已启动
	startedMu sync.RWMutex           // 启动状态锁
}

// Connection WebSocket 连接
// 表示一个 WebSocket 连接及其相关信息
type Connection struct {
	ID         string             // 连接 ID
	ClientConn *websocket.Conn    // 客户端连接
	ServerConn *websocket.Conn    // 服务端连接
	Backend    string             // 后端地址
	CreatedAt  time.Time          // 创建时间
	LastPing   time.Time          // 最后心跳时间
	ctx        context.Context    // 连接上下文
	cancel     context.CancelFunc // 取消函数
	mu         sync.RWMutex       // 连接锁
	closed     bool               // 是否已关闭
}

// NewServer 创建新的 WebSocket 服务器
func NewServer(cfg config.WebSocketConfig, logger *log.Log) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	if logger == nil {
		logger = log.GetLogger()
	}

	// 配置 WebSocket 升级器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.BufferSize,
		WriteBufferSize: cfg.BufferSize,
		// 允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return &Server{
		config:   cfg,
		upgrader: upgrader,
		conns:    make(map[string]*Connection),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动 WebSocket 服务器
func (s *Server) Start() error {
	s.startedMu.Lock()
	defer s.startedMu.Unlock()

	if s.started {
		return fmt.Errorf("WebSocket 服务器已经启动")
	}

	if !s.config.Enabled {
		return fmt.Errorf("WebSocket 功能未启用")
	}

	// 启动心跳检查
	go s.startHeartbeat()

	// 启动连接清理
	go s.startCleanup()

	s.started = true
	s.logger.Printf("WebSocket 服务器已启动")

	return nil
}

// Stop 停止 WebSocket 服务器
func (s *Server) Stop() error {
	s.startedMu.Lock()
	defer s.startedMu.Unlock()

	if !s.started {
		return fmt.Errorf("WebSocket 服务器未启动")
	}

	s.logger.Printf("正在停止 WebSocket 服务器...")

	// 关闭所有连接
	s.closeAllConnections()

	// 取消上下文
	s.cancel()

	s.started = false
	s.logger.Printf("WebSocket 服务器已停止")

	return nil
}

// HandleUpgrade 处理 WebSocket 升级请求
func (s *Server) HandleUpgrade(w http.ResponseWriter, r *http.Request) error {
	if !s.IsStarted() {
		return fmt.Errorf("WebSocket 服务器未启动")
	}

	// 检查是否为 WebSocket 升级请求
	if !s.isWebSocketRequest(r) {
		return fmt.Errorf("不是有效的 WebSocket 升级请求")
	}

	// 升级客户端连接
	clientConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("升级客户端连接失败: %v", err)
	}

	// 后端服务直接由proxy代理 直接转发到开启ws的端口
	backend := fmt.Sprintf("127.0.0.1:%s", config.GetWsPort(config.Get()))
	// 连接到后端服务器
	serverConn, err := s.connectToBackend(backend, r)
	if err != nil {
		clientConn.Close()
		return fmt.Errorf("连接后端服务器失败: %v", err)
	}

	// 创建连接对象
	conn := s.createConnection(clientConn, serverConn, backend)

	// 注册连接
	s.registerConnection(conn)

	// 启动数据转发
	go s.forwardData(conn)

	s.logger.Printf("WebSocket 连接已建立: %s -> %s", r.RemoteAddr, backend)

	return nil
}

// isWebSocketRequest 检查是否为 WebSocket 请求
func (s *Server) isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// selectBackend 选择后端服务器
func (s *Server) selectBackend(backends []string) (string, error) {
	if len(backends) == 0 {
		return "", fmt.Errorf("没有可用的后端服务器")
	}

	// 简单的轮询选择（可以根据需要实现更复杂的负载均衡）
	// 这里使用时间戳作为简单的选择策略
	index := int(time.Now().UnixNano()) % len(backends)
	return backends[index], nil
}

// connectToBackend 连接到后端服务器
func (s *Server) connectToBackend(backend string, originalReq *http.Request) (*websocket.Conn, error) {
	// 构建后端 WebSocket URL
	backendURL, err := s.buildBackendURL(backend, originalReq)
	if err != nil {
		return nil, fmt.Errorf("构建后端 URL 失败: %v", err)
	}

	// 复制原始请求头
	headers := make(http.Header)
	for key, values := range originalReq.Header {
		// 跳过一些不需要转发的头部
		if s.shouldSkipHeader(key) {
			continue
		}
		headers[key] = values
	}

	// 创建 WebSocket 拨号器
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   s.config.BufferSize,
		WriteBufferSize:  s.config.BufferSize,
	}

	// 连接到后端
	conn, _, err := dialer.Dial(backendURL, headers)
	if err != nil {
		return nil, fmt.Errorf("拨号到后端失败: %v", err)
	}

	return conn, nil
}

// buildBackendURL 构建后端 WebSocket URL
func (s *Server) buildBackendURL(backend string, originalReq *http.Request) (string, error) {
	// 解析后端地址
	if !strings.Contains(backend, "://") {
		// 如果没有协议，根据原始请求判断
		scheme := "ws"
		if originalReq.TLS != nil {
			scheme = "wss"
		}
		backend = scheme + "://" + backend
	}

	backendURL, err := url.Parse(backend)
	if err != nil {
		return "", err
	}

	// 保持原始路径和查询参数
	backendURL.Path = originalReq.URL.Path
	backendURL.RawQuery = originalReq.URL.RawQuery

	return backendURL.String(), nil
}

// shouldSkipHeader 判断是否应该跳过某个头部
func (s *Server) shouldSkipHeader(key string) bool {
	skipHeaders := []string{
		"Connection",
		"Upgrade",
		"Sec-WebSocket-Key",
		"Sec-WebSocket-Version",
		"Sec-WebSocket-Accept",
		"Sec-WebSocket-Extensions",
		"Sec-WebSocket-Protocol",
	}

	for _, skipHeader := range skipHeaders {
		if strings.EqualFold(key, skipHeader) {
			return true
		}
	}

	return false
}

// createConnection 创建连接对象
func (s *Server) createConnection(clientConn, serverConn *websocket.Conn, backend string) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)

	connID := fmt.Sprintf("%d", time.Now().UnixNano())

	return &Connection{
		ID:         connID,
		ClientConn: clientConn,
		ServerConn: serverConn,
		Backend:    backend,
		CreatedAt:  time.Now(),
		LastPing:   time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// registerConnection 注册连接
func (s *Server) registerConnection(conn *Connection) {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	s.conns[conn.ID] = conn
}

// unregisterConnection 注销连接
func (s *Server) unregisterConnection(connID string) {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	delete(s.conns, connID)
}

// forwardData 转发数据
func (s *Server) forwardData(conn *Connection) {
	defer func() {
		conn.Close()
		s.unregisterConnection(conn.ID)
	}()

	// 启动客户端到服务端的转发
	go s.forwardClientToServer(conn)

	// 启动服务端到客户端的转发
	s.forwardServerToClient(conn)
}

// forwardClientToServer 转发客户端到服务端的数据
func (s *Server) forwardClientToServer(conn *Connection) {
	defer conn.ServerConn.Close()

	for {
		select {
		case <-conn.ctx.Done():
			return
		default:
		}

		// 设置读取超时
		conn.ClientConn.SetReadDeadline(time.Now().Add(utils.ToSecond(s.config.PongTimeout)))

		// 读取客户端消息
		msgType, data, err := conn.ClientConn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				s.logger.Printf("读取客户端消息失败: %v", err)
			}
			return
		}

		// 转发到服务端
		if err := conn.ServerConn.WriteMessage(msgType, data); err != nil {
			s.logger.Printf("转发到服务端失败: %v", err)
			return
		}
	}
}

// forwardServerToClient 转发服务端到客户端的数据
func (s *Server) forwardServerToClient(conn *Connection) {
	defer conn.ClientConn.Close()

	for {
		select {
		case <-conn.ctx.Done():
			return
		default:
		}

		// 设置读取超时
		conn.ServerConn.SetReadDeadline(time.Now().Add(utils.ToSecond(s.config.PongTimeout)))

		// 读取服务端消息
		msgType, data, err := conn.ServerConn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				s.logger.Printf("读取服务端消息失败: %v", err)
			}
			return
		}

		// 转发到客户端
		if err := conn.ClientConn.WriteMessage(msgType, data); err != nil {
			s.logger.Printf("转发到客户端失败: %v", err)
			return
		}
	}
}

// startHeartbeat 启动心跳检查
func (s *Server) startHeartbeat() {
	ticker := time.NewTicker(utils.ToSecond(s.config.PingInterval))
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pingConnections()
		}
	}
}

// pingConnections 对所有连接发送心跳
func (s *Server) pingConnections() {
	s.connsMu.RLock()
	conns := make([]*Connection, 0, len(s.conns))
	for _, conn := range s.conns {
		conns = append(conns, conn)
	}
	s.connsMu.RUnlock()

	for _, conn := range conns {
		go s.pingConnection(conn)
	}
}

// pingConnection 对单个连接发送心跳
func (s *Server) pingConnection(conn *Connection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return
	}

	// 发送 ping 到客户端
	if err := conn.ClientConn.WriteMessage(websocket.PingMessage, nil); err != nil {
		s.logger.Printf("发送心跳到客户端失败: %v", err)
		conn.Close()
		return
	}

	// 发送 ping 到服务端
	if err := conn.ServerConn.WriteMessage(websocket.PingMessage, nil); err != nil {
		s.logger.Printf("发送心跳到服务端失败: %v", err)
		conn.Close()
		return
	}

	conn.LastPing = time.Now()
}

// startCleanup 启动连接清理
func (s *Server) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupConnections()
		}
	}
}

// cleanupConnections 清理超时连接
func (s *Server) cleanupConnections() {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	timeout := utils.ToSecond(s.config.PongTimeout)
	now := time.Now()

	for id, conn := range s.conns {
		if now.Sub(conn.LastPing) > timeout {
			s.logger.Printf("清理超时连接: %s", id)
			conn.Close()
			delete(s.conns, id)
		}
	}
}

// closeAllConnections 关闭所有连接
func (s *Server) closeAllConnections() {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	for _, conn := range s.conns {
		conn.Close()
	}

	s.conns = make(map[string]*Connection)
}

// Close 关闭连接
func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.cancel()

	if c.ClientConn != nil {
		c.ClientConn.Close()
	}

	if c.ServerConn != nil {
		c.ServerConn.Close()
	}
}

// IsStarted 检查服务器是否已启动
func (s *Server) IsStarted() bool {
	s.startedMu.RLock()
	defer s.startedMu.RUnlock()
	return s.started
}

// GetStats 获取统计信息
func (s *Server) GetStats() *Stats {
	s.connsMu.RLock()
	connCount := len(s.conns)
	s.connsMu.RUnlock()

	return &Stats{
		Enabled:           s.config.Enabled,
		Started:           s.IsStarted(),
		ActiveConnections: connCount,
		MaxMessageSize:    int(s.config.MaxMessageSize),
		BufferSize:        s.config.BufferSize,
		PingInterval:      utils.ToSecond(s.config.PingInterval),
		PongTimeout:       utils.ToSecond(s.config.PongTimeout) * time.Second,
	}
}

// Stats WebSocket 统计信息
type Stats struct {
	Enabled           bool          `json:"enabled"`            // 是否启用
	Started           bool          `json:"started"`            // 是否已启动
	ActiveConnections int           `json:"active_connections"` // 活跃连接数
	MaxMessageSize    int           `json:"max_message_size"`   // 最大消息大小
	BufferSize        int           `json:"buffer_size"`        // 缓冲区大小
	PingInterval      time.Duration `json:"ping_interval"`      // 心跳间隔
	PongTimeout       time.Duration `json:"pong_timeout"`       // 心跳超时
}
