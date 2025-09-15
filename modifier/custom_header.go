/*
Create: 2025/1/15
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package modifier

import (
	"net/http"
	"sandwich/config"
	"sandwich/log"
	"sandwich/utils"
	"sync"
)

// CustomHeaderModifier 自定义响应头修改器
type CustomHeaderModifier struct {
	enabled bool
	headers map[string]string
	mu      sync.RWMutex // 读写锁保护headers配置
}

// NewCustomHeaderModifier 创建新的自定义响应头修改器实例
func NewCustomHeaderModifier() *CustomHeaderModifier {
	cfg := config.Get()
	cm := &CustomHeaderModifier{
		enabled: len(cfg.CustomHeader) > 0, // 如果有自定义头配置则启用
		headers: make(map[string]string),
	}

	// 复制配置中的自定义头
	cm.mu.Lock()
	for key, value := range cfg.CustomHeader {
		cm.headers[key] = value
	}
	cm.mu.Unlock()

	return cm
}

func (cm *CustomHeaderModifier) Use(response *http.Response) {
	_ = cm.ModifyResponse(response)
}

// ModifyResponse 处理响应的自定义头添加
func (c *CustomHeaderModifier) ModifyResponse(response *http.Response) error {
	// 检查是否启用
	if !c.enabled {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 添加所有配置的自定义头
	utils.AddHeader(response, c.headers)

	return nil
}

// IsEnabled 返回是否启用自定义头修改器
func (c *CustomHeaderModifier) IsEnabled() bool {
	return c.enabled
}

// UpdateConfig 更新配置（支持热更新）
func (c *CustomHeaderModifier) UpdateConfig() {
	cfg := config.Get()

	c.mu.Lock()
	defer c.mu.Unlock()

	// 清空现有配置
	c.headers = make(map[string]string)

	// 重新加载配置
	for key, value := range cfg.CustomHeader {
		c.headers[key] = value
	}

	// 更新启用状态
	c.enabled = len(c.headers) > 0

	log.DebugF("自定义头配置已更新: enabled=%v, headers=%v", c.enabled, c.headers)
}

// GetName 获取修改器名称
func (c *CustomHeaderModifier) GetName() string {
	return "custom-header"
}

// AddHeader 动态添加自定义头
func (c *CustomHeaderModifier) AddHeader(key, value string) {
	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.headers[key] = value
	// 如果之前没有头部配置，现在启用修改器
	if !c.enabled && len(c.headers) > 0 {
		c.enabled = true
	}

	log.DebugF("动态添加自定义头: %s = %s", key, value)
}

// RemoveHeader 动态移除自定义头
func (c *CustomHeaderModifier) RemoveHeader(key string) {
	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.headers, key)
	// 如果没有头部配置了，禁用修改器
	if len(c.headers) == 0 {
		c.enabled = false
	}

	log.DebugF("动态移除自定义头: %s", key)
}

// GetHeaders 获取当前所有自定义头（只读副本）
func (c *CustomHeaderModifier) GetHeaders() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	headers := make(map[string]string)
	for key, value := range c.headers {
		headers[key] = value
	}
	return headers
}

// SetHeaders 批量设置自定义头
func (c *CustomHeaderModifier) SetHeaders(headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.headers = make(map[string]string)
	for key, value := range headers {
		c.headers[key] = value
	}

	c.enabled = len(c.headers) > 0

	log.DebugF("批量设置自定义头: enabled=%v, count=%d", c.enabled, len(c.headers))
}

// ClearHeaders 清空所有自定义头
func (c *CustomHeaderModifier) ClearHeaders() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.headers = make(map[string]string)
	c.enabled = false

	log.Debug("已清空所有自定义头")
}

// HasHeader 检查是否包含指定的头部
func (c *CustomHeaderModifier) HasHeader(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.headers[key]
	return exists
}

// GetHeader 获取指定头部的值
func (c *CustomHeaderModifier) GetHeader(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.headers[key]
	return value, exists
}
