package config

// ServerConfig MCP 服务器配置
type ServerConfig struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"` // "local" 或 "remote"
	Command  []string          `json:"command,omitempty"`
	URL      string            `json:"url,omitempty"`
	Enabled  bool              `json:"enabled"`
	Env      map[string]string `json:"env,omitempty"`
	PoolSize int               `json:"poolSize,omitempty"`
}

// MappingConfig 工具名映射配置
type MappingConfig struct {
	Prefix      string            `json:"prefix"`
	StripPrefix bool              `json:"stripPrefix"`
	Rename      map[string]string `json:"rename,omitempty"`
}

// ToolFilterConfig 工具过滤器配置
type ToolFilterConfig struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MinConnections int `json:"minConnections"`
	MaxConnections int `json:"maxConnections"`
	AcquireTimeout int `json:"acquireTimeout"`
	IdleTimeout    int `json:"idleTimeout"`
	MaxRetries     int `json:"maxRetries"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	CORS bool   `json:"cors"`
}

// Config 完整配置
type Config struct {
	Gateway     *GatewayConfig              `json:"gateway,omitempty"`
	Pool        *PoolConfig                 `json:"pool,omitempty"`
	Servers     []ServerConfig              `json:"servers"`
	Mapping     map[string]MappingConfig    `json:"mapping,omitempty"`
	ToolFilters map[string]ToolFilterConfig `json:"toolFilters,omitempty"`
}
