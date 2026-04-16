package pool

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/lpreterite/mcp-gateway/src/config"
)

// MCPClientConnection MCP 客户端连接
type MCPClientConnection struct {
	config    config.ServerConfig
	cmd       *exec.Cmd
	stdin     *os.File
	stdout    *os.File
	connected bool
	lastUsed  time.Time
	mu        sync.Mutex

	// 用于 JSON-RPC 通信
	requestID int
	requestMu sync.Mutex
	pending   map[int]chan *json.RawMessage
	pendingMu sync.Mutex
}

// NewMCPClientConnection 创建新的 MCP 客户端连接
func NewMCPClientConnection(cfg config.ServerConfig) *MCPClientConnection {
	return &MCPClientConnection{
		config:   cfg,
		lastUsed: time.Now(),
		pending:  make(map[int]chan *json.RawMessage),
	}
}

// Connect 建立连接
func (c *MCPClientConnection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	if c.config.Command == nil || len(c.config.Command) == 0 {
		return fmt.Errorf("no command configured for server %s", c.config.Name)
	}

	// 构建环境变量
	env := os.Environ()
	if c.config.Env != nil {
		for k, v := range c.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 创建子进程
	cmdPath := c.config.Command[0]
	cmdArgs := c.config.Command[1:]
	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Env = env

	// 获取 stdin/stdout pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	c.stdin = stdin.(*os.File)
	c.stdout = stdout.(*os.File)
	c.cmd = cmd

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	c.connected = true
	c.lastUsed = time.Now()

	slog.Info("MCP client connected",
		"server", c.config.Name,
		"command", c.config.Command,
	)

	// 启动读取协程
	go c.readResponses()

	return nil
}

// readResponses 异步读取响应
func (c *MCPClientConnection) readResponses() {
	decoder := json.NewDecoder(c.stdout)
	for decoder.More() {
		var response json.RawMessage
		if err := decoder.Decode(&response); err != nil {
			slog.Error("Failed to decode response",
				"server", c.config.Name,
				"error", err,
			)
			break
		}

		// 解析响应，查找对应的 pending channel
		c.handleResponse(&response)
	}

	// 进程已结束
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	slog.Info("MCP client disconnected", "server", c.config.Name)
}

// handleResponse 处理响应
func (c *MCPClientConnection) handleResponse(response *json.RawMessage) {
	// 解析为 JSON-RPC 响应
	var rpcResp struct {
		ID     int              `json:"id"`
		Result *json.RawMessage `json:"result,omitempty"`
		Error  *json.RawMessage `json:"error,omitempty"`
	}

	if err := json.Unmarshal(*response, &rpcResp); err != nil {
		slog.Warn("Invalid JSON-RPC response",
			"server", c.config.Name,
			"error", err,
		)
		return
	}

	// 查找对应的 pending channel
	c.pendingMu.Lock()
	ch, ok := c.pending[rpcResp.ID]
	if ok {
		delete(c.pending, rpcResp.ID)
	}
	c.pendingMu.Unlock()

	if ok && ch != nil {
		if rpcResp.Result != nil {
			ch <- rpcResp.Result
		} else if rpcResp.Error != nil {
			// 返回错误
			ch <- rpcResp.Error
		}
	}
}

// sendRequest 发送 JSON-RPC 请求
func (c *MCPClientConnection) sendRequest(method string, params interface{}) (*json.RawMessage, error) {
	c.requestMu.Lock()
	id := c.requestID
	c.requestID++
	c.pendingMu.Lock()
	ch := make(chan *json.RawMessage, 1)
	c.pending[id] = ch
	c.pendingMu.Unlock()
	c.requestMu.Unlock()

	// 构建请求
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	// 发送请求
	data, err := json.Marshal(req)
	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 添加换行符（JSON-RPC 协议要求）
	data = append(data, '\n')

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	_, err = c.stdin.Write(data)
	c.mu.Unlock()

	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// 等待响应（带超时）
	select {
	case result := <-ch:
		return result, nil
	case <-time.After(30 * time.Second):
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// ListTools 获取工具列表
func (c *MCPClientConnection) ListTools() ([]map[string]interface{}, error) {
	result, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, err
	}

	// 解析结果
	var resp struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.Unmarshal(*result, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	return resp.Tools, nil
}

// CallTool 调用工具
func (c *MCPClientConnection) CallTool(name string, args map[string]interface{}) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.sendRequest("tools/call", params)
	if err != nil {
		return nil, err
	}

	// 解析结果
	var toolResult map[string]interface{}
	if err := json.Unmarshal(*result, &toolResult); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	c.lastUsed = time.Now()
	return toolResult, nil
}

// Disconnect 断开连接
func (c *MCPClientConnection) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// 关闭 stdin，这通常会导致进程退出
	if c.stdin != nil {
		c.stdin.Close()
	}

	// 等待进程结束
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Wait()
	}

	c.connected = false
	slog.Info("MCP client disconnected", "server", c.config.Name)
	return nil
}

