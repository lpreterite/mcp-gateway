import { loadConfig } from "../config/loader.js";
import { MCPConnectionPool } from "./pool.js";
import { ToolRegistry } from "./registry.js";
import { ToolMapper } from "./mapper.js";
import { MCPGatewayServer } from "./server.js";
import { MCPClientConnection } from "../mcp/client.js";

const DEFAULT_POOL_CONFIG = {
  minConnections: 1,
  maxConnections: 5,
  acquireTimeout: 10000,
  idleTimeout: 60000,
  maxRetries: 3,
};

async function main(): Promise<void> {
  console.log("[gateway] Starting MCP Gateway...");

  // Load configuration
  const config = loadConfig();
  console.log(`[gateway] Loaded configuration with ${config.servers.length} servers`);

  // Initialize components
  const registry = new ToolRegistry();
  const mapper = new ToolMapper(config.mapping || {}, config.toolFilters || {});
  const pool = new MCPConnectionPool({
    ...DEFAULT_POOL_CONFIG,
    ...config.pool,
  });

  // Initialize connections for each server
  for (const serverConfig of config.servers) {
    if (!serverConfig.enabled) {
      console.log(`[gateway] Skipping disabled server: ${serverConfig.name}`);
      continue;
    }

    console.log(`[gateway] Connecting to ${serverConfig.name}...`);

    const connection = new MCPClientConnection(serverConfig);
    try {
      await connection.connect();
      console.log(`[gateway] Connected to ${serverConfig.name}`);

      // List and register tools
      const tools = await connection.listTools();
      console.log(`[gateway] Found ${tools.length} tools from ${serverConfig.name}`);

      for (const tool of tools) {
        const shouldInclude = mapper.shouldIncludeTool(serverConfig.name, tool.name);
        if (!shouldInclude) {
          console.log(`[gateway] Skipping filtered tool: ${tool.name}`);
          continue;
        }

        const gatewayName = mapper.getGatewayToolName(tool.name, serverConfig.name);
        registry.registerTool({
          name: gatewayName,
          originalName: tool.name,
          serverName: serverConfig.name,
          description: tool.description || "",
          inputSchema: tool.inputSchema as Record<string, unknown>,
          annotations: tool.annotations,
        });
        console.log(`[gateway] Registered tool: ${gatewayName} -> ${serverConfig.name}/${tool.name}`);
      }

      await connection.disconnect();
    } catch (error) {
      console.error(`[gateway] Failed to connect to ${serverConfig.name}:`, error);
    }
  }

  // Initialize the connection pool with the configured servers
  await pool.initialize(config.servers.filter((s) => s.enabled));

  // Create and start the gateway server
  const gateway = new MCPGatewayServer(config, pool, registry, mapper);
  await gateway.start();

  // Handle shutdown
  const shutdown = async (): Promise<void> => {
    console.log("[gateway] Shutting down...");
    await gateway.stop();
    process.exit(0);
  };

  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);
}

main().catch((error) => {
  console.error("[gateway] Fatal error:", error);
  process.exit(1);
});
