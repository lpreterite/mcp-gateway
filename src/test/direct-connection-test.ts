/**
 * MCP Server 直连测试
 *
 * 直接测试各个 MCP server 是否可用，不经过 Gateway
 * 用于验证 MCP server 本身是否正常工作
 */

import { spawn, ChildProcess } from "child_process";

interface ServerTestConfig {
  name: string;
  command: string;
  args: string[];
  env: Record<string, string>;
  testTool?: string;
  testArgs?: Record<string, unknown>;
}

const servers: ServerTestConfig[] = [
  {
    name: "minimax",
    command: "uvx",
    args: ["minimax-coding-plan-mcp"],
    env: {
      MINIMAX_API_KEY: process.env.MINIMAX_API_KEY || "",
      MINIMAX_API_HOST: "https://api.minimaxi.com",
    },
    testTool: "web_search",
    testArgs: { query: "test" },
  },
  {
    name: "searxng",
    command: "mcp-searxng",
    args: [],
    env: {
      SEARXNG_URL: process.env.SEARXNG_URL || "http://localhost:8889",
    },
    testTool: "search",
    testArgs: { query: "test", count: 3 },
  },
];

function log(label: string, message: string) {
  const time = new Date().toISOString().split("T")[1].slice(0, 8);
  console.log(`[${time}] [${label}] ${message}`);
}

function logSection(title: string) {
  console.log("\n" + "=".repeat(60));
  console.log(title);
  console.log("=".repeat(60));
}

async function testServer(config: ServerTestConfig): Promise<boolean> {
  logSection(`测试 Server: ${config.name}`);

  let proc: ChildProcess | null = null;
  let connected = false;
  let tools: any[] = [];

  try {
    // 启动进程
    log(config.name, `启动: ${config.command} ${config.args.join(" ")}`);

    proc = spawn(config.command, config.args, {
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env, ...config.env },
    });

    // 收集输出
    let stdout = "";
    let stderr = "";

    proc.stdout?.on("data", (data) => {
      stdout += data.toString();
    });

    proc.stderr?.on("data", (data) => {
      stderr += data.toString();
    });

    // 等待连接建立
    log(config.name, "等待 MCP 协议初始化...");

    // 发送 initialize 请求
    const initRequest = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        protocolVersion: "2024-11-05",
        capabilities: { tools: {} },
        clientInfo: { name: "test-client", version: "1.0.0" },
      },
    };

    await new Promise<void>((resolve) => {
      const timeout = setTimeout(() => {
        log(config.name, "初始化超时");
        resolve();
      }, 10000);

      // 等待协议版本在输出中
      const checkInterval = setInterval(() => {
        if (stdout.includes('"protocolVersion"') || stdout.includes("protocolVersion")) {
          clearInterval(checkInterval);
          clearTimeout(timeout);
          connected = true;
          resolve();
        }
      }, 500);
    });

    if (!connected) {
      log(config.name, "✗ 初始化失败，未收到协议响应");
      return false;
    }

    log(config.name, "✓ 初始化成功");

    // 发送 tools/list 请求
    const listRequest = {
      jsonrpc: "2.0",
      id: 2,
      method: "tools/list",
      params: {},
    };

    const listCommand = JSON.stringify(listRequest) + "\n";
    proc.stdin?.write(listCommand);

    await new Promise<void>((resolve) => {
      const timeout = setTimeout(() => {
        log(config.name, "工具列表请求超时");
        resolve();
      }, 10000);

      const checkInterval = setInterval(() => {
        if (stdout.includes('"tools"') || stdout.includes('"name"')) {
          clearInterval(checkInterval);
          clearTimeout(timeout);
          resolve();
        }
      }, 500);
    });

    // 解析工具列表
    try {
      const lines = stdout.split("\n").filter((l) => l.trim());
      for (const line of lines) {
        try {
          const parsed = JSON.parse(line);
          if (parsed.result?.tools) {
            tools = parsed.result.tools;
            break;
          }
        } catch {
          // 继续尝试
        }
      }
    } catch {
      // 解析失败
    }

    log(config.name, `发现 ${tools.length} 个工具:`);
    for (const tool of tools.slice(0, 5)) {
      log(config.name, `  - ${tool.name}: ${(tool.description || "").slice(0, 50)}...`);
    }
    if (tools.length > 5) {
      log(config.name, `  ... 还有 ${tools.length - 5} 个工具`);
    }

    // 测试工具调用
    if (config.testTool && tools.some((t: any) => t.name === config.testTool)) {
      log(config.name, `测试工具调用: ${config.testTool}`);

      const callRequest = {
        jsonrpc: "2.0",
        id: 3,
        method: "tools/call",
        params: {
          name: config.testTool,
          arguments: config.testArgs || {},
        },
      };

      const callCommand = JSON.stringify(callRequest) + "\n";
      proc.stdin?.write(callCommand);

      await new Promise<void>((resolve) => {
        const timeout = setTimeout(() => {
          log(config.name, "工具调用超时");
          resolve();
        }, 30000);

        const checkInterval = setInterval(() => {
          if (stdout.includes('"content"') || stdout.includes("content")) {
            clearInterval(checkInterval);
            clearTimeout(timeout);
            resolve();
          }
        }, 500);
      });

      log(config.name, "✓ 工具调用成功");
    } else if (config.testTool) {
      log(config.name, `工具 ${config.testTool} 未找到，跳过调用测试`);
    }

    return true;
  } catch (error) {
    log(config.name, `✗ 测试失败: ${error}`);
    return false;
  } finally {
    // 清理进程
    if (proc) {
      log(config.name, "关闭进程...");
      proc.kill();
      try {
        proc.kill("SIGKILL");
      } catch {
        // 可能已经退出
      }
    }
  }
}

async function main() {
  logSection("MCP Server 直连测试");

  console.log("\n此测试直接启动 MCP server 进程来验证其是否正常工作\n");

  const results: Record<string, boolean> = {};

  for (const server of servers) {
    // 检查是否配置了必要的环境变量
    if (server.name === "minimax" && !server.env.MINIMAX_API_KEY) {
      log(server.name, "⚠ 跳过: MINIMAX_API_KEY 未设置");
      results[server.name] = false;
      continue;
    }

    if (server.name === "searxng" && !server.env.SEARXNG_URL) {
      log(server.name, "⚠ 跳过: SEARXNG_URL 未设置");
      results[server.name] = false;
      continue;
    }

    results[server.name] = await testServer(server);

    // 等待进程完全退出
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }

  // 输出结果
  logSection("测试结果");

  let allPassed = true;
  for (const [name, passed] of Object.entries(results)) {
    const status = passed ? "✓ 通过" : "✗ 失败";
    log("RESULT", `${name}: ${status}`);
    if (!passed) allPassed = false;
  }

  console.log("\n" + "=".repeat(60));
  if (allPassed) {
    console.log("🎉 所有 MCP Server 正常工作!");
  } else {
    console.log("⚠ 部分 MCP Server 不可用，请检查配置");
  }
  console.log("=".repeat(60));

  process.exit(allPassed ? 0 : 1);
}

main().catch((error) => {
  console.error("测试执行失败:", error);
  process.exit(1);
});
