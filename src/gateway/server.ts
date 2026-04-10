import express, { Request, Response } from "express";
import cors from "cors";
import { z } from "zod";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { MCPConnectionPool } from "./pool.js";
import { ToolRegistry } from "./registry.js";
import { ToolMapper } from "./mapper.js";
import type { ValidatedConfig } from "../config/loader.js";

// Create request schemas with method literals
const ListToolsRequestSchema = z.object({
  method: z.literal("tools/list"),
  params: z.object({}).optional(),
});

const CallToolRequestSchema = z.object({
  method: z.literal("tools/call"),
  params: z.object({
    name: z.string(),
    arguments: z.record(z.unknown()).optional(),
  }),
});

interface ClientSession {
  transport: StreamableHTTPServerTransport;
}

export class MCPGatewayServer {
  private app: express.Application;
  private mcpServer: Server;
  private pool: MCPConnectionPool;
  private registry: ToolRegistry;
  private mapper: ToolMapper;
  private config: ValidatedConfig;
  private sessions = new Map<string, ClientSession>();

  constructor(
    config: ValidatedConfig,
    pool: MCPConnectionPool,
    registry: ToolRegistry,
    mapper: ToolMapper
  ) {
    this.config = config;
    this.pool = pool;
    this.registry = registry;
    this.mapper = mapper;

    this.app = express();
    this.app.use(cors());
    this.app.use(express.json());

    this.mcpServer = new Server(
      { name: "mcp-gateway", version: "1.0.0" },
      { capabilities: { tools: {} } }
    );

    this.setupHandlers();
    this.setupRoutes();
  }

  private setupHandlers(): void {
    // Handle tools/list request
    this.mcpServer.setRequestHandler(ListToolsRequestSchema, async () => {
      const tools = this.registry.getAllTools().map((tool) => ({
        name: tool.name,
        description: tool.description,
        inputSchema: tool.inputSchema || { type: "object", properties: {} },
        annotations: tool.annotations,
      }));
      console.log(`[gateway] tools/list called, returning ${tools.length} tools`);
      return { tools };
    });

    // Handle tools/call request
    this.mcpServer.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args = {} } = request.params;
      console.log(`[gateway] tools/call: ${name}`);

      try {
        const tool = this.registry.getTool(name);
        if (!tool) {
          throw new Error(`Tool ${name} not found`);
        }

        const serverName = tool.serverName;
        const originalName = this.mapper.getOriginalToolName(name, serverName) || tool.originalName;

        const result = await this.pool.callTool(serverName, originalName, args as Record<string, unknown>);
        return result;
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`[gateway] Tool call failed: ${message}`);
        return {
          content: [{ type: "text", text: `Error: ${message}` }],
          isError: true,
        };
      }
    });
  }

  private setupRoutes(): void {
    // MCP endpoint using Streamable HTTP transport
    this.app.post("/mcp", async (req: Request, res: Response) => {
      const transport = new StreamableHTTPServerTransport({
        sessionIdGenerator: () => `session-${Date.now()}`,
      });

      this.mcpServer.connect(transport).catch((err) => {
        console.error("[gateway] Failed to connect transport:", err);
      });

      await transport.handleRequest(req, res, req.body);
    });

    // SSE endpoint for client connections (GET for streaming)
    this.app.get("/sse", async (req: Request, res: Response) => {
      const clientId = (req.query.clientId as string) || `client-${Date.now()}`;

      console.log(`[gateway] SSE connection from ${clientId}`);

      res.setHeader("Content-Type", "text/event-stream");
      res.setHeader("Cache-Control", "no-cache");
      res.setHeader("Connection", "keep-alive");
      res.setHeader("Access-Control-Allow-Origin", "*");

      const transport = new StreamableHTTPServerTransport({
        sessionIdGenerator: () => clientId,
      });

      this.sessions.set(clientId, { transport });

      await this.mcpServer.connect(transport);
      await transport.handleRequest(req, res);

      // Keep connection alive with heartbeat
      const heartbeat = setInterval(() => {
        res.write(`: heartbeat\n\n`);
      }, 15000);

      req.on("close", () => {
        clearInterval(heartbeat);
        this.sessions.delete(clientId);
        console.log(`[gateway] SSE connection closed: ${clientId}`);
      });
    });

    // REST endpoint for tool calls
    this.app.post("/tools/call", async (req: Request, res: Response) => {
      const { name, arguments: args = {} } = req.body;

      if (!name) {
        res.status(400).json({ error: "Tool name is required" });
        return;
      }

      try {
        const tool = this.registry.getTool(name);
        if (!tool) {
          res.status(404).json({ error: `Tool ${name} not found` });
          return;
        }

        const serverName = tool.serverName;
        const originalName = this.mapper.getOriginalToolName(name, serverName) || tool.originalName;

        const result = await this.pool.callTool(serverName, originalName, args);
        res.json({ result });
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        console.error(`[gateway] Tool call failed: ${message}`);
        res.status(500).json({ error: message });
      }
    });

    // Health check endpoint
    this.app.get("/health", (_req: Request, res: Response) => {
      const stats = this.pool.getStats();
      res.json({
        status: "ok",
        sessions: this.sessions.size,
        pool: stats,
      });
    });

    // List all tools
    this.app.get("/tools", (_req: Request, res: Response) => {
      const tools = this.registry.getAllTools().map((tool) => ({
        name: tool.name,
        description: tool.description,
        serverName: tool.serverName,
      }));
      res.json({ tools });
    });
  }

  async start(port?: number): Promise<void> {
    const portToUse = port ?? this.config.gateway?.port ?? 3000;
    const host = this.config.gateway?.host ?? "0.0.0.0";

    return new Promise((resolve) => {
      this.app.listen(portToUse, host, () => {
        console.log(`[gateway] MCP Gateway listening on http://${host}:${portToUse}`);
        console.log(`[gateway] MCP endpoint: http://${host}:${portToUse}/mcp`);
        console.log(`[gateway] SSE endpoint: http://${host}:${portToUse}/sse`);
        console.log(`[gateway] REST endpoint: http://${host}:${portToUse}/tools/call`);
        resolve();
      });
    });
  }

  async stop(): Promise<void> {
    await this.mcpServer.close();
    await this.pool.disconnectAll();
    console.log("[gateway] MCP Gateway stopped");
  }
}
