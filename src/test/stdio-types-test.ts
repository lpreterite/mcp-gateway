/**
 * Stdio Bridge 类型定义测试
 *
 * 测试内容：
 * 1. 类型导出正确性
 * 2. 接口一致性
 */

import type {
  JsonRpcRequest,
  JsonRpcResponse,
  BridgeConfig,
  Tool,
  ToolInfo,
  ToolCallResult,
} from "../stdio-bridge/types.js";

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

// ==================== 测试 ====================

/**
 * 测试 1: JsonRpcRequest 类型
 */
function testJsonRpcRequest(): boolean {
  logSection("测试 1: JsonRpcRequest 类型");

  try {
    // 完整请求
    const request: JsonRpcRequest = {
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: {
        name: "test_tool",
        arguments: { query: "hello" },
      },
    };

    if (request.jsonrpc !== "2.0") {
      log("TEST", "✗ jsonrpc 版本应为 2.0");
      return false;
    }

    if (request.id !== 1) {
      log("TEST", "✗ id 应为 1");
      return false;
    }

    if (request.method !== "tools/call") {
      log("TEST", "✗ method 应为 tools/call");
      return false;
    }

    if (!request.params || typeof request.params !== "object") {
      log("TEST", "✗ params 应为对象");
      return false;
    }

    // 无 params 请求
    const noParamsRequest: JsonRpcRequest = {
      jsonrpc: "2.0",
      id: null,
      method: "tools/list",
    };

    if (noParamsRequest.params !== undefined) {
      log("TEST", "✗ tools/list 不应有 params");
      return false;
    }

    log("TEST", "✓ JsonRpcRequest 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ JsonRpcRequest 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 2: JsonRpcResponse 类型
 */
function testJsonRpcResponse(): boolean {
  logSection("测试 2: JsonRpcResponse 类型");

  try {
    // 成功响应
    const successResponse: JsonRpcResponse = {
      jsonrpc: "2.0",
      id: 1,
      result: {
        content: [{ type: "text", text: "hello" }],
      },
    };

    if (successResponse.jsonrpc !== "2.0") {
      log("TEST", "✗ jsonrpc 版本应为 2.0");
      return false;
    }

    if (successResponse.error !== undefined) {
      log("TEST", "✗ 成功响应不应有 error");
      return false;
    }

    // 错误响应
    const errorResponse: JsonRpcResponse = {
      jsonrpc: "2.0",
      id: 1,
      error: {
        code: -32600,
        message: "Invalid Request",
      },
    };

    if (errorResponse.error === undefined) {
      log("TEST", "✗ 错误响应应有 error");
      return false;
    }

    if (errorResponse.error.code !== -32600) {
      log("TEST", "✗ 错误码应为 -32600");
      return false;
    }

    log("TEST", "✓ JsonRpcResponse 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ JsonRpcResponse 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 3: BridgeConfig 类型
 */
function testBridgeConfig(): boolean {
  logSection("测试 3: BridgeConfig 类型");

  try {
    const config: BridgeConfig = {
      gatewayUrl: "http://localhost:4298/sse",
      stdioMode: true,
    };

    if (config.gatewayUrl !== "http://localhost:4298/sse") {
      log("TEST", "✗ gatewayUrl 不正确");
      return false;
    }

    if (config.stdioMode !== true) {
      log("TEST", "✗ stdioMode 应为 true");
      return false;
    }

    log("TEST", "✓ BridgeConfig 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ BridgeConfig 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 4: Tool 类型
 */
function testToolType(): boolean {
  logSection("测试 4: Tool 类型");

  try {
    const tool: Tool = {
      name: "test_tool",
      description: "A test tool",
      inputSchema: {
        type: "object",
        properties: {
          query: { type: "string" },
        },
      },
    };

    if (tool.name !== "test_tool") {
      log("TEST", "✗ tool.name 不正确");
      return false;
    }

    if (!tool.inputSchema || typeof tool.inputSchema !== "object") {
      log("TEST", "✗ tool.inputSchema 应为对象");
      return false;
    }

    log("TEST", "✓ Tool 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ Tool 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 5: ToolInfo 类型
 */
function testToolInfoType(): boolean {
  logSection("测试 5: ToolInfo 类型");

  try {
    const toolInfo: ToolInfo = {
      name: "minimax_test",
      description: "Test tool from minimax",
      serverName: "minimax",
      originalName: "test",
      inputSchema: { type: "object" },
    };

    if (toolInfo.name !== "minimax_test") {
      log("TEST", "✗ toolInfo.name 不正确");
      return false;
    }

    if (toolInfo.serverName !== "minimax") {
      log("TEST", "✗ toolInfo.serverName 不正确");
      return false;
    }

    if (toolInfo.originalName !== "test") {
      log("TEST", "✗ toolInfo.originalName 不正确");
      return false;
    }

    log("TEST", "✓ ToolInfo 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ ToolInfo 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 6: ToolCallResult 类型
 */
function testToolCallResultType(): boolean {
  logSection("测试 6: ToolCallResult 类型");

  try {
    const result: ToolCallResult = {
      content: [
        { type: "text", text: "Hello, world!" },
        { type: "image", text: "data:image/png;base64,abc123" },
      ],
    };

    if (!result.content || !Array.isArray(result.content)) {
      log("TEST", "✗ result.content 应为数组");
      return false;
    }

    if (result.content.length !== 2) {
      log("TEST", "✗ result.content 长度应为 2");
      return false;
    }

    if (result.content[0].type !== "text") {
      log("TEST", "✗ 第一个 content 类型应为 text");
      return false;
    }

    if (result.isError !== undefined) {
      log("TEST", "✗ isError 应为 undefined（未设置）");
      return false;
    }

    // 测试 isError
    const errorResult: ToolCallResult = {
      content: [{ type: "text", text: "Error occurred" }],
      isError: true,
    };

    if (errorResult.isError !== true) {
      log("TEST", "✗ isError 应为 true");
      return false;
    }

    log("TEST", "✓ ToolCallResult 类型正确");
    return true;
  } catch (error) {
    log("TEST", `✗ ToolCallResult 测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 7: 类型重导出
 */
function testTypeReExports(): boolean {
  logSection("测试 7: 类型重导出");

  try {
    // 验证类型可以从 mcp/types 正确重导出
    const tool: Tool = {
      name: "test",
      description: "test",
      inputSchema: {},
    };

    const toolInfo: ToolInfo = {
      name: "test",
      description: "test",
      serverName: "server",
      originalName: "test",
    };

    const result: ToolCallResult = {
      content: [{ type: "text", text: "test" }],
    };

    log("TEST", "✓ 类型重导出正确");
    return true;
  } catch (error) {
    log("TEST", `✗ 类型重导出测试失败: ${error}`);
    return false;
  }
}

// ==================== 主函数 ====================
function main() {
  logSection("Stdio Bridge 类型测试");

  const results: Record<string, boolean> = {};

  results["JsonRpcRequest"] = testJsonRpcRequest();
  results["JsonRpcResponse"] = testJsonRpcResponse();
  results["BridgeConfig"] = testBridgeConfig();
  results["Tool"] = testToolType();
  results["ToolInfo"] = testToolInfoType();
  results["ToolCallResult"] = testToolCallResultType();
  results["类型重导出"] = testTypeReExports();

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
    console.log("🎉 所有类型测试通过!");
  } else {
    console.log("⚠ 部分测试失败，请检查日志");
  }
  console.log("=".repeat(60));

  process.exit(allPassed ? 0 : 1);
}

main();
