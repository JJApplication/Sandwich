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
func (cm *CustomHeaderModifier) ModifyResponse(response *http.Response) error {
	// 检查是否启用
	if !cm.enabled {
		return nil
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 添加所有配置的自定义头
	utils.AddHeader(response, cm.headers)

	return nil
}

// IsEnabled 返回是否启用自定义头修改器
func (cm *CustomHeaderModifier) IsEnabled() bool {
	return cm.enabled
}

// UpdateConfig 更新配置（支持热更新）
func (cm *CustomHeaderModifier) UpdateConfig() {
	cfg := config.Get()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 清空现有配置
	cm.headers = make(map[string]string)

	// 重新加载配置
	for key, value := range cfg.CustomHeader {
		cm.headers[key] = value
	}

	// 更新启用状态
	cm.enabled = len(cm.headers) > 0

	log.DebugF("自定义头配置已更新: enabled=%v, headers=%v", cm.enabled, cm.headers)
}

// GetName 获取修改器名称
func (cm *CustomHeaderModifier) GetName() string {
	return "custom-header"
}

// AddHeader 动态添加自定义头
func (cm *CustomHeaderModifier) AddHeader(key, value string) {
	if key == "" {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.headers[key] = value
	// 如果之前没有头部配置，现在启用修改器
	if !cm.enabled && len(cm.headers) > 0 {
		cm.enabled = true
	}

	log.DebugF("动态添加自定义头: %s = %s", key, value)
}

// RemoveHeader 动态移除自定义头
func (cm *CustomHeaderModifier) RemoveHeader(key string) {
	if key == "" {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.headers, key)
	// 如果没有头部配置了，禁用修改器
	if len(cm.headers) == 0 {
		cm.enabled = false
	}

	log.DebugF("动态移除自定义头: %s", key)
}

// GetHeaders 获取当前所有自定义头（只读副本）
func (cm *CustomHeaderModifier) GetHeaders() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	headers := make(map[string]string)
	for key, value := range cm.headers {
		headers[key] = value
	}
	return headers
}

// SetHeaders 批量设置自定义头
func (cm *CustomHeaderModifier) SetHeaders(headers map[string]string) {
	if headers == nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.headers = make(map[string]string)
	for key, value := range headers {
		cm.headers[key] = value
	}

	cm.enabled = len(cm.headers) > 0

	log.DebugF("批量设置自定义头: enabled=%v, count=%d", cm.enabled, len(cm.headers))
}

// ClearHeaders 清空所有自定义头
func (cm *CustomHeaderModifier) ClearHeaders() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.headers = make(map[string]string)
	cm.enabled = false

	log.Debug("已清空所有自定义头")
}

// HasHeader 检查是否包含指定的头部
func (cm *CustomHeaderModifier) HasHeader(key string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.headers[key]
	return exists
}

// GetHeader 获取指定头部的值
func (cm *CustomHeaderModifier) GetHeader(key string) (string, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	value, exists := cm.headers[key]
	return value, exists
}
