import type { PoolConfig, ServerConfig, ToolCallResult } from "../mcp/types.js";
import { MCPClientConnection } from "../mcp/client.js";

export class MCPConnectionPool {
  private pools = new Map<string, MCPClientConnection[]>();
  private active = new Map<string, Set<MCPClientConnection>>();
  private configs = new Map<string, ServerConfig>();
  private config: PoolConfig;

  constructor(config: PoolConfig) {
    this.config = config;
  }

  async initialize(servers: ServerConfig[]): Promise<void> {
    for (const server of servers) {
      if (!server.enabled) continue;

      this.configs.set(server.name, server);
      this.pools.set(server.name, []);
      this.active.set(server.name, new Set());

      const poolSize = server.poolSize ?? this.config.minConnections;
      let successCount = 0;

      for (let i = 0; i < poolSize; i++) {
        try {
          const client = await this.createConnection(server);
          this.pools.get(server.name)!.push(client);
          successCount++;
        } catch (error) {
          console.warn(`[pool] Failed to create connection ${i + 1}/${poolSize} for ${server.name}:`, error instanceof Error ? error.message : error);
        }
      }

      if (successCount > 0) {
        console.log(`[pool] Initialized ${server.name} with ${successCount}/${poolSize} connections`);
      } else {
        console.warn(`[pool] ${server.name} has no working connections, will retry on demand`);
      }
    }
  }

  private async createConnection(serverConfig: ServerConfig): Promise<MCPClientConnection> {
    const connection = new MCPClientConnection(serverConfig);
    await connection.connect();
    return connection;
  }

  async acquire(serverName: string): Promise<MCPClientConnection> {
    const pool = this.pools.get(serverName);
    const config = this.configs.get(serverName);
    const activeSet = this.active.get(serverName);

    if (!pool || !config || !activeSet) {
      throw new Error(`Server ${serverName} not found in pool`);
    }

    const maxConnections = config.poolSize ?? this.config.maxConnections;
    const startTime = Date.now();

    while (true) {
      // Find an available connection
      for (const client of pool) {
        if (!activeSet.has(client)) {
          activeSet.add(client);
          client.touch();
          return client;
        }
      }

      // Check if we can create a new connection
      if (pool.length < maxConnections) {
        try {
          const newClient = await this.createConnection(config);
          pool.push(newClient);
          activeSet.add(newClient);
          newClient.touch();
          return newClient;
        } catch (error) {
          console.warn(`[pool] Failed to create new connection for ${serverName}:`, error instanceof Error ? error.message : error);
        }
      }

      // Wait for a connection to become available
      if (Date.now() - startTime > this.config.acquireTimeout) {
        throw new Error(`Timeout acquiring connection for ${serverName}`);
      }

      await new Promise((resolve) => setTimeout(resolve, 50));
    }
  }

  release(serverName: string, client: MCPClientConnection): void {
    const activeSet = this.active.get(serverName);
    if (!activeSet) return;

    activeSet.delete(client);
  }

  async execute<R>(
    serverName: string,
    fn: (client: MCPClientConnection) => Promise<R>
  ): Promise<R> {
    const client = await this.acquire(serverName);
    try {
      return await fn(client);
    } finally {
      this.release(serverName, client);
    }
  }

  async callTool(
    serverName: string,
    toolName: string,
    args: Record<string, unknown>
  ): Promise<ToolCallResult> {
    const pool = this.pools.get(serverName);
    if (!pool || pool.length === 0) {
      return {
        content: [{ type: "text", text: `Server ${serverName} is not available (no connections)` }],
        isError: true,
      };
    }
    return this.execute(serverName, async (client) => {
      return client.callTool(toolName, args);
    });
  }

  async disconnect(serverName: string): Promise<void> {
    const pool = this.pools.get(serverName);
    if (!pool) return;

    for (const client of pool) {
      try {
        await client.disconnect();
      } catch (error) {
        console.error(`[pool] Error disconnecting ${serverName}:`, error);
      }
    }

    this.pools.delete(serverName);
    this.active.delete(serverName);
    this.configs.delete(serverName);
  }

  async disconnectAll(): Promise<void> {
    for (const serverName of this.pools.keys()) {
      await this.disconnect(serverName);
    }
  }

  getStats(): Record<string, { total: number; active: number; idle: number }> {
    const stats: Record<string, { total: number; active: number; idle: number }> = {};

    for (const [serverName, pool] of this.pools) {
      const activeSet = this.active.get(serverName);
      stats[serverName] = {
        total: pool.length,
        active: activeSet?.size ?? 0,
        idle: pool.length - (activeSet?.size ?? 0),
      };
    }

    return stats;
  }
}
