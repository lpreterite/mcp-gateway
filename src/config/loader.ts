import { z } from "zod";
import { readFileSync, existsSync } from "fs";
import { resolve, join } from "path";
import { homedir } from "os";

// Validation schemas
export const ServerConfigSchema = z.object({
  name: z.string(),
  type: z.enum(["local", "remote"]),
  command: z.array(z.string()).optional(),
  url: z.string().optional(),
  enabled: z.boolean().default(true),
  env: z.record(z.string()).optional(),
  poolSize: z.number().optional(),
});

export const MappingConfigSchema = z.object({
  prefix: z.string(),
  stripPrefix: z.boolean().default(true),
  rename: z.record(z.string()).optional(),
});

export const ToolFilterSchema = z.object({
  include: z.array(z.string()).optional(),
  exclude: z.array(z.string()).optional(),
});

export const PoolConfigSchema = z.object({
  minConnections: z.number().min(1).default(1),
  maxConnections: z.number().min(1).default(5),
  acquireTimeout: z.number().min(1000).default(10000),
  idleTimeout: z.number().min(10000).default(60000),
  maxRetries: z.number().min(1).default(3),
});

export const GatewayConfigSchema = z.object({
  host: z.string().default("0.0.0.0"),
  port: z.number().min(1).max(65535).default(3000),
  cors: z.boolean().default(true),
});

export const ConfigSchema = z.object({
  gateway: GatewayConfigSchema.optional(),
  pool: PoolConfigSchema.optional(),
  servers: z.array(ServerConfigSchema),
  mapping: z.record(z.string(), MappingConfigSchema).optional(),
  toolFilters: z.record(z.string(), ToolFilterSchema).optional(),
});

export type ValidatedConfig = z.infer<typeof ConfigSchema>;

/**
 * Find config file path with priority:
 * 1. MCP_GATEWAY_CONFIG environment variable
 * 2. ~/.config/mcp-gateway/config.json (global install)
 * 3. ./config/servers.json (local development)
 */
function findConfigPath(): string | null {
  // 1. Environment variable
  const envPath = process.env.MCP_GATEWAY_CONFIG;
  if (envPath && existsSync(envPath)) {
    return envPath;
  }

  // 2. Global install config (~/.config/mcp-gateway/config.json)
  const globalConfig = join(homedir(), ".config", "mcp-gateway", "config.json");
  if (existsSync(globalConfig)) {
    return globalConfig;
  }

  // 3. Local development config
  const localConfig = resolve(process.cwd(), "config/servers.json");
  if (existsSync(localConfig)) {
    return localConfig;
  }

  return null;
}

export function loadConfig(configPath?: string): ValidatedConfig {
  let path: string;

  if (configPath) {
    path = configPath;
  } else {
    const foundPath = findConfigPath();
    if (!foundPath) {
      throw new Error(
        `Config file not found. Please create one of:\n` +
        `  - ~/.config/mcp-gateway/config.json (global install)\n` +
        `  - ./config/servers.json (local development)\n` +
        `  - Or set MCP_GATEWAY_CONFIG environment variable`
      );
    }
    path = foundPath;
  }

  console.error(`[config] Loading config from: ${path}`);

  const content = readFileSync(path, "utf-8");
  const raw = JSON.parse(content);

  const result = ConfigSchema.safeParse(raw);
  if (!result.success) {
    const errors = result.error.issues.map((i) => `${i.path.join(".")}: ${i.message}`);
    throw new Error(`Invalid configuration at ${path}:\n${errors.join("\n")}`);
  }

  return result.data;
}
