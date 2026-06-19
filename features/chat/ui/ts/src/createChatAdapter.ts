import type { ChatAdapter, ChatRequest, ChatResponse, ChatStreamEvent } from "./types";
import { normalizeResponse } from "./utils";

export type CreateChatAdapterOptions = {
  chatPath: string;
  conversationsPath?: string;
  fetcher?: typeof fetch;
  headers?: HeadersInit | (() => HeadersInit);
  mapRequest?: (input: ChatRequest) => unknown;
  mapResponse?: (response: unknown) => ChatResponse;
};

export function createChatAdapter({
  chatPath,
  conversationsPath,
  fetcher = fetch,
  headers,
  mapRequest = defaultMapRequest,
  mapResponse = normalizeResponse,
}: CreateChatAdapterOptions): ChatAdapter {
  const resolvedHeaders = () => ({
    "Content-Type": "application/json",
    ...(typeof headers === "function" ? headers() : headers ?? {}),
  });

  return {
    async sendMessage(input) {
      const response = await fetcher(chatPath, {
        method: "POST",
        headers: resolvedHeaders(),
        body: JSON.stringify(mapRequest(input)),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || `chat request failed: ${response.status}`);
      }
      return mapResponse(await response.json());
    },
    async listConversations(limit = 50) {
      if (!conversationsPath) return { items: [] };
      const url = withLimit(conversationsPath, limit);
      const response = await fetcher(url, { headers: resolvedHeaders() });
      if (!response.ok) {
        throw new Error((await response.text()) || `conversation list failed: ${response.status}`);
      }
      return response.json();
    },
    async getConversation(id) {
      if (!conversationsPath) {
        throw new Error("conversation endpoint is not configured");
      }
      const response = await fetcher(`${conversationsPath.replace(/\/$/, "")}/${encodeURIComponent(id)}`, {
        headers: resolvedHeaders(),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || `conversation detail failed: ${response.status}`);
      }
      return response.json();
    },
  };
}

export async function parseSseStream(
  response: Response,
  onEvent: (event: ChatStreamEvent) => void,
): Promise<void> {
  if (!response.body) {
    throw new Error("stream response has no body");
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const chunks = buffer.split(/\n\n+/);
    buffer = chunks.pop() ?? "";
    for (const chunk of chunks) {
      const event = parseSseEvent(chunk);
      if (event) onEvent(event);
    }
  }
  if (buffer.trim()) {
    const event = parseSseEvent(buffer);
    if (event) onEvent(event);
  }
}

function defaultMapRequest(input: ChatRequest): unknown {
  return {
    message: input.message,
    chat_id: input.chatId ?? null,
    task_id: input.taskId ?? null,
    agent_id: input.agentId ?? undefined,
    product_surface: input.productSurface ?? undefined,
    route_hint: input.routeHint ?? undefined,
    confirmed_actions: input.confirmedActions ?? [],
    workspace: input.workspace ?? undefined,
  };
}

function withLimit(path: string, limit: number): string {
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}limit=${encodeURIComponent(String(limit))}`;
}

function parseSseEvent(raw: string): ChatStreamEvent | null {
  let event = "message";
  const dataLines: string[] = [];
  for (const line of raw.split(/\r?\n/)) {
    if (line.startsWith("event:")) event = line.slice(6).trim();
    if (line.startsWith("data:")) dataLines.push(line.slice(5).trim());
  }
  if (dataLines.length === 0) return null;
  try {
    return { event, data: JSON.parse(dataLines.join("\n")) };
  } catch {
    return { event, data: { content: dataLines.join("\n") } };
  }
}
