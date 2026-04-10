import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

interface ServerConfig {
  name: string;
  command: string;
  args?: string[];
  env?: Record<string, string>;
}

const testServers: ServerConfig[] = [
  {
    name: "minimax",
    command: "/Users/packy/.local/bin/uvx",
    args: ["minimax-coding-plan-mcp", "-y"],
    env: {
      MINIMAX_API_KEY: "sk-cp-p-G7vn4hHvohWo8rVByjXz2jpHCDIqyj7Y1YBarnLRo2oMR8tP3RTR0K6bLMir4OHbcZOgGWusfnviCHzIt73sn3cHmlDcj_9RWfHaolt2lgwJha8NZpRZk",
      MINIMAX_API_HOST: "https://api.minimaxi.com"
    },
  },
  {
    name: "searxng",
    command: "mcp-searxng",
    env: { SEARXNG_URL: "http://localhost:8889" },
  },
];

async function testServer(config: ServerConfig): Promise<void> {
  console.error(`\n[测试] ${config.name}`);
  console.error("=".repeat(40));

  let client: Client | null = null;
  let transport: StdioClientTransport | null = null;

  try {
    transport = new StdioClientTransport({
      command: config.command,
      args: config.args || [],
      env: config.env,
    });

    client = new Client(
      { name: `test-client`, version: "1.0.0" },
      { capabilities: {} }
    );

    console.error(`[连接] 启动 ${config.command} ${config.args?.join(" ") || ""}`);
    await client.connect(transport);

    console.error(`[成功] 已连接，正在获取工具列表...`);
    const toolsResult = await client.listTools();
    console.error(`[成功] 工具数量: ${toolsResult.tools.length}`);
    
    for (const tool of toolsResult.tools) {
      const desc = tool.description?.replace(/\n/g, " ").slice(0, 60) || "无描述";
      console.error(`  - ${tool.name}: ${desc}...`);
    }

    if (toolsResult.tools.length > 0) {
      const firstTool = toolsResult.tools[0];
      console.error(`\n[测试调用] 尝试调用工具: ${firstTool.name}`);
      try {
        const result = await client.callTool({
          name: firstTool.name,
          arguments: config.name === "minimax"
            ? { query: "hello world" }
            : { query: "hello world", count: 3 }
        }) as { content: Array<{ type: string; text?: string }>; isError?: boolean };
        console.error(`[成功] 工具调用完成`);
        console.error(`[结果] isError: ${result.isError}, content数量: ${result.content.length}`);
        if (result.content[0]?.type === "text") {
          const text = result.content[0].text || "";
          console.error(`[内容] ${text.slice(0, 500)}...`);
        }
      } catch (callError) {
        console.error(`[警告] 工具调用失败: ${callError}`);
      }
    }

  } catch (error) {
    console.error(`[失败] ${error}`);
  } finally {
    if (client) {
      try {
        await client.close();
      } catch {}
    }
    if (transport) {
      try {
        await transport.close();
      } catch {}
    }
    console.error(`[清理] 进程已关闭`);
  }
}

async function main() {
  console.error("MCP 服务器连接测试");
  console.error("=".repeat(40));

  for (const server of testServers) {
    await testServer(server);
    await new Promise((r) => setTimeout(r, 2000));
  }

  console.error("\n[完成] 所有服务器测试完毕");
  process.exit(0);
}

main().catch((e) => {
  console.error("[致命错误]", e);
  process.exit(1);
});
