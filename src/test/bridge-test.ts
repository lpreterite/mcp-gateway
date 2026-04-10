/**
 * Stdio Bridge 单元测试
 *
 * 测试内容：
 * 1. 类型定义正确性
 * 2. StdioBridge 类的基本功能
 * 3. JSON-RPC 请求/响应处理
 */

import { StdioBridge } from "../stdio-bridge/bridge.js";
import type { JsonRpcRequest, BridgeConfig } from "../stdio-bridge/types.js";

// ==================== 配置 ====================
const GATEWAY_URL = "http://localhost:4298";

// ==================== 工具函数 ====================
function log(label: string, message: string) {
  const time = new Date().toISOString();
  console.log(`[${time}] [${label}] ${message}`);
}

function logSection(title: string) {
  console.log("\n" + "=".repeat(60));
  console.log(title);
  console.log("=".repeat(60));
}

// ==================== 单元测试 ====================

/**
 * 测试 1: 类型定义测试
 */
function testTypeDefinitions(): boolean {
  logSection("测试 1: 类型定义测试");

  try {
    // 验证 JsonRpcRequest 类型
    const request: JsonRpcRequest = {
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: {
        name: "test_tool",
        arguments: { key: "value" },
      },
    };

    if (request.jsonrpc !== "2.0") {
      log("TEST", "✗ JsonRpcRequest.jsonrpc 类型错误");
      return false;
    }

    if (request.method !== "tools/call") {
      log("TEST", "✗ JsonRpcRequest.method 类型错误");
      return false;
    }

    log("TEST", "✓ JsonRpcRequest 类型正确");

    // 验证 BridgeConfig 类型
    const config: BridgeConfig = {
      gatewayUrl: "http://localhost:4298/sse",
      stdioMode: true,
    };

    if (config.gatewayUrl !== "http://localhost:4298/sse") {
      log("TEST", "✗ BridgeConfig.gatewayUrl 类型错误");
      return false;
    }

    if (config.stdioMode !== true) {
      log("TEST", "✗ BridgeConfig.stdioMode 类型错误");
      return false;
    }

    log("TEST", "✓ BridgeConfig 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ 类型定义测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 2: StdioBridge 构造函数
 */
function testBridgeConstructor(): boolean {
  logSection("测试 2: StdioBridge 构造函数");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    if (!bridge) {
      log("TEST", "✗ StdioBridge 实例创建失败");
      return false;
    }

    if (bridge.isConnected()) {
      log("TEST", "✗ 初始状态应该未连接");
      return false;
    }

    log("TEST", "✓ StdioBridge 构造函数正常");
    return true;
  } catch (error) {
    log("TEST", `✗ 构造函数测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 3: StdioBridge 断开连接
 */
async function testBridgeDisconnect(): Promise<boolean> {
  logSection("测试 3: StdioBridge 断开连接");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    // 初始应该未连接
    if (bridge.isConnected()) {
      log("TEST", "✗ 初始状态应该未连接");
      return false;
    }

    // 断开连接应该可以正常执行（即使未连接）
    await bridge.disconnect();

    if (bridge.isConnected()) {
      log("TEST", "✗ 断开连接后状态应该未连接");
      return false;
    }

    log("TEST", "✓ StdioBridge 断开连接正常");
    return true;
  } catch (error) {
    log("TEST", `✗ 断开连接测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 4: StdioBridge 连接失败处理
 */
async function testBridgeConnectionFailure(): Promise<boolean> {
  logSection("测试 4: StdioBridge 连接失败处理");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: "http://invalid-host:9999/sse",
      stdioMode: true,
    });

    try {
      await bridge.connect();
      log("TEST", "✗ 应该抛出连接错误");
      return false;
    } catch (error) {
      if (error instanceof Error) {
        log("TEST", `✓ 正确抛出连接错误: ${error.message}`);
        return true;
      }
      log("TEST", "✗ 错误类型不正确");
      return false;
    }
  } catch (error) {
    log("TEST", `✗ 连接失败测试异常: ${error}`);
    return false;
  }
}

/**
 * 测试 5: StdioBridge 未连接时调用工具
 */
async function testBridgeCallWithoutConnection(): Promise<boolean> {
  logSection("测试 5: StdioBridge 未连接时调用工具");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    try {
      await bridge.callTool("test_tool", {});
      log("TEST", "✗ 未连接时调用工具应该抛出错误");
      return false;
    } catch (error) {
      if (error instanceof Error && error.message.includes("Not connected")) {
        log("TEST", "✓ 正确抛出未连接错误");
        return true;
      }
      log("TEST", `✗ 错误消息不正确: ${error}`);
      return false;
    }
  } catch (error) {
    log("TEST", `✗ 未连接调用测试异常: ${error}`);
    return false;
  }
}

/**
 * 测试 6: StdioBridge 获取工具列表（REST API 不需要连接）
 * 注意: listTools() 使用 REST /tools 端点，不需要 sessionId
 */
async function testBridgeListToolsWithoutConnection(): Promise<boolean> {
  logSection("测试 6: StdioBridge 获取工具列表 (REST)");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    // listTools uses REST endpoint /tools, which doesn't require sessionId
    // But it DOES require the gateway to be running
    try {
      const tools = await bridge.listTools();
      // If gateway is running, this succeeds
      if (Array.isArray(tools)) {
        log("TEST", `✓ REST API /tools 可用 (${tools.length} 个工具)`);
        return true;
      }
      log("TEST", "✗ 工具列表格式不正确");
      return false;
    } catch (error) {
      // If gateway is not running, this is expected
      if (error instanceof Error && error.message.includes("Failed to fetch tools")) {
        log("TEST", "✓ Gateway 未运行（这是预期行为）");
        return true;
      }
      log("TEST", `✗ 错误: ${error}`);
      return false;
    }
  } catch (error) {
    log("TEST", `✗ 未连接获取工具测试异常: ${error}`);
    return false;
  }
}

