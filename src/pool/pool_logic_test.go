package pool

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lpreterite/mcp-gateway/src/config"
)

// TestNewMCPClientConnection 测试 MCPClientConnection 创建
func TestNewMCPClientConnection(t *testing.T) {
	cfg := config.ServerConfig{
		Command: []string{"echo", "hello"},
		Name:    "test-client",
	}

	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	if client.config.Name != cfg.Name {
		t.Error("Config.Name not stored correctly")
	}
	if len(client.config.Command) != len(cfg.Command) {
		t.Error("Config.Command not stored correctly")
	}
	if client.pending == nil {
		t.Error("pending map not initialized")
	}
	if client.lastUsed.IsZero() {
		t.Error("lastUsed should be set to current time")
	}
}

// TestMCPClientConnectionGetName 测试名称返回
func TestMCPClientConnectionGetName(t *testing.T) {
	cfg := config.ServerConfig{Name: "my-tool"}
	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	if client.GetName() != "my-tool" {
		t.Errorf("Expected 'my-tool', got '%s'", client.GetName())
	}
}

// TestMCPClientConnectionTouch 测试时间更新
func TestMCPClientConnectionTouch(t *testing.T) {
	cfg := config.ServerConfig{Name: "test"}
	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	oldTime := client.lastUsed
	time.Sleep(1 * time.Millisecond)
	client.Touch()

	if !client.lastUsed.After(oldTime) {
		t.Error("lastUsed should be updated")
	}
}

// TestMCPClientConnectionIsConnected 测试初始连接状态
func TestMCPClientConnectionIsConnected(t *testing.T) {
	cfg := config.ServerConfig{Name: "test"}
	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	if client.IsConnected() {
		t.Error("New client should not be connected")
	}
}

// TestToolCallResultJSONSerialization 测试 JSON 序列化
func TestToolCallResultJSONSerialization(t *testing.T) {
	result := ToolCallResult{
		Content: []map[string]interface{}{
			{"type": "text", "text": "hello"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded ToolCallResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.IsError != result.IsError {
		t.Error("IsError mismatch")
	}
	if len(decoded.Content) != len(result.Content) {
		t.Error("Content length mismatch")
	}
}

// TestPoolGetStatsWithConnections 测试有连接时的统计
func TestPoolGetStatsWithConnections(t *testing.T) {
	cfg := config.PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
		AcquireTimeout: 1000,
	}
	pool := NewPool(cfg)
	stats := pool.GetStats()

	// 空池应该没有统计
	if len(stats) != 0 {
		t.Errorf("Expected 0 servers in stats, got %d", len(stats))
	}
}

// TestPoolDisconnectAllEmpty 测试空池断开
func TestPoolDisconnectAllEmpty(t *testing.T) {
	cfg := config.PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
	}
	pool := NewPool(cfg)
	pool.DisconnectAll()

	stats := pool.GetStats()
	if len(stats) != 0 {
		t.Errorf("Expected 0 servers in stats after disconnect all, got %d", len(stats))
	}
}

// TestNewMCPClientConnectionEmptyCommand 测试空命令配置
func TestNewMCPClientConnectionEmptyCommand(t *testing.T) {
	cfg := config.ServerConfig{
		Command: nil,
		Name:    "empty-command",
	}
	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	if client.GetName() != "empty-command" {
		t.Error("Name should still be set")
	}
	if client.IsConnected() {
		t.Error("Should not be connected with nil command")
	}
}

// TestMCPClientConnectionGetLastUsed 测试获取最后使用时间
func TestMCPClientConnectionGetLastUsed(t *testing.T) {
	cfg := config.ServerConfig{Name: "test"}
	client := NewMCPClientConnection(cfg, &DefaultProcessStarter{})

	// 验证返回的时间不是零值
	lastUsed := client.GetLastUsed()
	if lastUsed.IsZero() {
		t.Error("lastUsed should not be zero")
	}

	// 验证时间在合理范围内（不早于1秒前）
	oneSecondAgo := time.Now().Add(-1 * time.Second)
	if lastUsed.Before(oneSecondAgo) {
		t.Error("lastUsed should be within the last second")
	}
}

// TestToolCallResultWithError 测试错误结果
func TestToolCallResultWithError(t *testing.T) {
	result := ToolCallResult{
		Content: []map[string]interface{}{
			{"type": "text", "text": "error occurred"},
		},
		IsError: true,
	}

	if !result.IsError {
		t.Error("Expected IsError=true")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(result.Content))
	}
}
