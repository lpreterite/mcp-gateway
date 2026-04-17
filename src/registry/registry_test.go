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

func TestRegistryUnregisterTool(t *testing.T) {
	r := NewRegistry()

	r.RegisterTool(ToolInfo{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"})
	r.RegisterTool(ToolInfo{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"})

	if r.Count() != 2 {
		t.Fatalf("Expected 2 tools, got %d", r.Count())
	}

	// 注销存在的工具
	r.UnregisterTool("tool-1")
	if r.Count() != 1 {
		t.Errorf("Expected 1 tool after unregister, got %d", r.Count())
	}

	// 注销不存在的工具不应该 panic
	r.UnregisterTool("non-existent")
	if r.Count() != 1 {
		t.Errorf("Count should still be 1 after unregister non-existent, got %d", r.Count())
	}
}

func TestRegistryGetToolsByServer(t *testing.T) {
	r := NewRegistry()

	tools := []ToolInfo{
		{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"},
		{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"},
		{Name: "tool-3", ServerName: "server-2", OriginalName: "tool-3"},
	}

	for _, tool := range tools {
		r.RegisterTool(tool)
	}

	server1Tools := r.GetToolsByServer("server-1")
	if len(server1Tools) != 2 {
		t.Errorf("Expected 2 tools for server-1, got %d", len(server1Tools))
	}

	server2Tools := r.GetToolsByServer("server-2")
	if len(server2Tools) != 1 {
		t.Errorf("Expected 1 tool for server-2, got %d", len(server2Tools))
	}

	noServerTools := r.GetToolsByServer("non-existent")
	if len(noServerTools) != 0 {
		t.Errorf("Expected 0 tools for non-existent server, got %d", len(noServerTools))
	}
}

func TestRegistryHasTool(t *testing.T) {
	r := NewRegistry()

	r.RegisterTool(ToolInfo{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"})

	if !r.HasTool("tool-1") {
		t.Error("HasTool() should return true for existing tool")
	}

	if r.HasTool("non-existent") {
		t.Error("HasTool() should return false for non-existent tool")
	}
}

func TestRegistryClear(t *testing.T) {
	r := NewRegistry()

	r.RegisterTool(ToolInfo{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"})
	r.RegisterTool(ToolInfo{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"})

	if r.Count() != 2 {
		t.Fatalf("Expected 2 tools before clear, got %d", r.Count())
	}

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Expected 0 tools after clear, got %d", r.Count())
	}
}

func TestRegistryClearByServer(t *testing.T) {
	r := NewRegistry()

	tools := []ToolInfo{
		{Name: "tool-1", ServerName: "server-1", OriginalName: "tool-1"},
		{Name: "tool-2", ServerName: "server-1", OriginalName: "tool-2"},
		{Name: "tool-3", ServerName: "server-2", OriginalName: "tool-3"},
	}

	for _, tool := range tools {
		r.RegisterTool(tool)
	}

	if r.Count() != 3 {
		t.Fatalf("Expected 3 tools, got %d", r.Count())
	}

	// 清除 server-1 的工具
	r.ClearByServer("server-1")

	if r.Count() != 1 {
		t.Errorf("Expected 1 tool after clear, got %d", r.Count())
	}

	// tool-3 应该还在
	if !r.HasTool("tool-3") {
		t.Error("tool-3 should still exist")
	}

	// 清除不存在的服务器不应该 panic
	r.ClearByServer("non-existent")
}
