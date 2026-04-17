package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/pool"
	"github.com/lpreterite/mcp-gateway/src/registry"
)

// initContext holds the results of tool collection and synchronization primitives
type initContext struct {
	results   []initServerResult
	resultsMu sync.Mutex
	wg        sync.WaitGroup
}

// Server MCP 网关服务器
type Server struct {
	config   *config.Config
	pool     *pool.Pool
	registry *registry.Registry
	mapper   *registry.Mapper
	mux      *http.ServeMux
	server   *http.Server
	ready    atomic.Bool
	initErr  atomic.Value
	initOnce sync.Once
}

type initServerResult struct {
	ServerName      string
	CollectedTools  int
	RegisteredTools int
	Skipped         bool
	Err             error
}

// NewServer 创建新的网关服务器
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:   cfg,
		pool:     pool.NewPool(*cfg.Pool),
		registry: registry.NewRegistry(),
		mapper:   registry.NewMapper(cfg.Mapping, cfg.ToolFilters),
		mux:      http.NewServeMux(),
	}
}

func (s *Server) initializeRuntime() {
	s.initOnce.Do(func() {
		go func() {
			slog.Info("Initializing gateway runtime in background")

			if err := s.pool.Initialize(s.config.Servers); err != nil {
				s.initErr.Store(err)
				slog.Error("Failed to initialize pool", "error", err)
				return
			}

			ctx := &initContext{
				results: make([]initServerResult, 0, len(s.config.Servers)),
			}

			for _, serverConfig := range s.config.Servers {
				if !serverConfig.Enabled {
					ctx.resultsMu.Lock()
					ctx.results = append(ctx.results, initServerResult{
						ServerName: serverConfig.Name,
						Skipped:    true,
					})
					ctx.resultsMu.Unlock()
					continue
				}

				ctx.wg.Add(1)
				go func(cfg config.ServerConfig) {
					defer ctx.wg.Done()

					result := initServerResult{ServerName: cfg.Name}
					tools, err := collectTools(s.pool, cfg.Name)
					if err != nil {
						result.Err = err
						ctx.resultsMu.Lock()
						ctx.results = append(ctx.results, result)
						ctx.resultsMu.Unlock()
						slog.Warn("Failed to collect tools",
							"server", cfg.Name,
							"error", err,
						)
						return
					}
					result.CollectedTools = len(tools)

					for _, tool := range tools {
						if !s.mapper.ShouldIncludeTool(cfg.Name, tool.OriginalName) {
							continue
						}

						gatewayName := s.mapper.GetGatewayToolName(tool.OriginalName, cfg.Name)
						tool.Name = gatewayName
						s.registry.RegisterTool(tool)
						result.RegisteredTools++
					}

					ctx.resultsMu.Lock()
					ctx.results = append(ctx.results, result)
					ctx.resultsMu.Unlock()
					slog.Info("Server tool collection completed",
						"server", result.ServerName,
						"collected", result.CollectedTools,
						"registered", result.RegisteredTools,
					)
				}(serverConfig)
			}

			ctx.wg.Wait()

			s.ready.Store(true)
			slog.Info("Gateway runtime ready",
				"servers", formatInitResults(ctx.results),
				"totalTools", s.registry.Count(),
			)
		}()
	})
}

func (s *Server) isReady() bool {
	return s.ready.Load()
}

func (s *Server) initializationError() error {
	if v := s.initErr.Load(); v != nil {
		if err, ok := v.(error); ok {
			return err
		}
	}
	return nil
}

func formatInitResults(results []initServerResult) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		entry := map[string]interface{}{
			"server":     result.ServerName,
			"skipped":    result.Skipped,
			"collected":  result.CollectedTools,
			"registered": result.RegisteredTools,
		}
		if result.Err != nil {
			entry["error"] = result.Err.Error()
		}
		formatted = append(formatted, entry)
	}
	return formatted
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
	// SSE 端点（支持 GET 建立连接，也支持 POST 发送 JSON-RPC）
	s.mux.HandleFunc("GET /sse", s.handleSSE)
	s.mux.HandleFunc("POST /sse", s.handleSSEPOST)

	// 消息端点
	s.mux.HandleFunc("POST /messages", s.handleMessages)

	// 工具端点
	s.mux.HandleFunc("GET /tools", s.handleListTools)
	s.mux.HandleFunc("POST /tools/call", s.handleToolCall)

	// 健康检查
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

