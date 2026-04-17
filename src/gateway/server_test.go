package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/pool"
	"github.com/lpreterite/mcp-gateway/src/registry"
)

func newTestServer(t *testing.T) *Server {
	cfg := &config.Config{
		Pool: &config.PoolConfig{
			MinConnections: 1,
			MaxConnections: 5,
		},
	}

	s := NewServer(cfg)
	// 默认不设置 ready，测试需要时自己设置
	return s
}

func newReadyTestServer(t *testing.T) *Server {
	s := newTestServer(t)
	s.ready.Store(true)
	return s
}

func TestHandleHealth(t *testing.T) {
	s := newReadyTestServer(t)
	s.pool = pool.NewPool(*s.config.Pool)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Server 处于 ready 状态，所以 status 应该是 "ok"
	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got %q", resp.Status)
	}

	if !resp.Ready {
		t.Error("Expected Ready to be true")
	}
}

func TestHandleHealthNotReady(t *testing.T) {
	s := newTestServer(t)
	// 不设置 ready 为 true，保持初始状态

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "initializing" {
		t.Errorf("Expected status 'initializing', got %q", resp.Status)
	}

	if resp.Ready {
		t.Error("Expected Ready to be false")
	}
}

func TestHandleListTools(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册一些测试工具
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "test-tool",
		Description: "A test tool",
		ServerName:  "test-server",
	})

	req := httptest.NewRequest("GET", "/tools", nil)
	w := httptest.NewRecorder()

	s.handleListTools(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ToolsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(resp.Tools))
	}

	if resp.Tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %q", resp.Tools[0].Name)
	}
}

func TestHandleListToolsNotReady(t *testing.T) {
	s := newTestServer(t)
	s.ready.Store(false) // 设置为未就绪状态

	req := httptest.NewRequest("GET", "/tools", nil)
	w := httptest.NewRecorder()

	s.handleListTools(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var resp ToolsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(resp.Tools))
	}
}

func TestHandleToolCall(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册测试工具
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "echo-tool",
		Description: "Echo tool",
		ServerName:  "test-server",
	})

	reqBody := ToolCallRequest{
		Name: "echo-tool",
		Arguments: map[string]interface{}{
			"message": "hello",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tools/call", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleToolCall(w, req)

	// 由于 pool 没有真实的 MCP 服务器，这里会返回 IsError=true 的结果
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ToolCallResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// 由于没有真实服务器，应该返回 IsError=true 的结果
	if resp.Result == nil {
		t.Fatal("Expected result to be present")
	}
	if !resp.Result.IsError {
		t.Error("Expected IsError=true for tool call without real server")
	}
}

func TestHandleToolCallNotReady(t *testing.T) {
	s := newTestServer(t)
	s.ready.Store(false)

	reqBody := ToolCallRequest{
		Name:      "test-tool",
		Arguments: map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tools/call", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleToolCall(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var resp ToolCallResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "gateway is still initializing" {
		t.Errorf("Expected error 'gateway is still initializing', got %q", resp.Error)
	}
}

func TestHandleToolCallMissingName(t *testing.T) {
	s := newReadyTestServer(t)

	reqBody := map[string]interface{}{}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tools/call", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleToolCall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleToolCallInvalidJSON(t *testing.T) {
	s := newReadyTestServer(t)

	req := httptest.NewRequest("POST", "/tools/call", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleToolCall(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProcessJSONRPCRequest_Initialize(t *testing.T) {
	s := newReadyTestServer(t)

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}

	resp := s.processJSONRPCRequest(req)

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %q", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be map[string]interface{}")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion '2024-11-05', got %v", result["protocolVersion"])
	}
}

func TestProcessJSONRPCRequest_ToolsList(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册测试工具
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "tool1",
		Description: "First tool",
		ServerName:  "server1",
	})
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "tool2",
		Description: "Second tool",
		ServerName:  "server1",
	})

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}

	resp := s.processJSONRPCRequest(req)

	if resp.Error != nil {
		t.Fatalf("Expected no error, got %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be map[string]interface{}")
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tools to be []map[string]interface{}")
	}

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestProcessJSONRPCRequest_ToolsListNotReady(t *testing.T) {
	s := newTestServer(t)
	s.ready.Store(false)

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}

	resp := s.processJSONRPCRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error for not ready state")
	}

	if resp.Error.Code != -32000 {
		t.Errorf("Expected error code -32000, got %d", resp.Error.Code)
	}
}

func TestProcessJSONRPCRequest_UnknownMethod(t *testing.T) {
	s := newTestServer(t)

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "unknown/method",
		Params: json.RawMessage(`{}`),
	}

	resp := s.processJSONRPCRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestCallTool(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册测试工具
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "test-tool",
		Description: "Test tool",
		ServerName:  "test-server",
	})

	result, err := s.callTool("test-tool", map[string]interface{}{"key": "value"})

	// 由于没有真实服务器，callTool 返回 IsError=true 的结果而不是 error
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("Expected result to be present")
	}
	if !result.IsError {
		t.Error("Expected IsError=true when calling tool without real server")
	}
}

func TestCallToolNotFound(t *testing.T) {
	s := newReadyTestServer(t)

	_, err := s.callTool("non-existent-tool", nil)

	if err == nil {
		t.Error("Expected error for non-existent tool")
	}
}

func TestCallToolNotReady(t *testing.T) {
	s := newTestServer(t)

	_, err := s.callTool("test-tool", nil)

	if err == nil {
		t.Error("Expected error when server is not ready")
	}
}

func TestFormatInitResults(t *testing.T) {
	results := []initServerResult{
		{
			ServerName:      "server1",
			CollectedTools:  5,
			RegisteredTools: 3,
			Skipped:         false,
		},
		{
			ServerName: "server2",
			Skipped:    true,
		},
		{
			ServerName: "server3",
			Err:        fmt.Errorf("connection failed"),
		},
	}

	formatted := formatInitResults(results)

	if len(formatted) != 3 {
		t.Fatalf("Expected 3 formatted results, got %d", len(formatted))
	}

	// 检查第一个结果
	if formatted[0]["server"] != "server1" {
		t.Errorf("Expected server 'server1', got %v", formatted[0]["server"])
	}
	if formatted[0]["collected"].(int) != 5 {
		t.Errorf("Expected collected 5, got %v", formatted[0]["collected"])
	}
	if formatted[0]["registered"].(int) != 3 {
		t.Errorf("Expected registered 3, got %v", formatted[0]["registered"])
	}

	// 检查第二个结果（skipped）
	if formatted[1]["skipped"].(bool) != true {
		t.Error("Expected skipped to be true")
	}

	// 检查第三个结果（error）
	if formatted[2]["error"] == nil {
		t.Error("Expected error to be present")
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "string value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			input:    map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"key": nil},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("getString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsReady(t *testing.T) {
	s := &Server{}
	s.ready.Store(true)

	if !s.isReady() {
		t.Error("Expected isReady() to return true")
	}

	s.ready.Store(false)

	if s.isReady() {
		t.Error("Expected isReady() to return false")
	}
}

func TestInitializationError(t *testing.T) {
	s := &Server{}

	// 没有错误时
	if err := s.initializationError(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 设置错误
	s.initErr.Store(fmt.Errorf("test error"))

	err := s.initializationError()
	if err == nil {
		t.Error("Expected error to be returned")
	}
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got %q", err.Error())
	}
}