// ==================== 集成测试 ====================

/**
 * 测试 7: StdioBridge 连接到真实 Gateway
 */
async function testBridgeConnectToGateway(): Promise<boolean> {
  logSection("测试 7: StdioBridge 连接到真实 Gateway");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    try {
      await bridge.connect();

      if (!bridge.isConnected()) {
        log("TEST", "✗ 连接后状态应该已连接");
        return false;
      }

      log("TEST", "✓ StdioBridge 连接成功");
      return true;
    } catch (error) {
      if (error instanceof Error && error.message.includes("Failed to connect to gateway")) {
        log("TEST", `✗ Gateway 未运行: ${error.message}`);
        log("TEST", "请先启动 Gateway: npm run gateway");
        return false;
      }
      throw error;
    } finally {
      await bridge.disconnect();
    }
  } catch (error) {
    log("TEST", `✗ 连接 Gateway 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 8: StdioBridge 获取工具列表
 */
async function testBridgeListTools(): Promise<boolean> {
  logSection("测试 8: StdioBridge 获取工具列表");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    try {
      await bridge.connect();
      const tools = await bridge.listTools();

      if (!Array.isArray(tools)) {
        log("TEST", "✗ 工具列表应该是一个数组");
        return false;
      }

      log("TEST", `✓ 获取到 ${tools.length} 个工具`);

      if (tools.length > 0) {
        for (const tool of tools.slice(0, 3)) {
          log("TEST", `  - ${tool.name}`);
        }
        if (tools.length > 3) {
          log("TEST", `  ... 还有 ${tools.length - 3} 个工具`);
        }
      }

      return true;
    } finally {
      await bridge.disconnect();
    }
  } catch (error) {
    log("TEST", `✗ 获取工具列表测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 9: StdioBridge 调用工具
 */
async function testBridgeCallTool(): Promise<boolean> {
  logSection("测试 9: StdioBridge 调用工具");

  try {
    const bridge = new StdioBridge({
      gatewayUrl: GATEWAY_URL,
      stdioMode: true,
    });

    try {
      await bridge.connect();

      // Wait briefly for sessionId to be established
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // 先获取工具列表
      const tools = await bridge.listTools();

      if (tools.length === 0) {
        log("TEST", "⚠ 没有可测试的工具，跳过");
        return true;
      }

      // 使用第一个工具进行测试
      const testTool = tools[0];
      log("TEST", `使用工具测试: ${testTool.name}`);

      const result = await bridge.callTool(testTool.name, {});

      if (!result || !result.content) {
        log("TEST", "✗ 工具调用结果格式不正确");
        return false;
      }

      log("TEST", "✓ 工具调用成功");
      log("TEST", `  结果类型: ${result.content[0]?.type || "unknown"}`);
      return true;
    } finally {
      await bridge.disconnect();
    }
  } catch (error) {
    log("TEST", `✗ 工具调用测试失败: ${error}`);
    return false;
  }
}

// ==================== 主函数 ====================
async function main() {
  logSection("Stdio Bridge 单元测试");

  console.log("\n注意: 集成测试需要 Gateway 运行在 localhost:4298");
  console.log(`Gateway URL: ${GATEWAY_URL}`);
  console.log("如需启动 Gateway，请运行: npm run gateway\n");

  // 运行单元测试
  const results: Record<string, boolean> = {};

  results["类型定义"] = testTypeDefinitions();
  results["构造函数"] = testBridgeConstructor();
  results["断开连接"] = await testBridgeDisconnect();
  results["连接失败处理"] = await testBridgeConnectionFailure();
  results["未连接调用工具"] = await testBridgeCallWithoutConnection();
  results["未连接获取工具"] = await testBridgeListToolsWithoutConnection();

  // 集成测试（需要 Gateway 运行）
  let gatewayAvailable = false;
  try {
    const response = await fetch("http://localhost:4298/health");
    gatewayAvailable = response.ok;
  } catch {
    // Gateway 不可用
  }

  if (gatewayAvailable) {
    logSection("Gateway 可用，运行集成测试");

    results["连接到 Gateway"] = await testBridgeConnectToGateway();
    results["获取工具列表"] = await testBridgeListTools();
    results["调用工具"] = await testBridgeCallTool();
  } else {
    logSection("Gateway 不可用，跳过集成测试");
    log("TEST", "请先启动 Gateway: npm run gateway");
  }

  // 输出结果
  logSection("测试结果汇总");

  let allPassed = true;
  for (const [name, passed] of Object.entries(results)) {
    const status = passed ? "✓ 通过" : "✗ 失败";
    log("RESULT", `${name}: ${status}`);
    if (!passed) allPassed = false;
  }

  console.log("\n" + "=".repeat(60));
  if (allPassed) {
    console.log("🎉 所有测试通过!");
  } else {
    console.log("⚠ 部分测试失败，请检查日志");
  }
  console.log("=".repeat(60));

  process.exit(allPassed ? 0 : 1);
}

main().catch((error) => {
  console.error("测试执行失败:", error);
  process.exit(1);
});
