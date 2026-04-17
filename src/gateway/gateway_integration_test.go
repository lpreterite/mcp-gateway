package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/pool"
	"github.com/lpreterite/mcp-gateway/src/registry"
)

// --- 辅助函数：创建配置 ---

func newTestConfig() *config.Config {
	return &config.Config{
		Gateway: &config.GatewayConfig{
			Host: "127.0.0.1",
			Port: 0, // 随机端口
		},
		Pool: &config.PoolConfig{
			MinConnections: 1,
			MaxConnections: 5,
		},
		Servers: []config.ServerConfig{},
	}
}

// --- 辅助函数：创建测试服务器（已设置路由）--

func newTestServerWithRoutes(t *testing.T) *Server {
	cfg := newTestConfig()
	srv := NewServer(cfg)
	srv.SetupRoutes() // 关键：先设置路由
	srv.ready.Store(true)
	return srv
}

// --- 辅助函数：创建未就绪的测试服务器 ---

func newTestServerNotReady() *Server {
	cfg := newTestConfig()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	// 不设置 ready
	return srv
}

// --- 集成测试：HTTP 服务器生命周期 ---

// TestServerStartAndStop 测试服务器的启动和停止
func TestServerStartAndStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			Host: "127.0.0.1",
			Port: 0, // 使用随机端口
		},
		Pool: &config.PoolConfig{
			MinConnections: 1,
			MaxConnections: 5,
		},
		Servers: []config.ServerConfig{
			{Name: "test", Enabled: false}, // 禁用避免真实连接
		},
	}

	srv := NewServer(cfg)
	// 注意：不调用 SetupRoutes()，因为 Start() 会调用

	// 启动服务器
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// 等待服务器启动
	time.Sleep(300 * time.Millisecond)

	// 验证服务器已启动
	if srv.server == nil {
		t.Fatal("server should be started")
	}

	// 停止服务器
	err := srv.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}

	// 等待服务器关闭
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not stop in time")
	}
}

// --- HTTP 端点测试 ---

// TestServerHealthEndpoint 测试健康检查端点
func TestServerHealthEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to request health endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("Expected status 'ok', got %q", health.Status)
	}

	if !health.Ready {
		t.Error("Expected Ready to be true")
	}
}

// TestServerHealthEndpointInitializing 测试初始化中的健康检查
func TestServerHealthEndpointInitializing(t *testing.T) {
	srv := newTestServerNotReady()

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to request health endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health.Status != "initializing" {
		t.Errorf("Expected status 'initializing', got %q", health.Status)
	}

	if health.Ready {
		t.Error("Expected Ready to be false")
	}
}

// TestServerToolsEndpoint 测试工具列表端点
func TestServerToolsEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	// 注册测试工具
	srv.registry.RegisterTool(registry.ToolInfo{
		Name:        "test-tool",
		Description: "A test tool",
		ServerName:  "test-server",
	})

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tools")
	if err != nil {
		t.Fatalf("Failed to request tools endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var tools ToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		t.Fatalf("Failed to decode tools response: %v", err)
	}

	if len(tools.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools.Tools))
	}
}

// TestServerToolsEndpointNotReady 测试工具端点在未就绪时返回 503
func TestServerToolsEndpointNotReady(t *testing.T) {
	srv := newTestServerNotReady()

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tools")
	if err != nil {
		t.Fatalf("Failed to request tools endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}
}

// TestServerToolCallEndpoint 测试工具调用端点
func TestServerToolCallEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	// 注册测试工具
	srv.registry.RegisterTool(registry.ToolInfo{
		Name:        "test-tool",
		Description: "Test tool",
		ServerName:  "test-server",
	})

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发送工具调用请求
	reqBody := ToolCallRequest{
		Name: "test-tool",
		Arguments: map[string]interface{}{
			"message": "hello",
		},
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/tools/call", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to request tool call endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 即使没有真实服务器，也应该返回 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// --- SSE 端点测试 ---

// TestSSEEndpoint 测试 SSE 端点
func TestSSEEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发起 SSE 请求
	resp, err := http.Get(ts.URL + "/sse")
	if err != nil {
		t.Fatalf("Failed to request SSE endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// SSE 应该返回成功
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// 验证 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %q", contentType)
	}
}

// TestSSEPOSTEndpoint 测试 SSE POST 端点
func TestSSEPOSTEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发送 JSON-RPC 请求
	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/sse", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to request SSE POST endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// POST 到 /sse 应该返回 200 或 202
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusAccepted, resp.StatusCode)
	}
}

// TestSSEPOSTEndpointWithSession 测试带 session 的 SSE POST
func TestSSEPOSTEndpointWithSession(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	// 先建立 SSE 会话
	ch := make(chan string, 100)
	GlobalNotifier.Add("test-sse-session", ch)
	defer GlobalNotifier.Remove("test-sse-session")

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发送 JSON-RPC 请求
	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "tools/list",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", ts.URL+"/sse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Session-ID", "test-sse-session")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to request SSE POST endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 有 session 时应该返回 Accepted
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}
}

