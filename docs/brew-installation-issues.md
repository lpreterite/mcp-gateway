# Homebrew 安装问题故障排除指南

本指南帮助解决通过 Homebrew 安装 mcp-gateway 时可能遇到的常见问题。

## 常见问题

### 1. 权限问题

**症状**: 安装时提示权限拒绝错误。

**解决方案**:
```bash
# 确保 Homebrew 目录权限正确
sudo chown -R $(whoami) /usr/local/libexec
brew doctor
```

### 2. SHA256 校验和不匹配

**症状**: `SHA256 mismatch` 错误。

**解决方案**:
- 确认下载的二进制文件完整
- 检查网络连接是否稳定
- 重新下载尝试:
```bash
brew uninstall mcp-gateway
brew install mcp-gateway
```

### 3. 二进制文件无法执行

**症状**: `Permission denied` 或 `cannot execute binary file`。

**解决方案**:
```bash
# 添加执行权限
chmod +x /usr/local/bin/mcp-gateway

# 或重新安装
brew reinstall mcp-gateway
```

### 4. 找不到命令

**症状**: 安装成功但 `mcp-gateway` 命令找不到。

**解决方案**:
```bash
# 确保 PATH 包含 Homebrew bin 目录
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 验证安装
which mcp-gateway
mcp-gateway --version
```

### 5. 服务无法启动

**症状**: `mcp-gateway service start` 失败。

**解决方案**:
```bash
# 检查服务状态
mcp-gateway service status

# 查看日志
tail -f /var/log/mcp-gateway.log

# 重新安装服务
mcp-gateway service uninstall
mcp-gateway service install
mcp-gateway service start
```

## 获取帮助

如果问题持续存在，请提交 Issue:
- GitHub: https://github.com/lpreterite/mcp-gateway/issues
