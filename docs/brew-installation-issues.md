# Brew 安装与运行问题清单

本文档记录通过 Homebrew 安装并运行 mcp-gateway 项目时遇到的问题及解决方案。

## 🔴 核心问题

### 1. PATH 环境变量缺失

**问题描述**
- brew services 启动的服务无法访问用户的 PATH 环境变量
- 导致 `npx`、`node` 等命令找不到
- MCP 服务器启动失败

**表现**
- 日志显示：`command not found: npx`
- MCP 服务器无法连接
- 工具数量异常减少

**解决方案**

**方案 A：使用完整路径**
在 `/opt/homebrew/etc/mcp-gateway/config.json` 中使用完整路径：

```json
{
  "mcpServers": {
    "playwright": {
      "command": "/Users/packy/.nvm/versions/node/v22.21.0/bin/npx",
      "args": ["-y", "@executeautomation/playwright-mcp-server"]
    }
  }
}
```

**方案 B：自定义 launchd plist（推荐）**

项目提供了自动化脚本来简化 launchd 服务的创建和管理。

**使用安装脚本**

```bash
# 下载并赋予执行权限
curl -fsSL https://raw.githubusercontent.com/yourusername/mcp-gateway/main/scripts/install-launchd.sh -o install-launchd.sh
chmod +x install-launchd.sh

# 安装并启动服务
./install-launchd.sh install

# 查看服务状态
./install-launchd.sh status

# 查看日志
./install-launchd.sh logs

# 实时查看日志
./install-launchd.sh tail

# 重启服务
./install-launchd.sh restart

# 卸载服务
./install-launchd.sh uninstall
```

**脚本功能**

脚本会自动：
1. 检测系统中的常见工具路径（Homebrew、nvm Node.js、uv 等）
2. 创建包含正确 PATH 环境变量的 launchd plist 文件
3. 加载并启动服务
4. 提供服务状态检查和日志查看功能

**手动创建 plist**

如果需要手动创建，创建 `~/Library/LaunchAgents/com.mcp-gateway.plist`：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.mcp-gateway</string>
  <key>ProgramArguments</key>
  <array>
    <string>/opt/homebrew/opt/mcp-gateway/bin/mcp-gateway</string>
    <string>--config</string>
    <string>/opt/homebrew/etc/mcp-gateway/config.json</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>/opt/homebrew/bin:/opt/homebrew/sbin:/Users/packy/.nvm/versions/node/v22.21.0/bin:/usr/local/bin:/usr/bin:/bin</string>
  </dict>
  <key>StandardOutPath</key>
  <string>/opt/homebrew/var/log/mcp-gateway.log</string>
  <key>StandardErrorPath</key>
  <string>/opt/homebrew/var/log/mcp-gateway.err.log</string>
</dict>
</plist>
```

使用 launchctl 管理：
```bash
# 加载服务
launchctl load ~/Library/LaunchAgents/com.mcp-gateway.plist

# 启动服务
launchctl start com.mcp-gateway

# 停止服务
launchctl stop com.mcp-gateway

# 卸载服务
launchctl unload ~/Library/LaunchAgents/com.mcp-gateway.plist
```

### 2. brew services 不支持

**问题描述**
- Formula 没有实现 `#plist` 或 `#service`
- 无法使用 `brew services start/stop/restart` 命令

**表现**
```bash
$ brew services start mcp-gateway
Error: Formula `mcp-gateway` does not implement #plist or #service.
```

**解决方案**
- 手动创建和管理 launchd plist 文件
- 参考上文的"方案 B"

## 🟡 配置问题

### 3. 配置文件位置

**信息**
- 位置：`/opt/homebrew/etc/mcp-gateway/config.json`
- 需要 sudo 权限编辑

**编辑命令**
```bash
sudo vi /opt/homebrew/etc/mcp-gateway/config.json
```

### 4. 不支持远程服务器

**问题描述**
- `type="remote"` 配置不被支持
- 必须使用 `type="local"` 并提供本地命令

**错误配置示例**
```json
{
  "mcpServers": {
    "vectorvein-mcp-server": {
      "type": "remote",
      "url": "ws://localhost:3000"
    }
  }
}
```

