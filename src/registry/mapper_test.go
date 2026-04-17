package registry

import (
	"testing"

	"github.com/lpreterite/mcp-gateway/src/config"
)

func TestNewMapper(t *testing.T) {
	m := NewMapper(nil, nil)
	if m == nil {
		t.Fatal("NewMapper() returned nil")
	}
}

func TestMapperGetGatewayToolName(t *testing.T) {
	m := NewMapper(nil, nil)

	// 无前缀情况
	name := m.GetGatewayToolName("tool1", "server1")
	if name != "tool1" {
		t.Errorf("Expected 'tool1', got '%s'", name)
	}
}

func TestMapperWithPrefix(t *testing.T) {
	// GetGatewayToolName 在 StripPrefix=true 时添加 prefix + "_"
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
			Rename:      nil,
		},
	}

	m := NewMapper(mapping, nil)

	name := m.GetGatewayToolName("tool1", "server1")
	if name != "srv1_tool1" {
		t.Errorf("Expected 'srv1_tool1', got '%s'", name)
	}

	// 不在 mapping 中的服务器应该返回原名
	name = m.GetGatewayToolName("tool1", "unknown-server")
	if name != "tool1" {
		t.Errorf("Expected 'tool1' for unknown server, got '%s'", name)
	}

	// StripPrefix=false 时不添加前缀
	mapping2 := map[string]config.MappingConfig{
		"server2": {
			Prefix:      "srv2",
			StripPrefix: false,
		},
	}
	m2 := NewMapper(mapping2, nil)
	name = m2.GetGatewayToolName("tool1", "server2")
	if name != "tool1" {
		t.Errorf("Expected 'tool1' when StripPrefix=false, got '%s'", name)
	}
}

func TestMapperRename(t *testing.T) {
	// GetGatewayToolName 不处理 Rename，只处理 Prefix
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
			Rename: map[string]string{
				"old-name": "new-name",
			},
		},
	}

	m := NewMapper(mapping, nil)

	// GetGatewayToolName 只添加前缀
	name := m.GetGatewayToolName("old-name", "server1")
	if name != "srv1_old-name" {
		t.Errorf("Expected 'srv1_old-name', got '%s'", name)
	}

	// GetOriginalToolName 去除前缀后再应用重命名
	original := m.GetOriginalToolName("srv1_old-name", "server1")
	if original != "new-name" {
		t.Errorf("Expected 'new-name', got '%s'", original)
	}

	// 非重命名工具
	original = m.GetOriginalToolName("srv1_tool1", "server1")
	if original != "tool1" {
		t.Errorf("Expected 'tool1', got '%s'", original)
	}
}

func TestMapperShouldIncludeTool(t *testing.T) {
	filters := map[string]config.ToolFilterConfig{
		"server1": {
			Include: []string{"tool1", "tool2"},
			Exclude: nil,
		},
	}

	m := NewMapper(nil, filters)

	// 在 include 列表中
	if !m.ShouldIncludeTool("server1", "tool1") {
		t.Error("tool1 should be included")
	}

	// 不在 include 列表中
	if m.ShouldIncludeTool("server1", "tool3") {
		t.Error("tool3 should not be included (not in include list)")
	}
}

func TestMapperExcludeTool(t *testing.T) {
	filters := map[string]config.ToolFilterConfig{
		"server1": {
			Exclude: []string{"secret-tool"},
		},
	}

	m := NewMapper(nil, filters)

	// 在 exclude 列表中
	if m.ShouldIncludeTool("server1", "secret-tool") {
		t.Error("secret-tool should not be included")
	}

	// 不在 exclude 列表中
	if !m.ShouldIncludeTool("server1", "other-tool") {
		t.Error("other-tool should be included")
	}
}

