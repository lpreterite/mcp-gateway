package stdio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Bridge stdio 桥接器
type Bridge struct {
	mu        sync.Mutex
	connected bool
	writer    *bufio.Writer
	reader    *bufio.Reader
	handlers  map[string]Handler
}

// Handler 消息处理函数
type Handler func(method string, params interface{}) (interface{}, error)

// NewBridge 创建新的 stdio 桥接器
func NewBridge() *Bridge {
	return &Bridge{
		writer:   bufio.NewWriter(os.Stdout),
		reader:   bufio.NewReader(os.Stdin),
		handlers: make(map[string]Handler),
	}
}

// RegisterHandler 注册消息处理器
func (b *Bridge) RegisterHandler(method string, handler Handler) {
	b.handlers[method] = handler
}

// Start 开始处理 stdio 输入
func (b *Bridge) Start() error {
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()

	slog.Info("Starting stdio bridge")

	// 使用 scanner 逐行读取 JSON-RPC 请求
	scanner := bufio.NewScanner(b.reader)
	// 增大 scanner 的缓冲区，以处理更大的 JSON 消息
	const maxScanTokenSize = 1024 * 1024 // 1MB
	scanner.Buffer(make([]byte, 4096), maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			slog.Warn("Failed to decode request", "error", err, "line", string(line))
			continue
		}

		b.handleRequest(&req)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Stdin scanner error", "error", err)
	}

	return nil
}

// handleRequest 处理请求
func (b *Bridge) handleRequest(req *JSONRPCRequest) {
	b.mu.Lock()
	handler, ok := b.handlers[req.Method]
	b.mu.Unlock()

	var resp JSONRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	if !ok {
		resp.Error = &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	} else {
		result, err := handler(req.Method, req.Params)
		if err != nil {
			resp.Error = &JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			}
		} else {
			resp.Result = result
		}
	}

	// 发送响应
	if err := json.NewEncoder(b.writer).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}

	// 确保输出被刷新
	if err := b.writer.Flush(); err != nil {
		slog.Error("Failed to flush writer", "error", err)
	}
}

// SendNotification 发送通知
func (b *Bridge) SendNotification(method string, params interface{}) error {
	notification := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if err := json.NewEncoder(b.writer).Encode(notification); err != nil {
		return err
	}

	return b.writer.Flush()
}

// Close 关闭桥接器
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connected = false
}

// IsConnected 检查是否已连接
func (b *Bridge) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCNotification JSON-RPC 通知
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