**正确配置示例**
```json
{
  "mcpServers": {
    "vectorvein-mcp-server": {
      "type": "local",
      "command": "/path/to/server",
      "args": []
    }
  }
}
```

### 5. 工具路径依赖

**问题描述**
- 所有命令（npx、python 等）都需要使用完整路径
- 不同用户的环境路径不同，需要手动调整

**示例**
```json
{
  "mcpServers": {
    "playwright": {
      "command": "/Users/packy/.nvm/versions/node/v22.21.0/bin/npx",
      "args": ["-y", "@executeautomation/playwright-mcp-server"]
    },
    "searxng": {
      "command": "/opt/homebrew/bin/python3",
      "args": ["-m", "mcp_server_searxng"]
    }
  }
}
```

## 🟢 验证问题

### 6. 服务端口

**信息**
- 默认运行在 4298 端口（非 3000）
- 需要通过健康检查端点验证

**验证命令**
```bash
# 检查健康状态
curl http://localhost:4298/health

# 预期响应
{
  "status": "ok",
  "sessions": 0,
  "pool": {
    "server-name": {
      "active": 0,
      "idle": 2,
      "total": 2
    }
  }
}
```

### 7. 日志调试

**日志位置**
- 标准输出：`/opt/homebrew/var/log/mcp-gateway.log`
- 错误日志：`/opt/homebrew/var/log/mcp-gateway.err.log`

**查看日志**
```bash
# 实时查看错误日志
tail -f /opt/homebrew/var/log/mcp-gateway.err.log

# 查看最近的错误
tail -n 100 /opt/homebrew/var/log/mcp-gateway.err.log
```

## 💡 改进建议

### 1. Formula 改进

**在 formula 中添加默认的 launchd plist**
```ruby
def plist
  <<~EOS
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
      <key>Label</key>
      <string>#{plist_name}</string>
      <key>ProgramArguments</key>
      <array>
        <string>#{opt_bin}/mcp-gateway</string>
        <string>--config</string>
        <string>#{etc}/mcp-gateway/config.json</string>
      </array>
      <key>RunAtLoad</key>
      <true/>
      <key>KeepAlive</key>
      <true/>
      <key>EnvironmentVariables</key>
      <dict>
        <key>PATH</key>
        <string>/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/bin:/bin</string>
      </dict>
    </dict>
    </plist>
  EOS
end

def service; end; # 简单的 service 实现
```

### 2. 文档改进

**明确说明**
- 配置文件位置
- 服务启动方式（不使用 brew services）
- PATH 环境变量问题的解决方案
- 不支持 type="remote"

### 3. 自动化脚本

**提供配置生成脚本**
```bash
# 帮助用户生成包含正确环境变量的配置
mcp-gateway generate-config

# 检测常见工具路径并自动填充
mcp-gateway detect-tools
```

### 4. 健康检查命令

**添加 status 子命令**
```bash
# 检查服务状态
mcp-gateway status

# 输出示例
Status: Running
Port: 4298
Servers: 6/7 connected
Total Tools: 49
```

### 5. 错误提示改进

**配置验证**
- 启动时验证配置文件格式
- 明确提示 type="remote" 不被支持
- 检查命令路径是否存在

## 常见错误排查

### 问题：服务器无法启动
1. 检查日志：`tail -f /opt/homebrew/var/log/mcp-gateway.err.log`
2. 验证命令路径是否存在
3. 确认 PATH 环境变量是否正确配置

### 问题：工具数量异常
1. 检查每个服务器的连接状态
2. 确认所有服务器都成功启动
3. 查看是否有服务器启动失败

### 问题：无法使用 brew services
1. 使用手动 launchd 方式
2. 参考"方案 B"创建自定义 plist 文件

## 参考资源

- [Homebrew Services Documentation](https://docs.brew.sh/Manpage#services-subcommand)
- [launchd.plist(5) Manual Page](https://www.manpagez.com/man/5/launchd.plist/)
- [mcp-gateway Repository](https://github.com/yourusername/mcp-gateway)