// --- collectTools 逻辑测试 ---

// TestCollectToolsLogic 测试 collectTools 的核心逻辑
func TestCollectToolsLogic(t *testing.T) {
	// 测试 getString 辅助函数
	tools := []map[string]interface{}{
		{
			"name":        "tool1",
			"description": "First tool",
		},
		{
			"name":        "tool2",
			"description": "Second tool",
		},
	}

	// 模拟 collectTools 的数据转换逻辑
	toolInfos := make([]registry.ToolInfo, 0, len(tools))
	for _, t := range tools {
		toolInfo := registry.ToolInfo{
			Name:         getString(t, "name"),
			Description:  getString(t, "description"),
			ServerName:   "test-server",
			OriginalName: getString(t, "name"),
		}

		if inputSchema, ok := t["inputSchema"].(map[string]interface{}); ok {
			toolInfo.InputSchema = inputSchema
		}

		toolInfos = append(toolInfos, toolInfo)
	}

	if len(toolInfos) != 2 {
		t.Errorf("Expected 2 tool infos, got %d", len(toolInfos))
	}

	if toolInfos[0].Name != "tool1" {
		t.Errorf("Expected first tool name 'tool1', got %q", toolInfos[0].Name)
	}

	if toolInfos[0].OriginalName != "tool1" {
		t.Errorf("Expected first tool original name 'tool1', got %q", toolInfos[0].OriginalName)
	}

	if toolInfos[1].ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got %q", toolInfos[1].ServerName)
	}
}

