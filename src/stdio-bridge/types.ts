import type { Tool, ToolInfo, ToolCallResult } from "../mcp/types.js";

// Stdio bridge specific types
export interface BridgeConfig {
  gatewayUrl: string;
  stdioMode: boolean;
}

// JSON-RPC request/response types for stdio
export interface JsonRpcRequest {
  jsonrpc: "2.0";
  id: string | number | null;
  method: string;
  params?: Record<string, unknown>;
}

export interface JsonRpcResponse {
  jsonrpc: "2.0";
  id: string | number | null;
  result?: unknown;
  error?: {
    code: number;
    message: string;
    data?: unknown;
  };
}

export interface JsonRpcError {
  code: number;
  message: string;
  data?: unknown;
}

// Re-export commonly used types
export type { Tool, ToolInfo, ToolCallResult };