// IsConnected 检查是否已连接
func (c *MCPClientConnection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// GetName 获取服务器名称
func (c *MCPClientConnection) GetName() string {
	return c.config.Name
}

// Touch 更新最后使用时间
func (c *MCPClientConnection) Touch() {
	c.lastUsed = time.Now()
}

// GetLastUsed 获取最后使用时间
func (c *MCPClientConnection) GetLastUsed() time.Time {
	return c.lastUsed
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []map[string]interface{} `json:"content"`
	IsError bool                     `json:"isError,omitempty"`
}

// Pool 连接池
type Pool struct {
	config        config.PoolConfig
	pools         map[string][]*MCPClientConnection        // serverName -> 连接池
	active        map[string]map[*MCPClientConnection]bool // serverName -> 活跃连接集合
	serverConfigs map[string]config.ServerConfig
	mu            sync.RWMutex
}

// NewPool 创建连接池
func NewPool(cfg config.PoolConfig) *Pool {
	return &Pool{
		config:        cfg,
		pools:         make(map[string][]*MCPClientConnection),
		active:        make(map[string]map[*MCPClientConnection]bool),
		serverConfigs: make(map[string]config.ServerConfig),
	}
}

// Initialize 初始化连接池
func (p *Pool) Initialize(servers []config.ServerConfig) error {
	for _, server := range servers {
		if !server.Enabled {
			continue
		}

		p.serverConfigs[server.Name] = server
		p.pools[server.Name] = []*MCPClientConnection{}
		p.active[server.Name] = make(map[*MCPClientConnection]bool)

		poolSize := server.PoolSize
		if poolSize == 0 {
			poolSize = p.config.MinConnections
		}

		successCount := 0
		for i := 0; i < poolSize; i++ {
			client := NewMCPClientConnection(server)
			if err := client.Connect(); err != nil {
				slog.Warn("Failed to create connection",
					"server", server.Name,
					"index", i,
					"error", err,
				)
				continue
			}
			p.pools[server.Name] = append(p.pools[server.Name], client)
			successCount++
		}

		if successCount > 0 {
			slog.Info("Initialized server pool",
				"server", server.Name,
				"connections", fmt.Sprintf("%d/%d", successCount, poolSize),
			)
		} else {
			slog.Warn("Server has no working connections, will retry on demand",
				"server", server.Name,
			)
		}
	}

	return nil
}

// acquire 获取连接
func (p *Pool) acquire(serverName string) (*MCPClientConnection, error) {
	p.mu.RLock()
	pool, ok := p.pools[serverName]
	activeSet, ok2 := p.active[serverName]
	serverConfig, ok3 := p.serverConfigs[serverName]
	p.mu.RUnlock()

	if !ok || !ok2 || !ok3 {
		return nil, fmt.Errorf("server %s not found in pool", serverName)
	}

	maxConnections := serverConfig.PoolSize
	if maxConnections == 0 {
		maxConnections = p.config.MaxConnections
	}

	startTime := time.Now()

	for {
		p.mu.RLock()
		for _, client := range pool {
			if !activeSet[client] && client.IsConnected() {
				activeSet[client] = true
				client.Touch()
				p.mu.RUnlock()
				return client, nil
			}
		}
		p.mu.RUnlock()

		// 尝试创建新连接
		if len(pool) < maxConnections {
			p.mu.Lock()
			// 再次检查（可能其他协程已经创建了）
			if len(pool) < maxConnections {
				client := NewMCPClientConnection(serverConfig)
				if err := client.Connect(); err != nil {
					slog.Warn("Failed to create new connection",
						"server", serverName,
						"error", err,
					)
					p.mu.Unlock()
				} else {
					pool = append(pool, client)
					p.pools[serverName] = pool
					activeSet[client] = true
					p.mu.Unlock()
					return client, nil
				}
			} else {
				p.mu.Unlock()
			}
		}

		// 等待
		if time.Since(startTime) > time.Duration(p.config.AcquireTimeout)*time.Millisecond {
			return nil, fmt.Errorf("timeout acquiring connection for %s", serverName)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// release 释放连接
func (p *Pool) release(serverName string, client *MCPClientConnection) {
	p.mu.RLock()
	activeSet, ok := p.active[serverName]
	p.mu.RUnlock()

	if ok {
		p.mu.Lock()
		delete(activeSet, client)
		p.mu.Unlock()
	}
}

// Execute 执行操作
func (p *Pool) Execute(serverName string, fn func(*MCPClientConnection) (interface{}, error)) (interface{}, error) {
	client, err := p.acquire(serverName)
	if err != nil {
		return nil, err
	}

	defer p.release(serverName, client)
	return fn(client)
}

// CallTool 调用工具
func (p *Pool) CallTool(serverName string, toolName string, args map[string]interface{}) (*ToolCallResult, error) {
	p.mu.RLock()
	pool, ok := p.pools[serverName]
	p.mu.RUnlock()

	if !ok || len(pool) == 0 {
		return &ToolCallResult{
			Content: []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Server %s is not available (no connections)", serverName)},
			},
			IsError: true,
		}, nil
	}

	result, err := p.Execute(serverName, func(client *MCPClientConnection) (interface{}, error) {
		return client.CallTool(toolName, args)
	})

	if err != nil {
		return &ToolCallResult{
			Content: []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
			},
			IsError: true,
		}, nil
	}

	return result.(*ToolCallResult), nil
}

// Disconnect 断开指定服务器的连接
func (p *Pool) Disconnect(serverName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pool, ok := p.pools[serverName]
	if !ok {
		return nil
	}

	for _, client := range pool {
		client.Disconnect()
	}

	delete(p.pools, serverName)
	delete(p.active, serverName)
	delete(p.serverConfigs, serverName)

	return nil
}

// DisconnectAll 断开所有连接
func (p *Pool) DisconnectAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for serverName := range p.pools {
		for _, client := range p.pools[serverName] {
			client.Disconnect()
		}
	}

	p.pools = make(map[string][]*MCPClientConnection)
	p.active = make(map[string]map[*MCPClientConnection]bool)
	p.serverConfigs = make(map[string]config.ServerConfig)

	return nil
}

// GetStats 获取统计信息
func (p *Pool) GetStats() map[string]map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]map[string]int)
	for serverName, pool := range p.pools {
		activeSet := p.active[serverName]
		stats[serverName] = map[string]int{
			"total":  len(pool),
			"active": len(activeSet),
			"idle":   len(pool) - len(activeSet),
		}
	}

	return stats
}
