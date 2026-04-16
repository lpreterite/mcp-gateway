package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Session 会话
type Session struct {
	ID        string
	Transport http.ResponseWriter
	Notifier  *Notifier
	Created   time.Time
}

// Notifier SSE 通知器
type Notifier struct {
	mu       sync.RWMutex
	channels map[string]chan string
}

// NewNotifier 创建新的通知器
func NewNotifier() *Notifier {
	return &Notifier{
		channels: make(map[string]chan string),
	}
}

// Add 添加通道
func (n *Notifier) Add(sessionID string, ch chan string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.channels[sessionID] = ch
}

// Remove 移除通道
func (n *Notifier) Remove(sessionID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.channels, sessionID)
}

// Notify 通知所有通道
func (n *Notifier) Notify(message string) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, ch := range n.channels {
		select {
		case ch <- message:
		default:
			// 通道已满或已关闭，跳过
		}
	}
}

// Send 发送消息到指定会话
func (n *Notifier) Send(sessionID string, message string) error {
	n.mu.RLock()
	ch, ok := n.channels[sessionID]
	n.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	select {
	case ch <- message:
		return nil
	default:
		return fmt.Errorf("failed to send to session %s", sessionID)
	}
}

// GlobalNotifier 全局通知器
var GlobalNotifier = NewNotifier()

// SSETransport SSE 传输层
type SSETransport struct {
	SessionID string
	Writer    http.ResponseWriter
	Flusher   http.Flusher
	closed    bool
	mu        sync.Mutex
}

// NewSSETransport 创建新的 SSE 传输
func NewSSETransport(sessionID string, w http.ResponseWriter) *SSETransport {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	// 设置 SSE 相关头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &SSETransport{
		SessionID: sessionID,
		Writer:    w,
		Flusher:   flusher,
	}
}

// Send 发送 SSE 消息
func (t *SSETransport) Send(event, data string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport already closed")
	}

	message := fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
	_, err := t.Writer.Write([]byte(message))
	if err != nil {
		return err
	}
	t.Flusher.Flush()
	return nil
}

// Close 关闭传输
func (t *SSETransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	GlobalNotifier.Remove(t.SessionID)
	return nil
}

// JSONRPCToolRequest JSON-RPC 工具调用请求
type JSONRPCToolRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// JSONRPCToolParams JSON-RPC 工具调用参数
type JSONRPCToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status   string                    `json:"status"`
	Sessions int                       `json:"sessions"`
	Pool     map[string]map[string]int `json:"pool"`
}

// ToolsResponse 工具列表响应
type ToolsResponse struct {
	Tools []ToolResponse `json:"tools"`
}

// ToolResponse 工具响应
type ToolResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerName  string `json:"serverName"`
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResponse 工具调用响应
type ToolCallResponse struct {
	Result *ToolCallResult `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock 内容块
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
