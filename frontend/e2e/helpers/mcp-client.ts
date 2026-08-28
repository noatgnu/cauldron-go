/**
 * Minimal client for the Wails v3 built-in MCP server (`-tags mcp` / `WAILS_MCP=1`).
 *
 * The server speaks plain JSON-RPC 2.0 over a single HTTP POST endpoint (the MCP
 * "streamable HTTP transport"). This app's instance does not require session
 * negotiation for `tools/list`/`tools/call`, so this client sends bare JSON-RPC
 * requests rather than depending on `@modelcontextprotocol/sdk`.
 */

const MCP_URL = process.env['MCP_URL'] || 'http://127.0.0.1:9099/mcp';

let nextId = 1;

interface McpToolResult {
  content: Array<{ type: string; text: string }>;
  isError: boolean;
}

async function rpc(method: string, params?: object): Promise<any> {
  const response = await fetch(MCP_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json, text/event-stream',
    },
    body: JSON.stringify({ jsonrpc: '2.0', id: nextId++, method, params }),
  });
  const body = await response.json();
  if (body.error) {
    throw new Error(`MCP ${method} failed: ${body.error.message}`);
  }
  return body.result;
}

function parseToolResult(result: McpToolResult): any {
  const text = result.content.map(c => c.text).join('');
  if (result.isError) {
    throw new Error(`MCP tool call failed: ${text}`);
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

export async function callTool(name: string, args: Record<string, any> = {}): Promise<any> {
  const result = await rpc('tools/call', { name, arguments: args });
  return parseToolResult(result);
}

export async function listTools(): Promise<string[]> {
  const result = await rpc('tools/list');
  return result.tools.map((t: { name: string }) => t.name);
}

export async function appInfo(): Promise<any> {
  return callTool('app_info');
}

export async function waitForWindow(timeoutMs = 30000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const info = await appInfo();
      const win = info.windows?.[0];
      if (win && win.visible) return true;
    } catch {
      // server not up yet
    }
    await new Promise(r => setTimeout(r, 1000));
  }
  return false;
}

export async function navigate(hashRoute: string): Promise<void> {
  const route = hashRoute.startsWith('#') ? hashRoute : `#${hashRoute}`;
  await callTool('js_eval', { js: `window.location.hash = ${JSON.stringify(route)};` });
}

export async function getUrl(): Promise<string> {
  return callTool('js_eval', { js: 'return window.location.href;' });
}

export async function click(selector: string): Promise<void> {
  await callTool('mouse_click', { selector });
}

export async function fill(selector: string, value: string): Promise<void> {
  await click(selector);
  await callTool('keyboard_press', { key: 'a', modifiers: ['ctrl'] });
  await callTool('keyboard_type', { text: value });
}

export async function domQuery(selector: string, limit = 25): Promise<any> {
  return callTool('dom_query', { selector, limit });
}

export async function elementExists(selector: string): Promise<boolean> {
  const result = await domQuery(selector, 1);
  return result.count > 0;
}

export async function elementText(selector: string): Promise<string | undefined> {
  const result = await domQuery(selector, 1);
  return result.elements?.[0]?.text;
}

export async function waitForElement(selector: string, timeoutMs = 15000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (await elementExists(selector)) return true;
    await new Promise(r => setTimeout(r, 500));
  }
  return false;
}

export async function callBoundMethod(name: string, ...args: any[]): Promise<any> {
  return callTool('call_bound_method', { name, args });
}

export async function waitForEvent(name: string, timeoutMs = 30000): Promise<any> {
  return callTool('wait_for_event', { name, timeout_ms: timeoutMs });
}

export async function emitEvent(name: string, data?: any): Promise<void> {
  await callTool('emit_event', { name, data });
}
