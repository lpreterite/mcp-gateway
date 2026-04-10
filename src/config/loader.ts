import { z } from "zod";
import { readFileSync } from "fs";
import { resolve } from "path";

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

export function loadConfig(configPath?: string): ValidatedConfig {
  const path = configPath || resolve(process.cwd(), "config/servers.json");
  const content = readFileSync(path, "utf-8");
  const raw = JSON.parse(content);

  const result = ConfigSchema.safeParse(raw);
  if (!result.success) {
    const errors = result.error.issues.map((i) => `${i.path.join(".")}: ${i.message}`);
    throw new Error(`Invalid configuration:\n${errors.join("\n")}`);
  }

  return result.data;
}
