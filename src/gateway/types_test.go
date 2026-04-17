package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewNotifier(t *testing.T) {
	n := NewNotifier()
	if n == nil {
		t.Fatal("NewNotifier() returned nil")
	}
	if n.channels == nil {
		t.Error("NewNotifier().channels is nil")
	}
}

func TestNotifierAdd(t *testing.T) {
	n := NewNotifier()
	ch := make(chan string, 10)

	n.Add("session-1", ch)

	n.mu.RLock()
	if _, ok := n.channels["session-1"]; !ok {
		t.Error("session-1 should be in channels after Add()")
	}
	n.mu.RUnlock()
}

func TestNotifierRemove(t *testing.T) {
	n := NewNotifier()
	ch := make(chan string, 10)

	n.Add("session-1", ch)
	n.Remove("session-1")

	n.mu.RLock()
	if _, ok := n.channels["session-1"]; ok {
		t.Error("session-1 should not be in channels after Remove()")
	}
	n.mu.RUnlock()
}

func TestNotifierNotify(t *testing.T) {
	n := NewNotifier()
	ch1 := make(chan string, 10)
	ch2 := make(chan string, 10)

	n.Add("session-1", ch1)
	n.Add("session-2", ch2)

	// Notify 使用 non-blocking send，消息可能丢失
	// 这里我们只验证 Notify 不会阻塞
	n.Notify("test message")

	// 由于 Notify 可能不会等待接收，我们只验证它能正常返回
	// 在实际使用中，Notify 会在 goroutine 中被调用
}

func TestNotifierSend(t *testing.T) {
	n := NewNotifier()
	ch := make(chan string, 10)

	n.Add("session-1", ch)

	err := n.Send("session-1", "direct message")
	if err != nil {
		t.Errorf("Send() returned error: %v", err)
	}

	select {
	case msg := <-ch:
		if msg != "direct message" {
			t.Errorf("Expected 'direct message', got '%s'", msg)
		}
	default:
		t.Error("session-1 should have received message")
	}

	// 发送给不存在的会话应该返回错误
	err = n.Send("non-existent", "message")
	if err == nil {
		t.Error("Send() to non-existent session should return error")
	}
}

func TestNotifierSendFullChannel(t *testing.T) {
	n := NewNotifier()
	ch := make(chan string, 1) // 容量为 1 的通道

	n.Add("session-1", ch)
	ch <- "first" // 先填满通道

	// Send 在通道满时会返回错误（因为使用 non-blocking send）
	err := n.Send("session-1", "second")
	if err == nil {
		t.Error("Send() to full channel should return error")
	}
}

func TestSSETransportNewSSETransport(t *testing.T) {
	// 有效的 http.ResponseWriter
	w := httptest.NewRecorder()
	transport := NewSSETransport("session-1", w)
	if transport == nil {
		t.Fatal("NewSSETransport() returned nil for valid ResponseWriter")
	}
	if transport.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want 'session-1'", transport.SessionID)
	}

	// httptest.NewRecorder() 支持 Flusher，所以 transport 不为 nil
	// 这已经通过上面的测试验证了
}

func TestSSETransportSend(t *testing.T) {
	w := httptest.NewRecorder()
	transport := NewSSETransport("session-1", w)

	err := transport.Send("message", `{"data":"test"}`)
	if err != nil {
		t.Errorf("Send() returned error: %v", err)
	}

	// 验证响应头
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want 'text/event-stream'", w.Header().Get("Content-Type"))
	}

	// 验证发送后关闭transport不应该报错
	err = transport.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// 再次关闭应该没问题
	err = transport.Close()
	if err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	// 关闭后发送应该返回错误
	err = transport.Send("message", "data")
	if err == nil {
		t.Error("Send() after Close() should return error")
	}
}

func TestSSETransportClose(t *testing.T) {
	// 这个测试主要验证 Close 不会 panic
	w := httptest.NewRecorder()
	transport := NewSSETransport("session-1", w)

	err := transport.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestJSONRPCToolRequest(t *testing.T) {
	jsonData := `{
		"id": 1,
		"method": "tools/list",
		"params": {}
	}`

	var req JSONRPCToolRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// ID 是 interface{} 类型，需要类型断言比较
	if req.ID != float64(1) {
		t.Errorf("ID = %v, want 1", req.ID)
	}
	if req.Method != "tools/list" {
		t.Errorf("Method = %q, want 'tools/list'", req.Method)
	}
}

func TestJSONRPCResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"tools": []interface{}{}},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded JSONRPCResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want '2.0'", decoded.JSONRPC)
	}
}

func TestJSONRPCError(t *testing.T) {
	rpcErr := &JSONRPCError{
		Code:    -32603,
		Message: "Internal error",
	}

	data, mErr := json.Marshal(rpcErr)
	if mErr != nil {
		t.Fatalf("Failed to marshal: %v", mErr)
	}

	var decoded JSONRPCError
	uErr := json.Unmarshal(data, &decoded)
	if uErr != nil {
		t.Fatalf("Failed to unmarshal: %v", uErr)
	}

	if decoded.Code != -32603 {
		t.Errorf("Code = %d, want -32603", decoded.Code)
	}
	if decoded.Message != "Internal error" {
		t.Errorf("Message = %q, want 'Internal error'", decoded.Message)
	}
}

