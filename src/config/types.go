package config

// ServerConfig MCP 服务器配置
type ServerConfig struct {
	Name     string            `json:"name" mapstructure:"name"`
	Type     string            `json:"type" mapstructure:"type"` // "local" 或 "remote"
	Command  []string          `json:"command,omitempty" mapstructure:"command"`
	URL      string            `json:"url,omitempty" mapstructure:"url"`
	Enabled  bool              `json:"enabled" mapstructure:"enabled"`
	Env      map[string]string `json:"env,omitempty" mapstructure:"env"`
	PoolSize int               `json:"poolSize,omitempty" mapstructure:"poolSize"`
}

// MappingConfig 工具名映射配置
type MappingConfig struct {
	Prefix      string            `json:"prefix" mapstructure:"prefix"`
	StripPrefix bool              `json:"stripPrefix" mapstructure:"stripPrefix"`
	Rename      map[string]string `json:"rename,omitempty" mapstructure:"rename"`
}

// ToolFilterConfig 工具过滤器配置
type ToolFilterConfig struct {
	Include []string `json:"include,omitempty" mapstructure:"include"`
	Exclude []string `json:"exclude,omitempty" mapstructure:"exclude"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MinConnections int `json:"minConnections" mapstructure:"minConnections"`
	MaxConnections int `json:"maxConnections" mapstructure:"maxConnections"`
	AcquireTimeout int `json:"acquireTimeout" mapstructure:"acquireTimeout"`
	IdleTimeout    int `json:"idleTimeout" mapstructure:"idleTimeout"`
	MaxRetries     int `json:"maxRetries" mapstructure:"maxRetries"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Host string `json:"host" mapstructure:"host"`
	Port int    `json:"port" mapstructure:"port"`
	CORS bool   `json:"cors" mapstructure:"cors"`
}

// Config 完整配置
type Config struct {
	Gateway     *GatewayConfig              `json:"gateway,omitempty" mapstructure:"gateway"`
	Pool        *PoolConfig                 `json:"pool,omitempty" mapstructure:"pool"`
	Servers     []ServerConfig              `json:"servers" mapstructure:"servers"`
	Mapping     map[string]MappingConfig    `json:"mapping,omitempty" mapstructure:"mapping"`
	ToolFilters map[string]ToolFilterConfig `json:"toolFilters,omitempty" mapstructure:"toolFilters"`
}
