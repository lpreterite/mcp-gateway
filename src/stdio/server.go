package stdio

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/packy/mcp-gateway/src/config"
	"github.com/packy/mcp-gateway/src/pool"
	"github.com/packy/mcp-gateway/src/registry"
)

// Server stdio 模式服务器
type Server struct {
	config   *config.Config
	pool     *pool.Pool
	registry *registry.Registry
	mapper   *registry.Mapper
	bridge   *Bridge
}

// NewServer 创建新的 stdio 服务器
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

	slog.Info("Tool registry initialized (stdio mode)",
		"total", r.Count(),
	)

	return &Server{
		config:   cfg,
		pool:     p,
		registry: r,
		mapper:   m,
		bridge:   NewBridge(),
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

// Start 启动 stdio 服务器
func (s *Server) Start() error {
	// 注册工具列表处理器
	s.bridge.RegisterHandler("tools/list", s.handleListTools)

	// 注册工具调用处理器
	s.bridge.RegisterHandler("tools/call", s.handleToolCall)

	// 注册初始化处理器 (MCP 协议)
	s.bridge.RegisterHandler("initialize", s.handleInitialize)

	// 发送初始化通知
	s.sendNotification("initialized", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": struct{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcp-gateway",
			"version": "1.0.0",
		},
	})

	// 启动桥接器
	return s.bridge.Start()
}

// handleInitialize 处理初始化请求
func (s *Server) handleInitialize(method string, params interface{}) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": struct{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcp-gateway",
			"version": "1.0.0",
		},
	}, nil
}

// handleListTools 处理工具列表请求
func (s *Server) handleListTools(method string, params interface{}) (interface{}, error) {
	tools := s.registry.GetAllTools()
	toolResponses := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		toolResponses = append(toolResponses, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return map[string]interface{}{"tools": toolResponses}, nil
}

// handleToolCall 处理工具调用请求
func (s *Server) handleToolCall(method string, params interface{}) (interface{}, error) {
	// 解析 params
	var req struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 查找工具
	tool, ok := s.registry.GetTool(req.Name)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", req.Name)
	}

	// 获取原始工具名
	serverName := tool.ServerName
	originalName := s.mapper.GetOriginalToolName(req.Name, serverName)
	if originalName == "" {
		originalName = tool.OriginalName
	}

	// 调用工具
	result, err := s.pool.CallTool(serverName, originalName, req.Arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// sendNotification 发送通知
func (s *Server) sendNotification(method string, params interface{}) {
	if err := s.bridge.SendNotification(method, params); err != nil {
		slog.Warn("Failed to send notification", "method", method, "error", err)
	}
}

// Stop 停止服务器
func (s *Server) Stop() {
	s.pool.DisconnectAll()
}

// IsConnected 检查连接状态
func (s *Server) IsConnected() bool {
	return s.bridge != nil && s.bridge.IsConnected()
}