// TestCollectToolsWithInputSchema 测试带 inputSchema 的工具收集
func TestCollectToolsWithInputSchema(t *testing.T) {
	tools := []map[string]interface{}{
		{
			"name":        "tool-with-schema",
			"description": "A tool with input schema",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	toolInfos := make([]registry.ToolInfo, 0, len(tools))
	for _, t := range tools {
		toolInfo := registry.ToolInfo{
			Name:         getString(t, "name"),
			Description:  getString(t, "description"),
			ServerName:   "test-server",
			OriginalName: getString(t, "name"),
		}

		if inputSchema, ok := t["inputSchema"].(map[string]interface{}); ok {
			toolInfo.InputSchema = inputSchema
		}

		toolInfos = append(toolInfos, toolInfo)
	}

	if len(toolInfos) != 1 {
		t.Errorf("Expected 1 tool info, got %d", len(toolInfos))
	}

	if toolInfos[0].InputSchema == nil {
		t.Error("Expected input schema to be set")
	}

	schema, ok := toolInfos[0].InputSchema["type"].(string)
	if !ok || schema != "object" {
		t.Errorf("Expected schema type 'object', got %v", toolInfos[0].InputSchema["type"])
	}
}

// --- initializeRuntime 路径测试 ---

// TestInitializeRuntimeDisabledServers 测试禁用服务器的处理
func TestInitializeRuntimeDisabledServers(t *testing.T) {
	cfg := &config.Config{
		Pool: &config.PoolConfig{
			MinConnections: 1,
			MaxConnections: 5,
		},
		Servers: []config.ServerConfig{
			{Name: "disabled-server", Enabled: false},
		},
	}

	srv := NewServer(cfg)

	// initializeRuntime 在 goroutine 中运行
	// 由于服务器被禁用，pool.Initialize 不会真正连接
	srv.initializeRuntime()

	// 等待初始化完成
	time.Sleep(100 * time.Millisecond)

	// 验证 ready 状态（禁用服务器不会阻止启动）
	if !srv.ready.Load() {
		t.Error("Expected server to be ready even with disabled servers")
	}
}

// TestInitializeRuntimeWithErrors 测试初始化错误处理
func TestInitializeRuntimeWithErrors(t *testing.T) {
	cfg := &config.Config{
		Pool: &config.PoolConfig{
			MinConnections: 1,
			MaxConnections: 5,
		},
		Servers: []config.ServerConfig{
			{Name: "server1", Enabled: false},
			{Name: "server2", Enabled: false},
		},
	}

	srv := NewServer(cfg)

	// 初始化应该完成（即使服务器被禁用）
	srv.initializeRuntime()
	time.Sleep(100 * time.Millisecond)

	// 验证没有初始化错误
	if err := srv.initializationError(); err != nil {
		t.Errorf("Expected no initialization error, got %v", err)
	}
}

// --- handleGracefulShutdown 路径测试 ---

// TestGracefulShutdownChannel 测试服务器停止
func TestGracefulShutdownChannel(t *testing.T) {
	cfg := newTestConfig()
	srv := NewServer(cfg)
	srv.ready.Store(true)
	// 注意：不调用 SetupRoutes()，因为 Start() 会调用

	// 启动服务器
	_ = srv.Start()
	time.Sleep(100 * time.Millisecond)

	// 验证服务器已启动
	if srv.server == nil {
		t.Fatal("server should be started")
	}

	// Stop 会触发 Shutdown
	err := srv.Stop()
	if err != nil {
		t.Logf("Stop() returned error: %v (may be expected)", err)
	}
}

// TestServerMultipleStartStops 测试多次启停
func TestServerMultipleStartStops(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建新的服务器实例进行多次测试
	for i := 0; i < 3; i++ {
		cfg := newTestConfig()
		srv := NewServer(cfg)
		srv.ready.Store(true)
		// 注意：不调用 SetupRoutes()，因为 Start() 会调用

		// 启动
		_ = srv.Start()
		time.Sleep(100 * time.Millisecond)

		// 验证已启动
		if srv.server == nil {
			t.Fatalf("iteration %d: server should be started", i)
		}

		// 停止
		err := srv.Stop()
		if err != nil {
			t.Logf("iteration %d: Stop() returned error: %v", i, err)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// --- 辅助函数测试 ---

// TestFormatInitResultsVarious 测试 formatInitResults 处理各种情况
func TestFormatInitResultsVarious(t *testing.T) {
	tests := []struct {
		name     string
		results  []initServerResult
		expected int
	}{
		{
			name:     "empty results",
			results:  []initServerResult{},
			expected: 0,
		},
		{
			name: "single success",
			results: []initServerResult{
				{ServerName: "s1", CollectedTools: 5, RegisteredTools: 3},
			},
			expected: 1,
		},
		{
			name: "mixed results",
			results: []initServerResult{
				{ServerName: "s1", CollectedTools: 5, RegisteredTools: 3},
				{ServerName: "s2", Skipped: true},
				{ServerName: "s3", Err: fmt.Errorf("failed")},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatInitResults(tt.results)
			if len(formatted) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(formatted))
			}
		})
	}
}

// TestGetStringEdgeCases 测试 getString 边界情况
func TestGetStringEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "missing",
			expected: "",
		},
		{
			name:     "nil value",
			input:    map[string]interface{}{"key": nil},
			key:      "key",
			expected: "",
		},
		{
			name:     "integer value",
			input:    map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "map value",
			input:    map[string]interface{}{"key": map[string]interface{}{}},
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

// --- Pool 与 Server 交互测试 ---

// TestPoolExecuteWithRealServer 测试在有/无真实服务器情况下的 pool.Execute
func TestPoolExecuteWithRealServer(t *testing.T) {
	// 创建没有启用任何服务器的 pool
	p := pool.NewPool(config.PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
	})

	// Initialize 应该不会失败（即使没有服务器）
	err := p.Initialize([]config.ServerConfig{})
	if err != nil {
		t.Errorf("Initialize with empty servers should not fail: %v", err)
	}

	// Execute 应该返回错误（因为没有服务器）
	_, err = p.Execute("non-existent", func(client *pool.MCPClientConnection) (interface{}, error) {
		return nil, nil
	})

	if err == nil {
		t.Error("Expected error when executing on non-existent server")
	}
}

// TestCallToolWithNoServer 测试调用不存在的服务器
func TestCallToolWithNoServer(t *testing.T) {
	cfg := newTestConfig()
	srv := NewServer(cfg)
	srv.ready.Store(true)

	// 注册工具但服务器不存在
	srv.registry.RegisterTool(registry.ToolInfo{
		Name:        "orphan-tool",
		Description: "Tool without server",
		ServerName:  "non-existent-server",
	})

	result, err := srv.callTool("orphan-tool", nil)

	// 由于服务器不存在，pool.CallTool 会返回 IsError=true 的结果
	if err != nil {
		t.Errorf("callTool should not return error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be present")
	}

	if !result.IsError {
		t.Error("Expected IsError=true when calling tool on non-existent server")
	}
}

// TestServerSetupRoutes 测试路由设置
func TestServerSetupRoutes(t *testing.T) {
	cfg := newTestConfig()
	srv := NewServer(cfg)

	// SetupRoutes 前 mux 应该是新的
	if srv.mux == nil {
		t.Error("mux should be initialized by NewServer")
	}

	// 调用 SetupRoutes 不应该 panic
	srv.SetupRoutes()

	// 验证 mux 已设置
	if srv.mux == nil {
		t.Error("mux should not be nil after SetupRoutes")
	}
}

// TestHandleMessagesEndpoint 测试消息端点
func TestHandleMessagesEndpoint(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	// 注册会话
	ch := make(chan string, 10)
	GlobalNotifier.Add("msg-test-session", ch)
	defer GlobalNotifier.Remove("msg-test-session")

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发送消息请求
	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/messages?sessionId=msg-test-session", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to request messages endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// TestHandleMessagesEndpointMissingSession 测试缺少 sessionId 的情况
func TestHandleMessagesEndpointMissingSession(t *testing.T) {
	srv := newTestServerWithRoutes(t)

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// 发送消息请求（没有 sessionId）
	reqBody := JSONRPCToolRequest{
		ID:     1,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to request messages endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
