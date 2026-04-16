package registry

import (
	"strings"

	"github.com/lpreterite/mcp-gateway/src/config"
)

// MappingRule 映射规则
type MappingRule struct {
	ServerName  string
	Prefix      string
	StripPrefix bool
	RenameMap   map[string]string
}

// Mapper 工具名映射器
type Mapper struct {
	rules        map[string]MappingRule
	reverseIndex map[string]string
	filters      map[string]config.ToolFilterConfig
}

// NewMapper 创建新的映射器
func NewMapper(mapping map[string]config.MappingConfig, toolFilters map[string]config.ToolFilterConfig) *Mapper {
	m := &Mapper{
		rules:        make(map[string]MappingRule),
		reverseIndex: make(map[string]string),
		filters:      make(map[string]config.ToolFilterConfig),
	}

	for serverName, cfg := range mapping {
		m.addMapping(serverName, cfg)
	}

	for serverName, filter := range toolFilters {
		if filter.Include != nil || filter.Exclude != nil {
			m.filters[serverName] = filter
		}
	}

	return m
}

// addMapping 添加映射规则
func (m *Mapper) addMapping(serverName string, cfg config.MappingConfig) {
	renameMap := make(map[string]string)
	if cfg.Rename != nil {
		for from, to := range cfg.Rename {
			renameMap[from] = to
		}
	}

	rule := MappingRule{
		ServerName:  serverName,
		Prefix:      cfg.Prefix,
		StripPrefix: cfg.StripPrefix,
		RenameMap:   renameMap,
	}

	m.rules[serverName] = rule
	m.reverseIndex[strings.ToLower(cfg.Prefix)] = serverName
}

// GetServerForTool 根据工具名获取服务器名称
func (m *Mapper) GetServerForTool(toolName string) string {
	lowerName := strings.ToLower(toolName)
	for prefix, serverName := range m.reverseIndex {
		if strings.HasPrefix(lowerName, strings.ToLower(prefix+"_")) {
			return serverName
		}
		if lowerName == strings.ToLower(prefix) {
			return serverName
		}
	}
	return ""
}

// GetOriginalToolName 获取原始工具名
func (m *Mapper) GetOriginalToolName(gatewayToolName string, serverName string) string {
	rule, ok := m.rules[serverName]
	if !ok {
		return gatewayToolName
	}

	// 先检查是否需要去除前缀
	if rule.StripPrefix {
		prefix := rule.Prefix + "_"
		if strings.HasPrefix(strings.ToLower(gatewayToolName), strings.ToLower(prefix)) {
			gatewayToolName = gatewayToolName[len(prefix):]
		}
	}

	// 检查重命名映射（key 是原始工具名，不带前缀）
	if originalName, ok := rule.RenameMap[gatewayToolName]; ok {
		return originalName
	}

	return gatewayToolName
}

// GetGatewayToolName 获取网关工具名
func (m *Mapper) GetGatewayToolName(originalName string, serverName string) string {
	rule, ok := m.rules[serverName]
	if !ok {
		return originalName
	}

	if rule.StripPrefix {
		return rule.Prefix + "_" + originalName
	}

	return originalName
}

// ShouldIncludeTool 检查工具是否应该包含
func (m *Mapper) ShouldIncludeTool(serverName string, toolName string) bool {
	filter, ok := m.filters[serverName]
	if !ok {
		return true
	}

	if len(filter.Include) > 0 {
		for _, name := range filter.Include {
			if name == toolName {
				return true
			}
		}
		return false
	}

	if len(filter.Exclude) > 0 {
		for _, name := range filter.Exclude {
			if name == toolName {
				return false
			}
		}
	}

	return true
}

// GetAllPrefixes 获取所有前缀
func (m *Mapper) GetAllPrefixes() []string {
	prefixes := make([]string, 0, len(m.rules))
	for _, rule := range m.rules {
		prefixes = append(prefixes, rule.Prefix)
	}
	return prefixes
}

// GetRuleForServer 获取服务器映射规则
func (m *Mapper) GetRuleForServer(serverName string) MappingRule {
	return m.rules[serverName]
}
