import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import type { ServerConfig, Tool, ToolCallResult } from "./types.js";

export class MCPClientConnection {
  client: Client | null = null;
  transport: StdioClientTransport | null = null;
  config: ServerConfig;
  connected = false;
  lastUsed = 0;

  constructor(config: ServerConfig) {
    this.config = config;
  }

  async connect(): Promise<void> {
    if (this.connected || !this.config.command) {
      return;
    }

    const env: Record<string, string> = {};
    for (const [key, value] of Object.entries(process.env)) {
      if (value !== undefined) {
        env[key] = value;
      }
    }

    if (this.config.env) {
      for (const [key, value] of Object.entries(this.config.env)) {
        if (value !== undefined) {
          env[key] = value;
        }
      }
    }

    this.transport = new StdioClientTransport({
      command: this.config.command[0],
      args: this.config.command.slice(1),
      env,
    });

    this.client = new Client(
      { name: `gateway-${this.config.name}`, version: "1.0.0" },
      { capabilities: {} }
    );

    await this.client.connect(this.transport);
    this.connected = true;
    this.lastUsed = Date.now();
  }

  async disconnect(): Promise<void> {
    if (this.client) {
      await this.client.close();
      this.client = null;
      this.transport = null;
      this.connected = false;
    }
  }

  async listTools(): Promise<Tool[]> {
    if (!this.client) {
      throw new Error(`Server ${this.config.name} not connected`);
    }

    const response = await this.client.listTools();
    return (response.tools || []) as Tool[];
  }

  async callTool(name: string, args: Record<string, unknown>): Promise<ToolCallResult> {
    if (!this.client) {
      throw new Error(`Server ${this.config.name} not connected`);
    }

    this.lastUsed = Date.now();
    const response = await this.client.callTool({
      name,
      arguments: args,
    });
    return response as ToolCallResult;
  }

  isConnected(): boolean {
    return this.connected;
  }

  getName(): string {
    return this.config.name;
  }

  touch(): void {
    this.lastUsed = Date.now();
  }
}
