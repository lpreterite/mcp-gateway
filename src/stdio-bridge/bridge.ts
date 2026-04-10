import { EventEmitter } from "events";
import type {
  JsonRpcRequest,
  JsonRpcResponse,
  BridgeConfig,
} from "./types.js";

/**
 * StdioBridge connects to an MCP Gateway via HTTP/SSE and exposes
 * all gateway tools via stdio for Claude Desktop.
 */
export class StdioBridge extends EventEmitter {
  private config: BridgeConfig;
  private sessionId: string | null = null;
  private abortController: AbortController | null = null;
  private connected = false;
  private pendingRequests = new Map<string | number, {
    resolve: (result: unknown) => void;
    reject: (error: Error) => void;
  }>();

  // Incremental ID for JSON-RPC requests sent to gateway
  private nextId = 1;

  constructor(config: BridgeConfig) {
    super();
    this.config = config;
  }

  /**
   * Connect to the gateway via SSE.
   * This method is non-blocking - it starts the SSE connection
   * and extracts sessionId asynchronously.
   */
  async connect(): Promise<void> {
    if (this.connected) {
      return;
    }

    try {
      // Establish SSE connection to gateway
      const url = new URL(this.config.gatewayUrl);
      const sseUrl = `${url.origin}/sse`;

      this.abortController = new AbortController();

      // Start SSE connection
      const response = await fetch(sseUrl, {
        signal: this.abortController.signal,
        headers: {
          Accept: "text/event-stream",
          "Cache-Control": "no-cache",
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to connect to gateway: ${response.status} ${response.statusText}`);
      }

      if (!response.body) {
        throw new Error("Gateway response has no body");
      }

      // Start reading SSE stream in background
      // Don't await - let it run asynchronously
      const reader = response.body.getReader();
      const decoder = new TextDecoder();

      // Read first chunk to get sessionId (use readAsyncIterator pattern)
      this.connectAndReadSSE(reader, decoder);

      // Mark as connected immediately to not block
      this.connected = true;
    } catch (error) {
      if (error instanceof Error && error.name === "AbortError") {
        throw new Error("Connection to gateway was aborted");
      }
      throw error;
    }
  }

  /**
   * Connect and read SSE stream in background, extracting sessionId
   */
  private async connectAndReadSSE(
    reader: ReadableStreamDefaultReader,
    decoder: TextDecoder
  ): Promise<void> {
    let buffer = "";
    let chunksRead = 0;

    try {
      while (true) {
        const { done, value } = await reader.read();
        chunksRead++;

        if (done) {
          break;
        }

        const chunk = decoder.decode(value, { stream: false });
        buffer += chunk;

        // Parse SSE events from buffer
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);

            // Try to parse as JSON first
            try {
              const message = JSON.parse(data);
              if (message.sessionId && !this.sessionId) {
                this.sessionId = message.sessionId;
                console.error(`[bridge] SSE session established (JSON): ${this.sessionId}`);
              }
              this.handleGatewayMessage(message);
            } catch {
              // Not JSON - could be endpoint URL like "/messages?sessionId=xxx"
              // Extract sessionId from URL
              const sessionIdMatch = data.match(/sessionId=([a-f0-9-]+)/i);
              if (sessionIdMatch && !this.sessionId) {
                this.sessionId = sessionIdMatch[1];
                console.error(`[bridge] SSE session established (URL): ${this.sessionId}`);
              }
            }
          }
        }

        // Stop after reading enough chunks to avoid infinite loop in case no sessionId
        if (chunksRead > 100) {
          console.error("[bridge] Warning: SSE read many chunks, stopping");
          break;
        }
      }
    } catch (error) {
      if (error instanceof Error && error.name !== "AbortError") {
        console.error("[bridge] SSE read error:", error);
      }
    }
  }

  /**
   * Handle incoming JSON-RPC messages from gateway
   */
  private handleGatewayMessage(message: JsonRpcResponse & { method?: string }): void {
    // If it's a response to one of our requests
    if (message.id !== undefined && message.id !== null) {
      const pending = this.pendingRequests.get(message.id);
      if (pending) {
        this.pendingRequests.delete(message.id);
        if (message.error) {
          pending.reject(new Error(message.error.message));
        } else {
          pending.resolve(message.result);
        }
      }
    }

    // Emit the message for other handlers
    this.emit("message", message);
  }

  /**
   * Send a JSON-RPC request to the gateway
   */
  async sendRequest(request: JsonRpcRequest): Promise<unknown> {
    // If not connected at all, throw error
    if (!this.connected) {
      throw new Error("Not connected to gateway");
    }

    // Wait for sessionId if not yet available
    if (!this.sessionId) {
      const startTime = Date.now();
      const timeout = 5000;
      while (!this.sessionId && Date.now() - startTime < timeout) {
        await new Promise((resolve) => setTimeout(resolve, 100));
      }
      if (!this.sessionId) {
        throw new Error("SSE session not established - could not get sessionId");
      }
    }

    const url = new URL(this.config.gatewayUrl);
    const messagesUrl = `${url.origin}/messages?sessionId=${encodeURIComponent(this.sessionId)}`;

    const id = this.nextId++;
    const payload = {
      ...request,
      id,
    };

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });

      // Set timeout for request
      const timeout = setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error(`Request ${request.method} timed out`));
        }
      }, 30000);

      this.pendingRequests.get(id)!.reject = (err) => {
        clearTimeout(timeout);
        reject(err);
      };

      this.pendingRequests.get(id)!.resolve = (result) => {
        clearTimeout(timeout);
        resolve(result);
      };

      // Send request to gateway
      fetch(messagesUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      }).then(async (response) => {
        if (!response.ok) {
          this.pendingRequests.delete(id);
          clearTimeout(timeout);
          reject(new Error(`Gateway request failed: ${response.status}`));
        }
        // Response comes via SSE, not here
      }).catch((error) => {
        this.pendingRequests.delete(id);
        clearTimeout(timeout);
        reject(error);
      });
    });
  }

  /**
   * List all tools from the gateway
   */
  async listTools(): Promise<Array<{
    name: string;
    description: string;
    inputSchema: Record<string, unknown>;
  }>> {
    const url = new URL(this.config.gatewayUrl);
    const toolsUrl = `${url.origin}/tools`;

    const response = await fetch(toolsUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch tools: ${response.status}`);
    }

    const data = await response.json();
    return data.tools || [];
  }

  /**
   * Call a tool on the gateway
   */
  async callTool(name: string, args: Record<string, unknown>): Promise<{
    content: Array<{ type: string; text?: string }>;
    isError?: boolean;
  }> {
    const result = await this.sendRequest({
      jsonrpc: "2.0",
      id: null,
      method: "tools/call",
      params: { name, arguments: args },
    });

    return result as { content: Array<{ type: string; text?: string }>; isError?: boolean };
  }

  /**
   * Disconnect from the gateway
   */
  async disconnect(): Promise<void> {
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }
    this.sessionId = null;
    this.connected = false;
    this.pendingRequests.clear();
  }

  isConnected(): boolean {
    return this.connected;
  }
}
