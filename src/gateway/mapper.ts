import type { MappingConfig, ToolFilterConfig } from "../mcp/types.js";

interface MappingRule {
  serverName: string;
  prefix: string;
  stripPrefix: boolean;
  renameMap: Map<string, string>;
}

export class ToolMapper {
  private rules = new Map<string, MappingRule>();
  private reverseIndex = new Map<string, string>();
  private filters = new Map<string, ToolFilterConfig>();

  constructor(mapping: Record<string, MappingConfig> = {}, toolFilters: Record<string, ToolFilterConfig> = {}) {
    for (const [serverName, config] of Object.entries(mapping)) {
      this.addMapping(serverName, config);
    }
    for (const [serverName, filter] of Object.entries(toolFilters)) {
      if (filter) {
        this.filters.set(serverName, filter);
      }
    }
  }

  addMapping(serverName: string, config: MappingConfig): void {
    const renameMap = new Map<string, string>();
    if (config.rename) {
      for (const [from, to] of Object.entries(config.rename)) {
        renameMap.set(from, to);
      }
    }

    const rule: MappingRule = {
      serverName,
      prefix: config.prefix,
      stripPrefix: config.stripPrefix,
      renameMap,
    };

    this.rules.set(serverName, rule);
    this.reverseIndex.set(config.prefix.toLowerCase(), serverName);
  }

  getServerForTool(toolName: string): string | undefined {
    for (const [prefix, serverName] of this.reverseIndex) {
      if (toolName.toLowerCase().startsWith(prefix.toLowerCase() + "_")) {
        return serverName;
      }
      if (toolName.toLowerCase() === prefix.toLowerCase()) {
        return serverName;
      }
    }
    return undefined;
  }

  getOriginalToolName(gatewayToolName: string, serverName: string): string | undefined {
    const rule = this.rules.get(serverName);
    if (!rule) return undefined;

    if (rule.stripPrefix) {
      const prefix = rule.prefix + "_";
      if (gatewayToolName.toLowerCase().startsWith(prefix.toLowerCase())) {
        return gatewayToolName.slice(prefix.length);
      }
      return gatewayToolName;
    }

    const originalName = rule.renameMap.get(gatewayToolName);
    if (originalName) return originalName;
    return gatewayToolName;
  }

  getGatewayToolName(originalName: string, serverName: string): string {
    const rule = this.rules.get(serverName);
    if (!rule) return originalName;

    if (rule.stripPrefix) {
      return `${rule.prefix}_${originalName}`;
    }
    return originalName;
  }

  shouldIncludeTool(serverName: string, toolName: string): boolean {
    const filter = this.filters.get(serverName);
    if (!filter) return true;

    if (filter.include && filter.include.length > 0) {
      return filter.include.includes(toolName);
    }

    if (filter.exclude && filter.exclude.length > 0) {
      return !filter.exclude.includes(toolName);
    }

    return true;
  }

  getAllPrefixes(): string[] {
    return Array.from(this.rules.values()).map((r) => r.prefix);
  }

  getRuleForServer(serverName: string): MappingRule | undefined {
    return this.rules.get(serverName);
  }
}
