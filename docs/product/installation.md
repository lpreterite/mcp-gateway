# MCP Gateway 安装指南

本文档面向需要安装和使用 MCP Gateway 的终端用户，介绍各种安装方式及常见问题。

---

## 前置要求

### 操作系统

| 操作系统 | 支持架构 | 说明 |
|---------|---------|------|
| macOS | darwin/arm64 (Apple Silicon), darwin/amd64 | 推荐使用 Homebrew 安装 |
| Linux | linux/amd64 | 支持 systemd 的发行版 |
| Windows | windows/amd64 | 需要使用 PowerShell 或 Git Bash |

### Go 版本

- **最低版本**：Go 1.21+
- **推荐版本**：Go 1.26+

```bash
# 检查当前 Go 版本
go version
```

### 其他依赖

| 依赖 | 必需 | 说明 |
|------|------|------|
| Git | 是 | 用于源码克隆和版本管理 |
| Docker | 否 | 仅在使用 Docker 部署时需要 |
| Homebrew | 否 | 仅在使用 macOS Homebrew 安装时需要 |

---

## 安装方式

### 方式一：Homebrew 安装（推荐）

> 适用于 macOS 和 Linux 用户，最简单的安装方式，自动处理依赖和路径。

```bash
# 添加 Homebrew tap
brew tap lpreterite/tap/mcp-gateway

# 安装
brew install mcp-gateway

# 验证安装
mcp-gateway --version
```

**Homebrew 安装后的配置位置**：
- 二进制文件：`$(brew --prefix)/bin/mcp-gateway`
- 配置文件：`$(brew --prefix)/etc/mcp-gateway/config.json`

---

### 方式二：go install 安装

> 适用于已安装 Go 环境的开发者，一条命令完成安装。

```bash
# 安装最新版本
go install github.com/lpreterite/mcp-gateway@latest

# 验证安装
mcp-gateway --version
```

> **提示**：`go install` 会将二进制文件安装到 `$GOPATH/bin` 或 `$HOME/go/bin`，请确保该路径已加入 `PATH` 环境变量。

```bash
# 检查 PATH 是否包含 Go bin 目录
echo $PATH | tr ':' '\n' | grep -E 'go/bin|GOPATH'

# 如果没有，添加到 shell 配置
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```

---

### 方式三：二进制下载

> 适用于不想安装 Go 或构建工具的用户，直接下载预编译的二进制文件。

#### 手动下载

