package registry

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.tools == nil {
		t.Error("NewRegistry().tools is nil")
	}
}

func TestRegistryRegisterTool(t *testing.T) {
	r := NewRegistry()

	tool := ToolInfo{
		Name:         "test-tool",
		Description:  "A test tool",
		ServerName:   "test-server",
		OriginalName: "test-tool",
	}

	r.RegisterTool(tool)

	if r.Count() != 1 {
		t.Errorf("Expected 1 tool, got %d", r.Count())
	}

	// 注册同名工具应该覆盖
	tool2 := ToolInfo{
		Name:         "test-tool",
		Description:  "Updated test tool",
		ServerName:   "test-server",
		OriginalName: "test-tool",
	}
	r.RegisterTool(tool2)

	if r.Count() != 1 {
		t.Errorf("Expected 1 tool after overwrite, got %d", r.Count())
	}
}

func TestRegistryGetTool(t *testing.T) {
	r := NewRegistry()

	tool := ToolInfo{
		Name:         "test-tool",
		Description:  "A test tool",
		ServerName:   "test-server",
		OriginalName: "test-tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}

	r.RegisterTool(tool)

	got, ok := r.GetTool("test-tool")
	if !ok {
		t.Fatal("GetTool() returned false for existing tool")
	}

	if got.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", got.Name)
	}

	if got.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", got.Description)
	}

	// 获取不存在的工具
	_, ok = r.GetTool("non-existent")
	if ok {
		t.Error("GetTool() should return false for non-existent tool")
	}
}

func TestRegistryGetAllTools(t *testing.T) {
	r := NewRegistry()

	tools := []ToolInfo{
		{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"},
		{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"},
		{Name: "tool-3", ServerName: "server-2", OriginalName: "tool-3"},
	}

	for _, tool := range tools {
		r.RegisterTool(tool)
	}

	allTools := r.GetAllTools()
	if len(allTools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(allTools))
	}
}

func TestRegistryCount(t *testing.T) {
	r := NewRegistry()

	if r.Count() != 0 {
		t.Errorf("Expected 0 tools, got %d", r.Count())
	}

	r.RegisterTool(ToolInfo{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"})
	if r.Count() != 1 {
		t.Errorf("Expected 1 tool, got %d", r.Count())
	}

	r.RegisterTool(ToolInfo{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"})
	if r.Count() != 2 {
		t.Errorf("Expected 2 tools, got %d", r.Count())
	}
}
