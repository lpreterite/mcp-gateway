import express, { Request, Response } from "express";
import cors from "cors";
import { z } from "zod";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
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

// Store transports by session ID
const transports: Record<string, SSEServerTransport> = {};

// Create a new MCP server instance for each session
function createGatewayServer(
  pool: MCPConnectionPool,
  registry: ToolRegistry,
  mapper: ToolMapper
): Server {
  const server = new Server(
    { name: "mcp-gateway", version: "1.0.0" },
    { capabilities: { tools: {} } }
  );

  server.setRequestHandler(ListToolsRequestSchema, async () => {
    const tools = registry.getAllTools().map((tool) => ({
      name: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema || { type: "object", properties: {} },
      annotations: tool.annotations,
    }));
    return { tools };
  });

  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args = {} } = request.params;

    try {
      const tool = registry.getTool(name);
      if (!tool) {
        throw new Error(`Tool ${name} not found`);
      }

      const serverName = tool.serverName;
      const originalName = mapper.getOriginalToolName(name, serverName) || tool.originalName;

      const result = await pool.callTool(serverName, originalName, args as Record<string, unknown>);
      return result as { content: Array<{ type: string; text?: string }>; isError?: boolean };
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      return {
        content: [{ type: "text", text: `Error: ${message}` }],
        isError: true,
      };
    }
  });

  return server;
}

export class MCPGatewayServer {
  private app: express.Application;
  private pool: MCPConnectionPool;
  private registry: ToolRegistry;
  private mapper: ToolMapper;
  private config: ValidatedConfig;

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

    this.setupRoutes();
  }

  private setupRoutes(): void {
    // SSE endpoint for establishing the stream (GET)
    this.app.get("/sse", async (req: Request, res: Response) => {
      console.log(`[gateway] SSE connection from ${req.ip}`);

      try {
        const transport = new SSEServerTransport("/messages", res, {
          enableDnsRebindingProtection: false,
        });

        const sessionId = transport.sessionId;
        transports[sessionId] = transport;

        transport.onclose = () => {
          console.log(`[gateway] SSE transport closed for session ${sessionId}`);
          delete transports[sessionId];
        };

        const server = createGatewayServer(this.pool, this.registry, this.mapper);
        await server.connect(transport);

        console.log(`[gateway] Established SSE stream with session ID: ${sessionId}`);
      } catch (error) {
        console.error("[gateway] Error establishing SSE stream:", error);
        if (!res.headersSent) {
          res.status(500).send("Error establishing SSE stream");
        }
      }
    });

    // Messages endpoint for receiving client JSON-RPC requests (POST)
    this.app.post("/messages", async (req: Request, res: Response) => {
      const sessionId = req.query.sessionId as string;

      if (!sessionId) {
        console.error("[gateway] No session ID provided in request URL");
        res.status(400).send("Missing sessionId parameter");
        return;
      }

      const transport = transports[sessionId];
      if (!transport) {
        console.error(`[gateway] No active transport found for session ID: ${sessionId}`);
        res.status(404).send("Session not found");
        return;
      }

      try {
        await transport.handlePostMessage(req, res, req.body);
      } catch (error) {
        console.error("[gateway] Error handling request:", error);
        if (!res.headersSent) {
          res.status(500).send("Error handling request");
        }
      }
    });

    // REST endpoint for tool calls (backward compatibility)
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
        sessions: Object.keys(transports).length,
        pool: stats,
      });
    });

    // List all tools (REST)
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
        console.log(`[gateway] SSE endpoint: http://${host}:${portToUse}/sse`);
        console.log(`[gateway] Messages endpoint: http://${host}:${portToUse}/messages`);
        console.log(`[gateway] REST endpoint: http://${host}:${portToUse}/tools/call`);
        resolve();
      });
    });
  }

  async stop(): Promise<void> {
    // Close all active transports
    for (const sessionId in transports) {
      try {
        await transports[sessionId].close();
        delete transports[sessionId];
      } catch (error) {
        console.error(`[gateway] Error closing transport for session ${sessionId}:`, error);
      }
    }
    await this.pool.disconnectAll();
    console.log("[gateway] MCP Gateway stopped");
  }
}
