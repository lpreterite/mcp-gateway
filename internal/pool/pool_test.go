package pool

import (
	"testing"

	"github.com/packy/mcp-gateway/internal/config"
)

func TestNewPool(t *testing.T) {
	cfg := config.PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
	}

	p := NewPool(cfg)
	if p == nil {
		t.Fatal("NewPool() returned nil")
	}

	if p.config.MinConnections != 1 {
		t.Errorf("Expected MinConnections=1, got %d", p.config.MinConnections)
	}

	if p.config.MaxConnections != 5 {
		t.Errorf("Expected MaxConnections=5, got %d", p.config.MaxConnections)
	}
}

func TestToolCallResult(t *testing.T) {
	result := ToolCallResult{
		Content: []map[string]interface{}{
			{"type": "text", "text": "hello"},
			{"type": "text", "text": "world"},
		},
		IsError: false,
	}

	if len(result.Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(result.Content))
	}

	if result.IsError {
		t.Error("Expected IsError=false")
	}

	result2 := ToolCallResult{
		Content: []map[string]interface{}{
			{"type": "text", "text": "error occurred"},
		},
		IsError: true,
	}

	if !result2.IsError {
		t.Error("Expected IsError=true")
	}
}

func TestPoolGetStats(t *testing.T) {
	cfg := config.PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
	}

	p := NewPool(cfg)

	// 空池统计
	stats := p.GetStats()
	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}

	// 空池应该没有统计
	if len(stats) != 0 {
		t.Errorf("Expected 0 servers in stats, got %d", len(stats))
	}
}
