package registry

import (
	"testing"

	"github.com/packy/mcp-gateway/src/config"
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
