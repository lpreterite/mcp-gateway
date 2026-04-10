#!/usr/bin/env node

import { StdioBridge } from "./bridge.js";
import type { JsonRpcRequest, BridgeConfig } from "./types.js";

// Default gateway URL
const DEFAULT_GATEWAY_URL = "http://localhost:3000/sse";

/**
 * Stdio Bridge Entry Point
 *
 * This process acts as an MCP server via stdio, forwarding all tool
 * calls to an MCP Gateway via HTTP/SSE.
 */
async function main(): Promise<void> {
  // Get gateway URL from command line args or environment
  const gatewayUrl = process.argv[2] || process.env.MCP_GATEWAY_URL || DEFAULT_GATEWAY_URL;

  console.error("[bridge] Starting MCP Gateway Stdio Bridge...");
  console.error(`[bridge] Gateway URL: ${gatewayUrl}`);

  const bridge = new StdioBridge({
    gatewayUrl,
    stdioMode: true,
  });

  // Handle process termination
  const shutdown = async (): Promise<void> => {
    console.error("[bridge] Shutting down...");
    await bridge.disconnect();
    process.exit(0);
  };

  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);

  try {
    // Connect to gateway
    await bridge.connect();
    console.error("[bridge] Connected to gateway");

    // Fetch and log available tools
    const tools = await bridge.listTools();
    console.error(`[bridge] Available tools: ${tools.map((t) => t.name).join(", ")}`);

    // Send initialize response to stdio client
    const initResponse = {
      jsonrpc: "2.0",
      id: null,
      result: {
        protocolVersion: "2024-11-05",
        capabilities: {
          tools: {},
        },
        serverInfo: {
          name: "mcp-gateway-bridge",
          version: "1.0.0",
        },
      },
    };
    process.stdout.write(JSON.stringify(initResponse) + "\n");

    // Send tools/list response
    const toolsResponse = {
      jsonrpc: "2.0",
      id: null,
      result: {
        tools: tools.map((tool) => ({
          name: tool.name,
          description: tool.description,
          inputSchema: tool.inputSchema,
        })),
      },
    };
    process.stdout.write(JSON.stringify(toolsResponse) + "\n");

    // Handle incoming stdio messages
    process.stdin.on("data", async (data: Buffer) => {
      try {
        const line = data.toString().trim();
        if (!line) return;

        const request: JsonRpcRequest = JSON.parse(line);

        // Skip notifications (no id)
        if (request.id === undefined || request.id === null) {
          // Handle notifications if needed
          return;
        }

        const { method, params, id } = request;

        if (method === "tools/call") {
          const callParams = params as { name?: string; arguments?: Record<string, unknown> } | undefined;
          const name = callParams?.name;
          const args = callParams?.arguments || {};

          if (!name) {
            const response = {
              jsonrpc: "2.0",
              id,
              error: {
                code: -32602,
                message: "Missing tool name",
              },
            };
            process.stdout.write(JSON.stringify(response) + "\n");
            return;
          }

          try {
            const result = await bridge.callTool(name, args);
            const response = {
              jsonrpc: "2.0",
              id,
              result,
            };
            process.stdout.write(JSON.stringify(response) + "\n");
          } catch (error) {
            const response = {
              jsonrpc: "2.0",
              id,
              error: {
                code: -32603,
                message: error instanceof Error ? error.message : String(error),
              },
            };
            process.stdout.write(JSON.stringify(response) + "\n");
          }
        } else if (method === "initialize") {
          // Already handled above, just acknowledge
          const response = {
            jsonrpc: "2.0",
            id,
            result: {
              protocolVersion: "2024-11-05",
              capabilities: {
                tools: {},
              },
              serverInfo: {
                name: "mcp-gateway-bridge",
                version: "1.0.0",
              },
            },
          };
          process.stdout.write(JSON.stringify(response) + "\n");
        } else if (method === "tools/list") {
          // Re-send tools list
          const response = {
            jsonrpc: "2.0",
            id,
            result: {
              tools: tools.map((tool) => ({
                name: tool.name,
                description: tool.description,
                inputSchema: tool.inputSchema,
              })),
            },
          };
          process.stdout.write(JSON.stringify(response) + "\n");
        } else {
          // Unknown method
          const response = {
            jsonrpc: "2.0",
            id,
            error: {
              code: -32601,
              message: `Method not found: ${method}`,
            },
          };
          process.stdout.write(JSON.stringify(response) + "\n");
        }
      } catch (error) {
        console.error("[bridge] Error handling request:", error);
      }
    });

    // Handle stdin close
    process.stdin.on("end", () => {
      console.error("[bridge] stdin closed");
      shutdown();
    });

  } catch (error) {
    console.error("[bridge] Failed to start:", error);
    process.exit(1);
  }
}

main();