func TestMapperGetOriginalToolName(t *testing.T) {
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
			Rename: map[string]string{
				"old-name": "new-name",
			},
		},
	}

	m := NewMapper(mapping, nil)

	// GetOriginalToolName 去除前缀后再应用重命名
	original := m.GetOriginalToolName("srv1_old-name", "server1")
	if original != "new-name" {
		t.Errorf("Expected 'new-name', got '%s'", original)
	}

	// 不带前缀的工具名直接返回（不会匹配到 rule）
	original = m.GetOriginalToolName("tool1", "server1")
	if original != "tool1" {
		t.Errorf("Expected 'tool1', got '%s'", original)
	}
}

func TestMapperGetServerForTool(t *testing.T) {
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
		},
		"server2": {
			Prefix:      "srv2",
			StripPrefix: true,
		},
	}

	m := NewMapper(mapping, nil)

	// 匹配带前缀的工具名
	server := m.GetServerForTool("srv1_tool1")
	if server != "server1" {
		t.Errorf("Expected 'server1', got '%s'", server)
	}

	server = m.GetServerForTool("srv2_tool2")
	if server != "server2" {
		t.Errorf("Expected 'server2', got '%s'", server)
	}

	// 不带前缀的工具名不匹配
	server = m.GetServerForTool("tool1")
	if server != "" {
		t.Errorf("Expected empty string for non-prefixed tool, got '%s'", server)
	}

	// 未知的工具名
	server = m.GetServerForTool("unknown_tool")
	if server != "" {
		t.Errorf("Expected empty string for unknown tool, got '%s'", server)
	}

	// 大小写不敏感
	server = m.GetServerForTool("SRV1_TOOL1")
	if server != "server1" {
		t.Errorf("Expected 'server1' (case insensitive), got '%s'", server)
	}
}

func TestMapperGetAllPrefixes(t *testing.T) {
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
		},
		"server2": {
			Prefix:      "srv2",
			StripPrefix: true,
		},
	}

	m := NewMapper(mapping, nil)

	prefixes := m.GetAllPrefixes()
	if len(prefixes) != 2 {
		t.Errorf("Expected 2 prefixes, got %d", len(prefixes))
	}

	// 检查是否包含预期的前缀
	found := make(map[string]bool)
	for _, p := range prefixes {
		found[p] = true
	}
	if !found["srv1"] {
		t.Error("Expected 'srv1' in prefixes")
	}
	if !found["srv2"] {
		t.Error("Expected 'srv2' in prefixes")
	}
}

func TestMapperGetRuleForServer(t *testing.T) {
	mapping := map[string]config.MappingConfig{
		"server1": {
			Prefix:      "srv1",
			StripPrefix: true,
			Rename: map[string]string{
				"old": "new",
			},
		},
	}

	m := NewMapper(mapping, nil)

	rule := m.GetRuleForServer("server1")
	if rule.ServerName != "server1" {
		t.Errorf("Expected ServerName 'server1', got '%s'", rule.ServerName)
	}
	if rule.Prefix != "srv1" {
		t.Errorf("Expected Prefix 'srv1', got '%s'", rule.Prefix)
	}
	if !rule.StripPrefix {
		t.Error("Expected StripPrefix to be true")
	}
	if rule.RenameMap["old"] != "new" {
		t.Error("Expected RenameMap to contain 'old' -> 'new'")
	}

	// 不存在的服务器
	rule = m.GetRuleForServer("non-existent")
	if rule.ServerName != "" {
		t.Errorf("Expected empty rule for non-existent server, got '%s'", rule.ServerName)
	}
}

func TestMapperEmptyMapping(t *testing.T) {
	m := NewMapper(nil, nil)

	// 空映射应该正常工作
	name := m.GetGatewayToolName("tool1", "server1")
	if name != "tool1" {
		t.Errorf("Expected 'tool1', got '%s'", name)
	}

	server := m.GetServerForTool("tool1")
	if server != "" {
		t.Errorf("Expected empty string, got '%s'", server)
	}

	prefixes := m.GetAllPrefixes()
	if len(prefixes) != 0 {
		t.Errorf("Expected 0 prefixes, got %d", len(prefixes))
	}

	if !m.ShouldIncludeTool("server1", "any-tool") {
		t.Error("All tools should be included when no filters")
	}
}
