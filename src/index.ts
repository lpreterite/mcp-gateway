import { execSync, spawn } from "child_process";
import { existsSync, mkdirSync, writeFileSync, readFileSync, rmSync } from "fs";
import { resolve } from "path";

interface ToolInfo {
  name: string;
  description: string;
  serverName: string;
  originalName: string;
}

interface ServerInfo {
  name: string;
  status: string;
  transport: string;
  tools: ToolInfo[];
}

interface Manifest {
  servers: ServerInfo[];
  cliPaths: Record<string, string>;
  generatedAt: string;
}

const CLI_DIR = resolve(process.env.HOME || "~", ".mcporter-gateway", "clis");
const MANIFEST_PATH = resolve(process.env.HOME || "~", ".mcporter-gateway", "manifest.json");

export class MCPorterGateway {
  private manifest: Manifest | null = null;

  async initialize(): Promise<void> {
    console.error("[gateway] Initializing MCPorter Gateway...");

    if (!existsSync(CLI_DIR)) {
      mkdirSync(CLI_DIR, { recursive: true });
    }

    const servers = await this.discoverServers();
    await this.generateCLIs(servers);

    const cliPaths = this.getCliPathsFromDisk(servers);

    this.manifest = {
      servers,
      cliPaths,
      generatedAt: new Date().toISOString(),
    };

    this.saveManifest();
    console.error(`[gateway] Ready with ${servers.reduce((acc, s) => acc + s.tools.length, 0)} tools`);
  }

  private async discoverServers(): Promise<ServerInfo[]> {
    console.error("[gateway] Discovering servers from mcporter...");

    const output = execSync("mcporter list --json 2>&1", {
      encoding: "utf-8",
      maxBuffer: 50 * 1024 * 1024,
    });

    const jsonMatch = output.match(/\{[\s\S]*\}/);
    if (!jsonMatch) {
      throw new Error("No JSON found in mcporter output");
    }
    const data = JSON.parse(jsonMatch[0]);

    const servers: ServerInfo[] = [];

    for (const server of data.servers || []) {
      if (server.status !== "ok") {
        console.error(`[gateway] Skipping ${server.name} (status: ${server.status})`);
        continue;
      }

      if (!server.transport?.toLowerCase().includes("stdio")) {
        console.error(`[gateway] Skipping ${server.name} (non-stdio: ${server.transport})`);
        continue;
      }

      const tools: ToolInfo[] = (server.tools || []).map((t: { name: string; description: string }) => ({
        name: `${server.name}_${t.name}`,
        originalName: t.name,
        description: t.description,
        serverName: server.name,
      }));

      servers.push({
        name: server.name,
        status: server.status,
        transport: server.transport,
        tools,
      });

      console.error(`[gateway] Found ${server.name}: ${tools.length} tools`);
    }

    return servers;
  }

  private async generateCLIs(servers: ServerInfo[]): Promise<void> {
    console.error("[gateway] Generating CLIs...");

    rmSync(CLI_DIR, { recursive: true, force: true });
    mkdirSync(CLI_DIR, { recursive: true });

    for (const server of servers) {
      const cliPath = resolve(CLI_DIR, server.name);

      try {
        console.error(`[gateway] Generating CLI for ${server.name}...`);
        execSync(`mcporter generate-cli --server ${server.name} --compile ${cliPath} 2>/dev/null`, {
          timeout: 120000,
        });
        console.error(`[gateway] Generated: ${cliPath}`);
      } catch (error) {
        console.error(`[gateway] Failed to generate CLI for ${server.name}:`, error);
      }
    }
  }

  private getCliPathsFromDisk(servers: ServerInfo[]): Record<string, string> {
    const paths: Record<string, string> = {};
    for (const server of servers) {
      const cliPath = resolve(CLI_DIR, server.name);
      if (existsSync(cliPath)) {
        paths[server.name] = cliPath;
      }
    }
    return paths;
  }