1. 访问 [GitHub Releases](https://github.com/lpreterite/mcp-gateway/releases) 页面
2. 下载对应平台的压缩包：
   - `mcp-gateway-darwin-arm64.tar.gz` - macOS Apple Silicon
   - `mcp-gateway-darwin-amd64.tar.gz` - macOS Intel
   - `mcp-gateway-linux-amd64.tar.gz` - Linux
   - `mcp-gateway-windows-amd64.zip` - Windows

3. 解压并安装：

```bash
# Linux/macOS
tar -xzf mcp-gateway-*.tar.gz
sudo mv mcp-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/mcp-gateway

# Windows (使用 PowerShell)
# 解压后移动到合适的位置，添加到 PATH
```

#### 使用安装脚本（Linux/macOS）

```bash
curl -fsSL https://raw.githubusercontent.com/lpreterite/mcp-gateway/main/scripts/install.sh | bash
```

---

### 方式四：源码编译

> 适用于需要自定义构建或参与开发的用户。

#### 1. 克隆源码

```bash
git clone https://github.com/lpreterite/mcp-gateway.git
cd mcp-gateway
```

#### 2. 安装 Go 依赖

```bash
go mod download
```

#### 3. 编译二进制文件

```bash
# 编译当前平台版本
make build

# 或手动编译
go build -o mcp-gateway ./cmd/gateway

# 交叉编译（可选）
make build-all  # 编译所有平台版本
```

编译产物位于 `dist/` 目录：

```
dist/
├── mcp-gateway-darwin-arm64
├── mcp-gateway-darwin-amd64
├── mcp-gateway-linux-amd64
└── mcp-gateway-windows-amd64.exe
```

#### 4. 安装到系统

```bash
# Linux/macOS
sudo cp dist/mcp-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/mcp-gateway

# 验证
mcp-gateway --version
```

---

## 快速开始

### 第一步：初始化配置

```bash
# 创建用户配置文件（推荐）
mcp-gateway config init
```

这将在以下位置创建默认配置文件：
- **macOS/Linux**：`~/.config/mcp-gateway/config.json`
- **Homebrew 安装**：`$(brew --prefix)/etc/mcp-gateway/config.json`

### 第二步：编辑配置

使用文本编辑器打开配置文件，添加你的 MCP 服务器：

```bash
# 使用 vim
vim ~/.config/mcp-gateway/config.json

# 或使用 VS Code
code ~/.config/mcp-gateway/config.json
```

**最小配置示例**：

```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 4298,
    "cors": true
  },
  "pool": {
    "minConnections": 1,
    "maxConnections": 5,
    "acquireTimeout": 5000,
    "idleTimeout": 30000
  },
  "servers": [
    {
      "name": "minimax",
      "type": "local",
      "command": ["uvx", "minimax-coding-plan-mcp"],
      "enabled": true,
      "poolSize": 3
    }
  ]
}
```

### 第三步：启动服务

```bash
# 直接运行（开发模式）
mcp-gateway

# 作为后台服务运行
mcp-gateway service install
mcp-gateway service start

# 或使用 Homebrew 服务（macOS）
brew services start mcp-gateway
```

### 第四步：验证安装

```bash
# 检查服务状态
mcp-gateway service status

# 或访问健康检查端点
curl http://localhost:4298/health
```

**健康检查响应示例**：

```json
{
  "status": "ok",
  "version": "1.2.1",
  "uptime": 3600,
  "sessions": 0,
  "pool": {
    "minimax": {"total": 3, "active": 0, "idle": 3}
  }
}
```

---

## 卸载说明

### Homebrew 卸载

```bash
# 停止服务（如果正在运行）
brew services stop mcp-gateway

# 卸载
brew uninstall mcp-gateway

# 清理配置文件（可选）
rm -rf ~/.config/mcp-gateway
rm -rf $(brew --prefix)/etc/mcp-gateway
```

### go install 卸载

```bash
# 删除二进制文件
rm $(go env GOPATH)/bin/mcp-gateway

# 或
rm $HOME/go/bin/mcp-gateway

# 删除配置文件（可选）
rm -rf ~/.config/mcp-gateway
```

### 源码安装卸载

```bash
# 删除二进制文件
sudo rm /usr/local/bin/mcp-gateway

# 删除配置文件（可选）
rm -rf ~/.config/mcp-gateway
```

### Docker 卸载

```bash
# 停止并删除容器
docker-compose down

# 删除镜像
docker rmi mcp-gateway:latest

# 删除配置文件（可选）
rm -rf ~/mcp-gateway
```

---

## 常见问题

### Q: 提示 "command not found: mcp-gateway"

**原因**：PATH 环境变量未包含二进制文件所在目录。

**解决方法**：

```bash
# 重新加载 shell 配置
source ~/.zshrc

# 或手动添加到 PATH
export PATH=$PATH:/usr/local/bin
```

### Q: 配置文件加载失败

**原因**：配置文件不存在或格式错误。

**解决方法**：

```bash
# 查看配置路径优先级
mcp-gateway config info

# 初始化新配置
mcp-gateway config init
```

### Q: 端口 4298 被占用

**解决方法**：使用 `--port` 参数指定其他端口：

```bash
mcp-gateway --port 8080
```

### Q: MCP 服务器启动失败

**可能原因**：
- 命令不存在（如 `uvx` 未安装）
- 环境变量缺失
- 权限问题

**解决方法**：

```bash
# 使用 debug 模式查看详细日志
mcp-gateway --log-level debug

# 检查 MCP 服务器命令是否可用
which uvx
```

---

## 下一步

- 配置你的 MCP 服务器：参考 [部署说明](./deployment.md)
- 集成到 OpenCode 或 Claude Desktop：参考 [GitHub README](https://github.com/lpreterite/mcp-gateway)
