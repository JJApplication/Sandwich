/*
Create: 2025/1/15
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

package modifier

import (
	"net/http"
	"sandwich/log"
	"sync"
)

// ModifierManager 修改器管理器
// 负责管理和协调所有响应修改器
type ModifierManager struct {
	lock      *sync.RWMutex
	chain     *ModifierChain
	modifiers []Modifier
}

var (
	m *ModifierManager
)

func init() {
	m = NewModifierManager()
	m.InitModifiers()
}

func GetManager() *ModifierManager {
	if m != nil {
		return m
	}
	return new(ModifierManager)
}

// NewModifierManager 创建新的修改器管理器
func NewModifierManager() *ModifierManager {
	manager := &ModifierManager{
		lock:      new(sync.RWMutex),
		chain:     NewModifierChain(),
		modifiers: make([]Modifier, 0),
	}

	// 注册默认的修改器
	manager.registerDefaultModifiers()

	return manager
}

// registerDefaultModifiers 注册默认的修改器
func (mm *ModifierManager) registerDefaultModifiers() {
	// 注册gzip压缩修改器
	gzipModifier := NewGzipModifier()
	mm.chain.AddModifier(gzipModifier)
	log.DebugF("已注册修改器: %s", gzipModifier.GetName())

	// 注册自定义头修改器
	customHeaderModifier := NewCustomHeaderModifier()
	mm.chain.AddModifier(customHeaderModifier)
	log.DebugF("已注册修改器: %s", customHeaderModifier.GetName())
}

func (mm *ModifierManager) InitModifiers() {
	// add trace
	mm.RegisterModifier(NewTraceModifier())
	// add secure header
	mm.RegisterModifier(NewSecureHeaderModifier())
	// no cache
	mm.RegisterModifier(NewNoCache())
	// custom header
	mm.RegisterModifier(NewCustomHeaderModifier())
	// 应用gzip压缩中间件
	mm.RegisterModifier(NewGzipModifier())
}

func (mm *ModifierManager) RegisterModifier(modifier Modifier) {
	mm.lock.Lock()
	defer mm.lock.Unlock()
	mm.modifiers = append(mm.modifiers, modifier)
}

// ModifyResponse 对响应应用所有启用的修改器
func (mm *ModifierManager) ModifyResponse(response *http.Response) error {
	return mm.chain.ModifyResponse(response)
}

// UpdateConfig 更新所有修改器的配置
func (mm *ModifierManager) UpdateConfig() {
	mm.chain.UpdateConfig()
	log.Debug("所有修改器配置已更新")
}

// GetEnabledModifiers 获取所有启用的修改器
func (mm *ModifierManager) GetEnabledModifiers() []Modifier {
	return mm.chain.GetEnabledModifiers()
}

// GetModifierByName 根据名称获取修改器
func (mm *ModifierManager) GetModifierByName(name string) Modifier {
	return mm.chain.GetModifierByName(name)
}

// AddCustomModifier 添加自定义修改器
func (mm *ModifierManager) AddCustomModifier(modifier Modifier) {
	if modifier != nil {
		mm.chain.AddModifier(modifier)
		log.DebugF("已添加自定义修改器: %s", modifier.GetName())
	}
}

// GetGzipModifier 获取gzip修改器实例（类型安全的访问方式）
func (mm *ModifierManager) GetGzipModifier() *GzipModifier {
	modifier := mm.GetModifierByName("gzip")
	if gzipModifier, ok := modifier.(*GzipModifier); ok {
		return gzipModifier
	}
	return nil
}

// GetCustomHeaderModifier 获取自定义头修改器实例（类型安全的访问方式）
func (mm *ModifierManager) GetCustomHeaderModifier() *CustomHeaderModifier {
	modifier := mm.GetModifierByName("custom_header")
	if customHeaderModifier, ok := modifier.(*CustomHeaderModifier); ok {
		return customHeaderModifier
	}
	return nil
}

// GetStatus 获取修改器管理器状态信息
func (mm *ModifierManager) GetStatus() map[string]interface{} {
	enabledModifiers := mm.GetEnabledModifiers()

	status := map[string]interface{}{
		"total_modifiers":   len(mm.chain.modifiers),
		"enabled_modifiers": len(enabledModifiers),
		"modifiers":         make([]map[string]interface{}, 0),
	}

	for _, modifier := range mm.chain.modifiers {
		modifierInfo := map[string]interface{}{
			"name":    modifier.GetName(),
			"enabled": modifier.IsEnabled(),
		}

		// 添加特定修改器的详细信息
		switch m := modifier.(type) {
		case *GzipModifier:
			modifierInfo["level"] = m.GetLevel()
			modifierInfo["types"] = m.GetTypes()
		case *CustomHeaderModifier:
			modifierInfo["headers_count"] = len(m.GetHeaders())
		}

		status["modifiers"] = append(status["modifiers"].([]map[string]interface{}), modifierInfo)
	}

	return status
}

func (mm *ModifierManager) GetModifiers() []Modifier {
	mm.lock.RLock()
	defer mm.lock.RUnlock()
	return mm.modifiers
}
