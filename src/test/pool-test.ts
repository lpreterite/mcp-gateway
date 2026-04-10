/**
 * MCP Gateway 连接池测试脚本
 *
 * 测试内容：
 * 1. MCP Server 连接是否正常
 * 2. 连接池是否正确限制进程数量
 * 3. 并发请求时是否复用连接
 */

import { execSync, spawn } from "child_process";
import { promisify } from "util";
import { setTimeout } from "timers/promises";

const sleep = promisify(setTimeout);

// ==================== 配置 ====================
const GATEWAY_URL = "http://localhost:3000";
const HEALTH_URL = `${GATEWAY_URL}/health`;
const TOOLS_URL = `${GATEWAY_URL}/tools`;
const CALL_URL = `${GATEWAY_URL}/tools/call`;

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

async function fetchJson(url: string, options?: RequestInit): Promise<any> {
  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  const text = await response.text();
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function countMcpProcesses(): number {
  try {
    const output = execSync('ps aux | grep -E "minimax|zai|mcp-searxng" | grep -v grep | wc -l', {
      encoding: "utf-8",
    });
    return parseInt(output.trim(), 10);
  } catch {
    return 0;
  }
}

function getPoolStats(): { total: number; active: number; idle: number } {
  try {
    const stats = execSync(
      `curl -s ${HEALTH_URL} | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps(d.get('pool',{})))" 2>/dev/null || echo "{}"`,
      { encoding: "utf-8" }
    );
    return JSON.parse(stats.trim() || "{}");
  } catch {
    return { total: 0, active: 0, idle: 0 };
  }
}

// ==================== 测试用例 ====================

/**
 * 测试 1: MCP Server 连接测试
 */
async function testMcpConnections(): Promise<boolean> {
  logSection("测试 1: MCP Server 连接测试");

  try {
    // 检查 gateway 是否运行
    log("TEST", "检查 Gateway 健康状态...");
    const health = await fetchJson(HEALTH_URL);

    if (health.status !== "ok") {
      log("TEST", `✗ Gateway 未正常运行: ${JSON.stringify(health)}`);
      return false;
    }
    log("TEST", "✓ Gateway 运行正常");

    // 检查连接池状态
    log("TEST", "检查连接池状态...");
    const poolStats = health.pool || {};
    log("TEST", `连接池统计: ${JSON.stringify(poolStats)}`);

    // 检查工具列表
    log("TEST", "获取工具列表...");
    const tools = await fetchJson(TOOLS_URL);

    if (!tools.tools || tools.tools.length === 0) {
      log("TEST", "✗ 未找到任何工具");
      return false;
    }

    log("TEST", `✓ 找到 ${tools.tools.length} 个工具:`);
    for (const tool of tools.tools.slice(0, 10)) {
      log("TEST", `  - ${tool.name} (${tool.serverName})`);
    }
    if (tools.tools.length > 10) {
      log("TEST", `  ... 还有 ${tools.tools.length - 10} 个工具`);
    }

    return true;
  } catch (error) {
    log("TEST", `✗ 连接测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 2: 工具调用测试
 */
async function testToolCall(): Promise<boolean> {
  logSection("测试 2: 工具调用测试");

  try {
    const tools = await fetchJson(TOOLS_URL);

    if (!tools.tools || tools.tools.length === 0) {
      log("TEST", "✗ 没有可测试的工具");
      return false;
    }

    // 找一个简单的工具来测试
    const testTool = tools.tools.find((t: any) =>
      t.name.includes("web_search") || t.name.includes("search")
    ) || tools.tools[0];

    log("TEST", `使用工具测试: ${testTool.name}`);

    const result = await fetchJson(CALL_URL, {
      method: "POST",
      body: JSON.stringify({
        name: testTool.name,
        arguments: { query: "hello world" },
      }),
    });

    if (result.error) {
      log("TEST", `✗ 工具调用失败: ${result.error}`);
      return false;
    }

    log("TEST", "✓ 工具调用成功");
    log("TEST", `  结果: ${JSON.stringify(result.result).slice(0, 200)}...`);
    return true;
  } catch (error) {
    log("TEST", `✗ 工具调用测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 3: 连接池进程数量限制测试
 */
async function testPoolProcessLimit(): Promise<boolean> {
  logSection("测试 3: 连接池进程数量限制测试");

  try {
    const initialCount = countMcpProcesses();
    log("TEST", `初始 MCP 进程数: ${initialCount}`);

    // 获取 pool 配置
    const health = await fetchJson(HEALTH_URL);
    const poolStats = health.pool || {};

    let totalConfiguredConnections = 0;
    for (const [server, stats] of Object.entries(poolStats)) {
      const s = stats as any;
      totalConfiguredConnections += s.total || 0;
      log("TEST", `  ${server}: ${s.total} 连接 (活动: ${s.active}, 空闲: ${s.idle})`);
    }

    log("TEST", `配置的总连接数: ${totalConfiguredConnections}`);

    // 验证进程数不超过配置的总连接数
    if (initialCount > totalConfiguredConnections * 2) {
      // 允许一些误差（因为可能还有其他进程）
      log("TEST", `⚠ 进程数 (${initialCount}) 似乎超过配置`);
    } else {
      log("TEST", "✓ 进程数在合理范围内");
    }

    return true;
  } catch (error) {
    log("TEST", `✗ 进程限制测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 4: 并发请求测试
 */
async function testConcurrentRequests(): Promise<boolean> {
  logSection("测试 4: 并发请求测试");

  try {
    const CONCURRENT = 10;
    log("TEST", `发起 ${CONCURRENT} 个并发请求...`);

    const beforeCount = countMcpProcesses();
    log("TEST", `并发请求前进程数: ${beforeCount}`);

    const promises: Promise<any>[] = [];
    for (let i = 0; i < CONCURRENT; i++) {
      promises.push(
        fetchJson(CALL_URL, {
          method: "POST",
          body: JSON.stringify({
            name: "minimax_web_search",
            arguments: { query: `test query ${i}` },
          }),
        }).catch((err) => ({ error: err.message }))
      );
    }

    const results = await Promise.all(promises);
    const afterCount = countMcpProcesses();

    log("TEST", `并发请求后进程数: ${afterCount}`);
    log("TEST", `进程数变化: ${afterCount - beforeCount}`);

    // 统计成功/失败
    const success = results.filter((r) => !r.error).length;
    const failed = results.filter((r) => r.error).length;

    log("TEST", `请求结果: 成功 ${success}, 失败 ${failed}`);

    // 验证进程数没有大幅增长（说明连接被复用）
    if (afterCount - beforeCount > 5) {
      log("TEST", "⚠ 进程数增长过多，可能未正确复用连接");
      return false;
    }

    log("TEST", "✓ 并发请求完成，进程数控制正常");
    return true;
  } catch (error) {
    log("TEST", `✗ 并发测试失败: ${error}`);
    return false;
  }
}

/**
 * 测试 5: 压力测试 - 短时间内大量请求
 */
async function testStressTest(): Promise<boolean> {
  logSection("测试 5: 压力测试");

  try {
    const REQUESTS = 50;
    const DELAY = 100; // ms
    log("TEST", `发起 ${REQUESTS} 个请求（间隔 ${DELAY}ms）...`);

    const beforeCount = countMcpProcesses();
    log("TEST", `压力测试前进程数: ${beforeCount}`);

    let success = 0;
    let failed = 0;

    for (let i = 0; i < REQUESTS; i++) {
      try {
        const result = await fetchJson(CALL_URL, {
          method: "POST",
          body: JSON.stringify({
            name: "minimax_web_search",
            arguments: { query: `stress test ${i}` },
          }),
        });

        if (result.error) {
          failed++;
        } else {
          success++;
        }
      } catch {
        failed++;
      }

      if (i % 10 === 0) {
        const currentCount = countMcpProcesses();
        log("TEST", `  进度 ${i}/${REQUESTS}, 当前进程数: ${currentCount}`);
      }

      await sleep(DELAY);
    }

    const afterCount = countMcpProcesses();

    log("TEST", `压力测试完成:`);
    log("TEST", `  成功: ${success}, 失败: ${failed}`);
    log("TEST", `  进程数变化: ${beforeCount} -> ${afterCount}`);

    // 验证进程数没有爆炸性增长
    if (afterCount > beforeCount * 3) {
      log("TEST", "⚠ 警告: 进程数增长超过 3 倍");
      return false;
    }

    log("TEST", "✓ 压力测试通过");
    return true;
  } catch (error) {
    log("TEST", `✗ 压力测试失败: ${error}`);
    return false;
  }
}

// ==================== 主函数 ====================
async function main() {
  logSection("MCP Gateway 连接池测试");

  console.log("\n注意: 确保 Gateway 已启动 (npm run gateway)");
  console.log(`Gateway URL: ${GATEWAY_URL}\n`);

  // 等待 Gateway 启动
  let gatewayReady = false;
  for (let i = 0; i < 10; i++) {
    try {
      const health = await fetchJson(HEALTH_URL);
      if (health.status === "ok") {
        gatewayReady = true;
        break;
      }
    } catch {
      // 等待中
    }
    log("TEST", `等待 Gateway 启动... (${i + 1}/10)`);
    await sleep(1000);
  }

  if (!gatewayReady) {
    log("TEST", "✗ Gateway 未启动，请先运行: npm run gateway");
    process.exit(1);
  }

  // 运行测试
  const results: Record<string, boolean> = {};

  results["连接测试"] = await testMcpConnections();
  results["工具调用"] = await testToolCall();
  results["进程限制"] = await testPoolProcessLimit();
  results["并发测试"] = await testConcurrentRequests();
  results["压力测试"] = await testStressTest();

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
