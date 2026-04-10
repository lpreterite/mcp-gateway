import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

// Tool definition from MCP server
export interface Tool {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
  annotations?: Record<string, unknown>;
}

// Tool info with server mapping
export interface ToolInfo {
  name: string;
  description: string;
  serverName: string;
  originalName: string;
  inputSchema?: Record<string, unknown>;
  annotations?: Record<string, unknown>;
}

// Server configuration from config file
export interface ServerConfig {
  name: string;
  type: "local" | "remote";
  command?: string[];
  url?: string;
  enabled: boolean;
  env?: Record<string, string>;
  poolSize?: number;
}

// Mapping configuration
export interface MappingConfig {
  prefix: string;
  stripPrefix: boolean;
  rename?: Record<string, string>;
}

// Tool filter configuration
export interface ToolFilterConfig {
  include?: string[];
  exclude?: string[];
}

// Connection pool configuration
export interface PoolConfig {
  minConnections: number;
  maxConnections: number;
  acquireTimeout: number;
  idleTimeout: number;
  maxRetries: number;
}

// Gateway server configuration
export interface GatewayConfig {
  host: string;
  port: number;
  cors: boolean;
}

// Full configuration
export interface Config {
  gateway: GatewayConfig;
  pool: PoolConfig;
  servers: ServerConfig[];
  mapping: Record<string, MappingConfig>;
  toolFilters: Record<string, ToolFilterConfig>;
}

// MCP client wrapper
export interface MCPClient {
  client: Client | null;
  transport: StdioClientTransport | null;
  config: ServerConfig;
  connected: boolean;
  lastUsed: number;
  connect(): Promise<void>;
  disconnect(): Promise<void>;
  listTools(): Promise<Tool[]>;
  callTool(name: string, args: Record<string, unknown>): Promise<ToolCallResult>;
  isConnected(): boolean;
  getName(): string;
  touch(): void;
}

// Tool call result
export interface ToolCallResult {
  content: Array<{ type: string; text?: string }>;
  isError?: boolean;
}
