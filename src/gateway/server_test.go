package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestHandleMessages(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册会话到 GlobalNotifier
	ch := make(chan string, 10)
	GlobalNotifier.Add("test-session", ch)

	defer GlobalNotifier.Remove("test-session")

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/messages?sessionId=test-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %q", resp.JSONRPC)
	}
}

func TestHandleMessagesMissingSessionId(t *testing.T) {
	s := newReadyTestServer(t)

	req := httptest.NewRequest("POST", "/messages", nil)
	w := httptest.NewRecorder()

	s.handleMessages(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMessagesSessionNotFound(t *testing.T) {
	s := newReadyTestServer(t)

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/messages?sessionId=non-existent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleMessages(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleMessagesInvalidJSON(t *testing.T) {
	s := newReadyTestServer(t)

	ch := make(chan string, 10)
	GlobalNotifier.Add("test-session", ch)
	defer GlobalNotifier.Remove("test-session")

	req := httptest.NewRequest("POST", "/messages?sessionId=test-session", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleMessages(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleSSEPOST(t *testing.T) {
	s := newReadyTestServer(t)

	// 先建立一个 SSE 会话
	ch := make(chan string, 100)
	GlobalNotifier.Add("sse-test-session", ch)
	defer GlobalNotifier.Remove("sse-test-session")

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/sse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Session-ID", "sse-test-session")
	w := httptest.NewRecorder()

	s.handleSSEPOST(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}

func TestHandleSSEPOSTNoSession(t *testing.T) {
	s := newReadyTestServer(t)

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/sse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleSSEPOST(w, req)

	// 没有 SSE 会话时，应该直接返回响应
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %q", resp.JSONRPC)
	}
}

func TestHandleSSEPOSTInvalidJSON(t *testing.T) {
	s := newReadyTestServer(t)

	req := httptest.NewRequest("POST", "/sse", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleSSEPOST(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleSSEPOSTFallbackToExistingSession(t *testing.T) {
	s := newReadyTestServer(t)

	// 建立一个 sse- 前缀的会话
	ch := make(chan string, 100)
	GlobalNotifier.Add("sse-existing-session", ch)
	defer GlobalNotifier.Remove("sse-existing-session")

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/sse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 MCP-Session-ID，期望 fallback 到现有的 sse- 前缀会话
	w := httptest.NewRecorder()

	s.handleSSEPOST(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}
}

func TestHandleHealthDegraded(t *testing.T) {
	s := newTestServer(t)
	s.pool = pool.NewPool(*s.config.Pool)
	s.ready.Store(true)                                            // 设置为 ready
	s.initErr.Store(fmt.Errorf("initialization partially failed")) // 设置错误

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

	if resp.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got %q", resp.Status)
	}
}

func TestHandleSSEChannelFull(t *testing.T) {
	s := newReadyTestServer(t)

	// 建立一个满的通道
	ch := make(chan string, 1)
	ch <- "first" // 填满
	GlobalNotifier.Add("sse-full-channel", ch)
	defer GlobalNotifier.Remove("sse-full-channel")

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/sse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Session-ID", "sse-full-channel")
	w := httptest.NewRecorder()

	s.handleSSEPOST(w, req)

	// 通道满时应该返回 ServiceUnavailable
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestSetupRoutes(t *testing.T) {
	s := newTestServer(t)
	// SetupRoutes 只是注册处理器，不会 panic
	s.SetupRoutes()

	// 验证 mux 已设置
	if s.mux == nil {
		t.Error("mux should not be nil after SetupRoutes")
	}
}

// noFlusherResponseWriter 是一个不支持 Flusher 的 ResponseWriter
type noFlusherResponseWriter struct {
	header http.Header
	status int
	body   *bytes.Buffer
}

func newNoFlusherResponseWriter() *noFlusherResponseWriter {
	return &noFlusherResponseWriter{
		header: make(http.Header),
		status: 200,
		body:   &bytes.Buffer{},
	}
}

func (w *noFlusherResponseWriter) Header() http.Header {
	return w.header
}

func (w *noFlusherResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *noFlusherResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

// TestHandleSSETransportNil 测试 transport 为 nil 的情况
func TestHandleSSETransportNil(t *testing.T) {
	s := newReadyTestServer(t)

	// 创建一个不支持 Flusher 的 ResponseWriter
	w := newNoFlusherResponseWriter()
	req := httptest.NewRequest("GET", "/sse", nil)

	// 由于 NewSSETransport 返回 nil，会触发 http.Error
	s.handleSSE(w, req)

	if w.status != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.status)
	}
}

// TestProcessJSONRPCRequest_ToolsCallWithInvalidParams 测试 tools/call 发送无效 params
func TestProcessJSONRPCRequest_ToolsCallWithInvalidParams(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册测试工具
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "test-tool",
		Description: "Test tool",
		ServerName:  "test-server",
	})

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/call",
		Params: json.RawMessage(`{invalid json}`), // 无效的 JSON
	}

	resp := s.processJSONRPCRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error for invalid params")
	}

	if resp.Error.Code != -32700 {
		t.Errorf("Expected error code -32700, got %d", resp.Error.Code)
	}
}

// TestProcessJSONRPCRequest_ToolsCallToolNotFound 测试 tools/call 工具不存在
func TestProcessJSONRPCRequest_ToolsCallToolNotFound(t *testing.T) {
	s := newReadyTestServer(t)

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "non-existent-tool"}`),
	}

	resp := s.processJSONRPCRequest(req)

	if resp.Error == nil {
		t.Fatal("Expected error for tool not found")
	}

	if resp.Error.Code != -32603 {
		t.Errorf("Expected error code -32603, got %d", resp.Error.Code)
	}
}

// TestHandleListToolsEncodeError 测试 json.Encode 错误（通过 mock）
// 由于 json.Encode 在正常情况下不会失败，这里只验证 handleListTools 的基本逻辑
func TestHandleListToolsEmpty(t *testing.T) {
	s := newReadyTestServer(t)
	// 不注册任何工具

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

	if len(resp.Tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(resp.Tools))
	}
}

// TestHandleMessagesEncodeError 测试 json.Encode 错误
// 同样，这里主要验证 handleMessages 的基本逻辑
func TestHandleMessagesJSONRPCResponse(t *testing.T) {
	s := newReadyTestServer(t)

	ch := make(chan string, 10)
	GlobalNotifier.Add("test-session-json", ch)
	defer GlobalNotifier.Remove("test-session-json")

	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/messages?sessionId=test-session-json", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", w.Header().Get("Content-Type"))
	}
}

// TestToolCallResponseWithResult 测试包含结果的响应
func TestToolCallResponseWithResult(t *testing.T) {
	// 这个测试验证 ToolCallResponse 包含 Result 时的序列化
	resp := ToolCallResponse{
		Result: &ToolCallResult{
			Content: []ContentBlock{
				{Type: "text", Text: "success output"},
			},
			IsError: false,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolCallResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Error != "" {
		t.Errorf("Error should be empty, got %q", decoded.Error)
	}
	if decoded.Result == nil {
		t.Fatal("Result should not be nil")
	}
	if decoded.Result.IsError {
		t.Error("IsError should be false")
	}
}

// TestProcessJSONRPCRequest_ToolsCallSuccess 测试 tools/call 成功路径
// 由于没有真实服务器，这个测试验证 processJSONRPCRequest 正确处理响应
func TestProcessJSONRPCRequest_ToolsCallServerError(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册工具，但 pool 没有真实服务器
	s.registry.RegisterTool(registry.ToolInfo{
		Name:        "mock-tool",
		Description: "Mock tool",
		ServerName:  "mock-server",
	})

	req := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "mock-tool", "arguments": {}}`),
	}

	resp := s.processJSONRPCRequest(req)

	// 由于 pool 没有真实服务器，callTool 返回 IsError=true 的结果而不是返回 error
	// 所以 resp.Error 是 nil，但 resp.Result 中 IsError 应该为 true
	if resp.Error != nil {
		t.Fatalf("Expected no JSON-RPC error, got %v", resp.Error)
	}

	if resp.Result == nil {
		t.Fatal("Expected result to be present")
	}

	// resp.Result 是 *pool.ToolCallResult 类型
	// 通过 JSON 序列化/反序列化来验证
	resultData, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	var decodedResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resultData, &decodedResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if !decodedResult.IsError {
		t.Error("Expected isError=true for tool call without real server")
	}
}

// TestNewSSETransportWithRealWriter 使用真实的 http.ResponseWriter 测试
// 注意：这个测试会创建一个实际的 transport
func TestNewSSETransportWithRealWriter(t *testing.T) {
	// 创建一个支持 Flusher 的 httptest.ResponseRecorder
	w := httptest.NewRecorder()
	transport := NewSSETransport("session-real", w)

	if transport == nil {
		t.Fatal("NewSSETransport() should not return nil for httptest.ResponseRecorder")
	}

	if transport.SessionID != "session-real" {
		t.Errorf("SessionID = %q, want 'session-real'", transport.SessionID)
	}

	// 验证 headers 已设置
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want 'text/event-stream'", w.Header().Get("Content-Type"))
	}
}

// TestHandleSSEReceiveMessage 测试 handleSSE 接收消息的情况
func TestHandleSSEReceiveMessage(t *testing.T) {
	s := newReadyTestServer(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sse", nil)

	// 在另一个 goroutine 中发送消息
	done := make(chan struct{})
	go func() {
		// 等待 transport 被创建并注册
		time.Sleep(100 * time.Millisecond)

		// 找到注册的 session
		GlobalNotifier.mu.RLock()
		for sessID, ch := range GlobalNotifier.channels {
			if strings.HasPrefix(sessID, "sse-") {
				ch <- `{"test":"data"}`
				break
			}
		}
		GlobalNotifier.mu.RUnlock()

		// 等待一段时间让 handleSSE 处理消息
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	// 启动 handleSSE（但它会阻塞直到 context 取消或 channel 关闭）
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// 修改 req 的 context
	req = req.WithContext(ctx)

	// handleSSE 会因为 context 超时而返回
	s.handleSSE(w, req)

	// 清理 GlobalNotifier
	GlobalNotifier.mu.Lock()
	for sessID := range GlobalNotifier.channels {
		if strings.HasPrefix(sessID, "sse-") {
			GlobalNotifier.Remove(sessID)
		}
	}
	GlobalNotifier.mu.Unlock()
}

// TestHandleSSESessionCleanup 测试 handleSSE 结束时正确清理 session
func TestHandleSSESessionCleanup(t *testing.T) {
	s := newReadyTestServer(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sse", nil)

	// 快速取消 context 让 handleSSE 退出
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	req = req.WithContext(ctx)

	s.handleSSE(w, req)

	// 由于使用了 defer transport.Close()，session 应该被清理
	// 但 GlobalNotifier 的通道可能在 transport.Close() 中被移除
	// 这里我们主要验证 handleSSE 不会 panic
	_ = cancel // 避免编译错误
}

// TestSSETransportSendEvent 测试发送各种类型的 SSE 事件
func TestSSETransportSendEvent(t *testing.T) {
	w := httptest.NewRecorder()
	transport := NewSSETransport("session-event-test", w)

	err := transport.Send("message", `{"type":"test"}`)
	if err != nil {
		t.Errorf("Send() returned error: %v", err)
	}

	err = transport.Send("connected", `{"status":"ok"}`)
	if err != nil {
		t.Errorf("Send() returned error: %v", err)
	}

	// 验证响应体包含两个事件
	body := w.Body.String()
	if !strings.Contains(body, "event: message") {
		t.Error("Response should contain 'event: message'")
	}
	if !strings.Contains(body, "event: connected") {
		t.Error("Response should contain 'event: connected'")
	}
}

// TestHandleToolCallSuccessResponse 测试 handleToolCall 成功响应路径
// 由于没有真实服务器，我们主要验证响应格式
func TestHandleToolCallErrorResponse(t *testing.T) {
	s := newReadyTestServer(t)

	// 注册一个不存在的工具会返回错误
	reqBody := ToolCallRequest{
		Name:      "non-existent-tool",
		Arguments: map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tools/call", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleToolCall(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ToolCallResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// 应该返回错误而不是结果
	if resp.Error == "" {
		t.Error("Expected error message for non-existent tool")
	}
}
