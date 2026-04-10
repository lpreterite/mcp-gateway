import type { ToolInfo } from "../mcp/types.js";

export class ToolRegistry {
  private tools = new Map<string, ToolInfo>();

  registerTool(tool: ToolInfo): void {
    this.tools.set(tool.name, tool);
  }

  unregisterTool(name: string): void {
    this.tools.delete(name);
  }

  getTool(name: string): ToolInfo | undefined {
    return this.tools.get(name);
  }

  getAllTools(): ToolInfo[] {
    return Array.from(this.tools.values());
  }

  getToolsByServer(serverName: string): ToolInfo[] {
    return this.getAllTools().filter((t) => t.serverName === serverName);
  }

  hasTool(name: string): boolean {
    return this.tools.has(name);
  }

  clear(): void {
    this.tools.clear();
  }

  clearByServer(serverName: string): void {
    for (const [name, tool] of this.tools) {
      if (tool.serverName === serverName) {
        this.tools.delete(name);
      }
    }
  }
}