// handleSSEPOST 处理 POST 到 /sse 的 JSON-RPC 请求
func (s *Server) handleSSEPOST(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("MCP-Session-ID")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("sessionId")
	}

	// 尝试找到对应的 SSE 会话
	var ch chan string
	if sessionID != "" {
		GlobalNotifier.mu.RLock()
		existingCh, ok := GlobalNotifier.channels[sessionID]
		if ok {
			ch = existingCh
		}
		GlobalNotifier.mu.RUnlock()
	}

	// 如果没有 session 或会话不存在，尝试找一个活跃 SSE 会话
	if ch == nil {
		GlobalNotifier.mu.RLock()
		for sessID, existingCh := range GlobalNotifier.channels {
			if strings.HasPrefix(sessID, "sse-") {
				sessionID = sessID
				ch = existingCh
				break
			}
		}
		GlobalNotifier.mu.RUnlock()
	}

	var req JSONRPCToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	slog.Info("Received SSE JSON-RPC request",
		"session", sessionID,
		"method", req.Method,
	)

	resp := s.processJSONRPCRequest(req)

	respData, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal response", "error", err)
		return
	}

	// 如果有 SSE 通道，通过它发送；否则直接返回
	if ch != nil {
		select {
		case ch <- string(respData):
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	} else {
		// 没有 SSE 通道，直接返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respData)
	}
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
	defer func() { _ = transport.Close() }()

	slog.Info("SSE connection established",
		"session", sessionID,
		"ip", r.RemoteAddr,
	)

	// 发送初始连接消息
	if err := transport.Send("connected", fmt.Sprintf(`{"sessionId":"%s"}`, sessionID)); err != nil {
		slog.Error("Failed to send initial SSE message", "error", err)
		return
	}

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

// processJSONRPCRequest 处理 JSON-RPC 请求
func (s *Server) processJSONRPCRequest(req JSONRPCToolRequest) JSONRPCResponse {
	var resp JSONRPCResponse
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": struct{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp-gateway",
				"version": "1.0.0",
			},
		}

	case "tools/list":
		if !s.isReady() {
			resp.Error = &JSONRPCError{
				Code:    -32000,
				Message: "gateway is still initializing",
			}
			break
		}

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
		if !s.isReady() {
			resp.Error = &JSONRPCError{
				Code:    -32000,
				Message: "gateway is still initializing",
			}
			break
		}

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

	return resp
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

	resp := s.processJSONRPCRequest(req)

	// 发送响应
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// callTool 调用工具
func (s *Server) callTool(name string, args map[string]interface{}) (*pool.ToolCallResult, error) {
	if !s.isReady() {
		return nil, fmt.Errorf("gateway is still initializing")
	}

	tool, ok := s.registry.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	serverName := tool.ServerName
	originalName := s.mapper.GetOriginalToolName(name, serverName)
	if originalName == "" {
		originalName = tool.OriginalName
	}

	slog.Info("Calling tool",
		"tool", name,
		"originalName", originalName,
		"server", serverName,
	)

	result, err := s.pool.CallTool(serverName, originalName, args)
	if err != nil {
		slog.Error("Tool call failed",
			"tool", name,
			"server", serverName,
			"error", err,
		)
		return nil, err
	}

	slog.Info("Tool call succeeded",
		"tool", name,
		"server", serverName,
		"isError", result.IsError,
	)

	return result, nil
}

// handleListTools 处理工具列表请求
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	if !s.isReady() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(ToolsResponse{Tools: []ToolResponse{}}); err != nil {
			slog.Error("Failed to encode initializing tools response", "error", err)
		}
		return
	}

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
	if err := json.NewEncoder(w).Encode(ToolsResponse{Tools: responses}); err != nil {
		slog.Error("Failed to encode tools response", "error", err)
	}
}

// handleToolCall 处理工具调用请求
func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if !s.isReady() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(ToolCallResponse{Error: "gateway is still initializing"}); err != nil {
			slog.Error("Failed to encode initializing tool call response", "error", err)
		}
		return
	}

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
		if err := json.NewEncoder(w).Encode(ToolCallResponse{
			Error: err.Error(),
		}); err != nil {
			slog.Error("Failed to encode tool call error response", "error", err)
		}
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
	if err := json.NewEncoder(w).Encode(ToolCallResponse{Result: gatewayResult}); err != nil {
		slog.Error("Failed to encode tool call success response", "error", err)
	}
}

// handleHealth 处理健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	GlobalNotifier.mu.RLock()
	sessionCount := len(GlobalNotifier.channels)
	GlobalNotifier.mu.RUnlock()

	stats := s.pool.GetStats()
	status := "ok"
	if !s.isReady() {
		status = "initializing"
	}
	if err := s.initializationError(); err != nil {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(HealthResponse{
		Status:   status,
		Ready:    s.isReady(),
		Sessions: sessionCount,
		Pool:     stats,
	}); err != nil {
		slog.Error("Failed to encode health response", "error", err)
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.SetupRoutes()
	s.initializeRuntime()

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

	if err := s.pool.DisconnectAll(); err != nil {
		slog.Error("Pool disconnect error", "error", err)
	}

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
