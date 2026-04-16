package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/packy/mcp-gateway/internal/config"
	"github.com/packy/mcp-gateway/internal/pool"
	"github.com/packy/mcp-gateway/internal/registry"
)

// Server MCP 网关服务器
type Server struct {
	config   *config.Config
	pool     *pool.Pool
	registry *registry.Registry
	mapper   *registry.Mapper
	mux      *http.ServeMux
	server   *http.Server
}

// NewServer 创建新的网关服务器
func NewServer(cfg *config.Config) *Server {
	// 初始化连接池
	p := pool.NewPool(*cfg.Pool)
	if err := p.Initialize(cfg.Servers); err != nil {
		slog.Error("Failed to initialize pool", "error", err)
	}

	// 初始化注册表
	r := registry.NewRegistry()

	// 初始化映射器
	m := registry.NewMapper(cfg.Mapping, cfg.ToolFilters)

	// 收集所有工具
	for _, serverConfig := range cfg.Servers {
		if !serverConfig.Enabled {
			continue
		}

		// 获取该服务器的工具
		tools, err := collectTools(p, serverConfig.Name)
		if err != nil {
			slog.Warn("Failed to collect tools",
				"server", serverConfig.Name,
				"error", err,
			)
			continue
		}

		for _, tool := range tools {
			// 应用过滤器
			if !m.ShouldIncludeTool(serverConfig.Name, tool.OriginalName) {
				continue
			}

			// 添加前缀
			gatewayName := m.GetGatewayToolName(tool.OriginalName, serverConfig.Name)
			tool.Name = gatewayName
			r.RegisterTool(tool)
		}
	}

	slog.Info("Tool registry initialized",
		"total", r.Count(),
	)

	return &Server{
		config:   cfg,
		pool:     p,
		registry: r,
		mapper:   m,
		mux:      http.NewServeMux(),
	}
}

// collectTools 收集指定服务器的 tools
func collectTools(p *pool.Pool, serverName string) ([]registry.ToolInfo, error) {
	result, err := p.Execute(serverName, func(client *pool.MCPClientConnection) (interface{}, error) {
		tools, err := client.ListTools()
		if err != nil {
			return nil, err
		}

		toolInfos := make([]registry.ToolInfo, 0, len(tools))
		for _, t := range tools {
			toolInfo := registry.ToolInfo{
				Name:         getString(t, "name"),
				Description:  getString(t, "description"),
				ServerName:   serverName,
				OriginalName: getString(t, "name"),
			}

			if inputSchema, ok := t["inputSchema"].(map[string]interface{}); ok {
				toolInfo.InputSchema = inputSchema
			}

			toolInfos = append(toolInfos, toolInfo)
		}

		return toolInfos, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]registry.ToolInfo), nil
}