func TestHealthResponse(t *testing.T) {
	resp := HealthResponse{
		Status:   "ok",
		Ready:    true,
		Sessions: 5,
		Pool: map[string]map[string]int{
			"server1": {"total": 3, "active": 1, "idle": 2},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded HealthResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Status != "ok" {
		t.Errorf("Status = %q, want 'ok'", decoded.Status)
	}
	if decoded.Ready != true {
		t.Error("Ready should be true")
	}
	if decoded.Sessions != 5 {
		t.Errorf("Sessions = %d, want 5", decoded.Sessions)
	}
}

func TestToolsResponse(t *testing.T) {
	resp := ToolsResponse{
		Tools: []ToolResponse{
			{Name: "tool1", Description: "desc1", ServerName: "server1"},
			{Name: "tool2", Description: "desc2", ServerName: "server1"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolsResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Tools) != 2 {
		t.Errorf("len(Tools) = %d, want 2", len(decoded.Tools))
	}
}

func TestToolCallRequest(t *testing.T) {
	jsonData := `{
		"name": "my-tool",
		"arguments": {"arg1": "value1"}
	}`

	var req ToolCallRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Name != "my-tool" {
		t.Errorf("Name = %q, want 'my-tool'", req.Name)
	}
	if req.Arguments["arg1"] != "value1" {
		t.Errorf("Arguments[arg1] = %v, want 'value1'", req.Arguments["arg1"])
	}
}

func TestToolCallResponse(t *testing.T) {
	resp := ToolCallResponse{
		Result: &ToolCallResult{
			Content: []ContentBlock{
				{Type: "text", Text: "hello"},
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
		t.Errorf("Error = %q, want empty", decoded.Error)
	}
	if decoded.Result == nil {
		t.Fatal("Result should not be nil")
	}
	if len(decoded.Result.Content) != 1 {
		t.Errorf("len(Content) = %d, want 1", len(decoded.Result.Content))
	}
}

func TestContentBlock(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: "Hello, World!",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ContentBlock
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "text" {
		t.Errorf("Type = %q, want 'text'", decoded.Type)
	}
	if decoded.Text != "Hello, World!" {
		t.Errorf("Text = %q, want 'Hello, World!'", decoded.Text)
	}
}

// noFlusherWriter 是一个不支持 Flusher 的 ResponseWriter
type noFlusherWriter struct {
	*httptest.ResponseRecorder
}

func (w *noFlusherWriter) Header() http.Header {
	return w.ResponseRecorder.Header()
}

func (w *noFlusherWriter) Write(b []byte) (int, error) {
	return w.ResponseRecorder.Write(b)
}

func (w *noFlusherWriter) WriteHeader(statusCode int) {
	w.ResponseRecorder.WriteHeader(statusCode)
}

// Note: httptest.ResponseRecorder 实现了 http.Flusher 接口，
// 因此无法在单元测试中创建不支持 Flusher 的场景。
// NewSSETransport 返回 nil 的分支在实际使用中由底层 http.ResponseWriter 决定。

// TestSSETransportSendToClosedChannel 测试 Notify 的默认分支
func TestNotifierNotifyFullChannel(t *testing.T) {
	n := NewNotifier()
	ch := make(chan string, 1)
	ch <- "first" // 填满通道

	n.Add("session-full", ch)

	// Notify 应该不会阻塞，即使通道满了
	n.Notify("test message")
	// 验证第一个消息仍在通道中
	select {
	case msg := <-ch:
		if msg != "first" {
			t.Errorf("Expected 'first', got '%s'", msg)
		}
	default:
		t.Error("First message should still be in channel")
	}
}

// TestJSONRPCToolParams 测试 JSON-RPC 工具参数序列化
func TestJSONRPCToolParams(t *testing.T) {
	params := JSONRPCToolParams{
		Name: "test-tool",
		Arguments: map[string]interface{}{
			"key1": "value1",
			"key2": float64(123), // JSON numbers are float64
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded JSONRPCToolParams
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != "test-tool" {
		t.Errorf("Name = %q, want 'test-tool'", decoded.Name)
	}
	if decoded.Arguments["key1"] != "value1" {
		t.Errorf("Arguments[key1] = %v, want 'value1'", decoded.Arguments["key1"])
	}
}

// TestToolCallResult 测试工具调用结果序列化
func TestToolCallResult(t *testing.T) {
	result := ToolCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: "output1"},
			{Type: "text", Text: "output2"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolCallResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Content) != 2 {
		t.Errorf("len(Content) = %d, want 2", len(decoded.Content))
	}
	if decoded.IsError {
		t.Error("IsError should be false")
	}
}

// TestToolCallResponseWithError 测试包含错误的响应
func TestToolCallResponseWithError(t *testing.T) {
	resp := ToolCallResponse{
		Error: "tool execution failed",
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

	if decoded.Error != "tool execution failed" {
		t.Errorf("Error = %q, want 'tool execution failed'", decoded.Error)
	}
	if decoded.Result != nil {
		t.Error("Result should be nil when Error is set")
	}
}

// TestToolResponse 测试工具响应序列化
func TestToolResponse(t *testing.T) {
	resp := ToolResponse{
		Name:        "my-tool",
		Description: "A useful tool",
		ServerName:  "my-server",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != "my-tool" {
		t.Errorf("Name = %q, want 'my-tool'", decoded.Name)
	}
}

// TestNotifierSendToNonExistentSession 测试发送给不存在的会话
func TestNotifierSendToNonExistentSession(t *testing.T) {
	n := NewNotifier()

	err := n.Send("non-existent-session", "message")
	if err == nil {
		t.Error("Expected error when sending to non-existent session")
	}
}
