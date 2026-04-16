package registry

// ToolInfo 工具信息
type ToolInfo struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ServerName   string                 `json:"serverName"`
	OriginalName string                 `json:"originalName"`
	InputSchema  map[string]interface{} `json:"inputSchema,omitempty"`
	Annotations  map[string]interface{} `json:"annotations,omitempty"`
}

// Registry 工具注册表
type Registry struct {
	tools map[string]ToolInfo
}

// NewRegistry 创建新的注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolInfo),
	}
}

// RegisterTool 注册工具
func (r *Registry) RegisterTool(tool ToolInfo) {
	r.tools[tool.Name] = tool
}

// UnregisterTool 注销工具
func (r *Registry) UnregisterTool(name string) {
	delete(r.tools, name)
}

// GetTool 获取工具
func (r *Registry) GetTool(name string) (ToolInfo, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetAllTools 获取所有工具
func (r *Registry) GetAllTools() []ToolInfo {
	tools := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolsByServer 获取指定服务器的工具
func (r *Registry) GetToolsByServer(serverName string) []ToolInfo {
	var tools []ToolInfo
	for _, tool := range r.tools {
		if tool.ServerName == serverName {
			tools = append(tools, tool)
		}
	}
	return tools
}

// HasTool 检查工具是否存在
func (r *Registry) HasTool(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// Clear 清空注册表
func (r *Registry) Clear() {
	r.tools = make(map[string]ToolInfo)
}

// ClearByServer 清空指定服务器的工具
func (r *Registry) ClearByServer(serverName string) {
	for name, tool := range r.tools {
		if tool.ServerName == serverName {
			delete(r.tools, name)
		}
	}
}

// Count 返回工具数量
func (r *Registry) Count() int {
	return len(r.tools)
}