  private saveManifest(): void {
    writeFileSync(MANIFEST_PATH, JSON.stringify(this.manifest, null, 2));
    console.error(`[gateway] Manifest saved to ${MANIFEST_PATH}`);
  }

  async callTool(toolName: string, args: Record<string, unknown>): Promise<{ content: Array<{ type: string; text?: string }>; isError?: boolean }> {
    const server = this.manifest?.servers.find((s) =>
      s.tools.some((t) => t.name === toolName)
    );
    const tool = server?.tools.find((t) => t.name === toolName);

    if (!server || !tool) {
      throw new Error(`Tool ${toolName} not found`);
    }

    const cliPath = this.manifest?.cliPaths[server.name];
    if (!cliPath) {
      throw new Error(`CLI for ${server.name} not found`);
    }

    const commandArgs = this.buildArgs(tool.originalName, args);

    console.error(`[gateway] Calling: ${server.name}/${tool.originalName}`);

    return new Promise((resolve) => {
      const child = spawn(cliPath, commandArgs, {
        stdio: ["pipe", "pipe", "pipe"],
      });

      let stdout = "";
      let stderr = "";

      child.stdout?.on("data", (data) => {
        stdout += data.toString();
      });

      child.stderr?.on("data", (data) => {
        stderr += data.toString();
      });

      child.on("close", (code) => {
        resolve({
          content: [{ type: "text", text: code === 0 ? stdout : stderr || stdout }],
          isError: code !== 0,
        });
      });

      child.on("error", (error) => {
        resolve({
          content: [{ type: "text", text: error.message }],
          isError: true,
        });
      });
    });
  }

  private buildArgs(toolName: string, args: Record<string, unknown>): string[] {
    const subcommand = toolName.replace(/_/g, "-");
    const result: string[] = [subcommand];

    for (const [key, value] of Object.entries(args)) {
      const flag = `--${key.replace(/_/g, "-")}`;

      if (value === true) {
        result.push(flag);
      } else if (value === false) {
        // Skip false booleans
      } else if (Array.isArray(value)) {
        result.push(flag, JSON.stringify(value));
      } else if (value !== null && value !== undefined) {
        result.push(flag, String(value));
      }
    }

    return result;
  }

  getTools(): ToolInfo[] {
    return this.manifest?.servers.flatMap((s) => s.tools) || [];
  }
}

async function main() {
  const gateway = new MCPorterGateway();
  await gateway.initialize();

  let buffer = "";

  process.stdin.on("data", async (chunk: string) => {
    buffer += chunk;
    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";

    for (const line of lines) {
      if (!line.trim()) continue;

      try {
        const request = JSON.parse(line);

        if (request.method === "initialize") {
          process.stdout.write(
            JSON.stringify({
              jsonrpc: "2.0",
              id: request.id,
              result: {
                protocolVersion: "2024-11-05",
                capabilities: { tools: {} },
                serverInfo: { name: "mcporter-gateway", version: "1.0.0" },
              },
            }) + "\n"
          );
        } else if (request.method === "tools/list") {
          process.stdout.write(
            JSON.stringify({
              jsonrpc: "2.0",
              id: request.id,
              result: {
                tools: gateway.getTools().map((t) => ({
                  name: t.name,
                  description: t.description,
                  inputSchema: { type: "object", properties: {} },
                })),
              },
            }) + "\n"
          );
        } else if (request.method === "tools/call") {
          const { name, arguments: args = {} } = request.params ?? {};
          try {
            const result = await gateway.callTool(name, args);
            process.stdout.write(
              JSON.stringify({ jsonrpc: "2.0", id: request.id, result }) + "\n"
            );
          } catch (error) {
            process.stdout.write(
              JSON.stringify({
                jsonrpc: "2.0",
                id: request.id,
                error: { code: -32603, message: String(error) },
              }) + "\n"
            );
          }
        }
      } catch {
        // Ignore parse errors for now
      }
    }
  });
}

main().catch((error) => {
  console.error("[gateway] Fatal:", error);
  process.exit(1);
});