// getString 从 map 获取字符串
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetupRoutes 设置路由
func (s *Server) SetupRoutes() {
	// SSE 端点
	s.mux.HandleFunc("GET /sse", s.handleSSE)

	// 消息端点
	s.mux.HandleFunc("POST /messages", s.handleMessages)

	// 工具端点
	s.mux.HandleFunc("GET /tools", s.handleListTools)
	s.mux.HandleFunc("POST /tools/call", s.handleToolCall)

	// 健康检查
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

// handleSSE 处理 SSE 连接
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 生成会话 ID
	sessionID := fmt.Sprintf("sse-%d", time.Now().UnixNano())

	// 创建 SSE 传输
	transport := NewSSETransport(sessionID, w)
	if transport == nil {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// 注册到通知器
	ch := make(chan string, 100)
	GlobalNotifier.Add(sessionID, ch)
	defer transport.Close()

	slog.Info("SSE connection established",
		"session", sessionID,
		"ip", r.RemoteAddr,
	)

	// 发送初始连接消息
	transport.Send("connected", fmt.Sprintf(`{"sessionId":"%s"}`, sessionID))

	// 保持连接直到关闭
	clientGone := r.Context().Done()
	for {
		select {
		case <-clientGone:
			slog.Info("SSE client disconnected", "session", sessionID)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := transport.Send("message", msg); err != nil {
				slog.Warn("Failed to send SSE message",
					"session", sessionID,
					"error", err,
				)
				return
			}
		}
	}
}

// handleMessages 处理 JSON-RPC 消息
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	// 检查会话是否存在
	GlobalNotifier.mu.RLock()
	_, ok := GlobalNotifier.channels[sessionID]
	GlobalNotifier.mu.RUnlock()

	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// 解析请求
	var req JSONRPCToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	slog.Info("Received JSON-RPC request",
		"session", sessionID,
		"method", req.Method,
	)

	// 处理请求
	var resp JSONRPCResponse
	resp.JSONRPC = "2.0"

	switch req.Method {
	case "tools/list":
		tools := s.registry.GetAllTools()
		toolResponses := make([]map[string]interface{}, 0, len(tools))
		for _, t := range tools {
			toolResponses = append(toolResponses, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.InputSchema,
				"annotations": t.Annotations,
			})
		}
		resp.Result = map[string]interface{}{"tools": toolResponses}

	case "tools/call":
		var params JSONRPCToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &JSONRPCError{
				Code:    -32700,
				Message: fmt.Sprintf("Invalid params: %v", err),
			}
		} else {
			result, err := s.callTool(params.Name, params.Arguments)
			if err != nil {
				resp.Error = &JSONRPCError{
					Code:    -32603,
					Message: err.Error(),
				}
			} else {
				resp.Result = result
			}
		}

	default:
		resp.Error = &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	// 发送响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// callTool 调用工具
func (s *Server) callTool(name string, args map[string]interface{}) (*pool.ToolCallResult, error) {
	tool, ok := s.registry.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	serverName := tool.ServerName
	originalName := s.mapper.GetOriginalToolName(name, serverName)
	if originalName == "" {
		originalName = tool.OriginalName
	}

	result, err := s.pool.CallTool(serverName, originalName, args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleListTools 处理工具列表请求
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.registry.GetAllTools()
	responses := make([]ToolResponse, 0, len(tools))
	for _, t := range tools {
		responses = append(responses, ToolResponse{
			Name:        t.Name,
			Description: t.Description,
			ServerName:  t.ServerName,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ToolsResponse{Tools: responses})
}

// handleToolCall 处理工具调用请求
func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	result, err := s.callTool(req.Name, req.Arguments)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ToolCallResponse{
			Error: err.Error(),
		})
		return
	}

	// 转换为 gateway.ToolCallResult
	gatewayResult := &ToolCallResult{
		Content: make([]ContentBlock, len(result.Content)),
		IsError: result.IsError,
	}
	for i, c := range result.Content {
		gatewayResult.Content[i] = ContentBlock{
			Type: c["type"].(string),
			Text: c["text"].(string),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ToolCallResponse{Result: gatewayResult})
}

// handleHealth 处理健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	GlobalNotifier.mu.RLock()
	sessionCount := len(GlobalNotifier.channels)
	GlobalNotifier.mu.RUnlock()

	stats := s.pool.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:   "ok",
		Sessions: sessionCount,
		Pool:     stats,
	})
}

// Start 启动服务器
func (s *Server) Start() error {
	s.SetupRoutes()

	host := "0.0.0.0"
	port := 4298

	if s.config.Gateway != nil {
		if s.config.Gateway.Host != "" {
			host = s.config.Gateway.Host
		}
		if s.config.Gateway.Port != 0 {
			port = s.config.Gateway.Port
		}
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	// 启动优雅关闭
	go s.handleGracefulShutdown()

	slog.Info("MCP Gateway starting...",
		"host", host,
		"port", port,
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// handleGracefulShutdown 处理优雅关闭
func (s *Server) handleGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	slog.Info("Received signal, initiating graceful shutdown", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}

	s.pool.DisconnectAll()

	slog.Info("Graceful shutdown completed")
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}
