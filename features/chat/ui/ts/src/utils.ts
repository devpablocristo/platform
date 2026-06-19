import type { ChatMessage, ChatResponse, ChatStreamEvent } from "./types";

export function responseChatId(response: ChatResponse | Record<string, unknown>): string | null {
  const direct = stringValue((response as ChatResponse).chatId);
  if (direct) return direct;
  return stringValue((response as Record<string, unknown>).chat_id);
}

export function responseTaskId(response: ChatResponse | Record<string, unknown>): string | null {
  const direct = stringValue((response as ChatResponse).taskId);
  if (direct) return direct;
  return stringValue((response as Record<string, unknown>).task_id);
}

export function responseRunId(response: ChatResponse | Record<string, unknown>): string | null {
  const direct = stringValue((response as ChatResponse).runId);
  if (direct) return direct;
  return stringValue((response as Record<string, unknown>).run_id);
}

export function normalizeResponse(raw: unknown): ChatResponse {
  if (!isRecord(raw)) {
    return { reply: String(raw ?? "") };
  }
  return {
    chatId: stringValue(raw.chatId) ?? stringValue(raw.chat_id) ?? undefined,
    taskId: stringValue(raw.taskId) ?? stringValue(raw.task_id) ?? undefined,
    runId: stringValue(raw.runId) ?? stringValue(raw.run_id) ?? undefined,
    agentId: stringValue(raw.agentId) ?? stringValue(raw.agent_id) ?? undefined,
    reply: stringValue(raw.reply) ?? "",
    blocks: arrayValue(raw.blocks),
    toolCalls: arrayValue(raw.toolCalls) ?? arrayValue(raw.tool_calls),
    pendingConfirmations: arrayValue(raw.pendingConfirmations) ?? arrayValue(raw.pending_confirmations),
    messages: normalizeMessages(arrayValue(raw.messages)),
  };
}

export function normalizeMessages(raw: unknown[] | undefined): ChatMessage[] | undefined {
  if (!raw) return undefined;
  return raw.map((item, index) => {
    if (!isRecord(item)) {
      return { id: `message-${index}`, role: "assistant", content: String(item ?? "") };
    }
    return {
      id: stringValue(item.id) ?? `message-${index}`,
      role: stringValue(item.role) ?? "assistant",
      content: stringValue(item.content) ?? stringValue(item.text) ?? "",
      timestamp: stringValue(item.timestamp) ?? stringValue(item.ts),
      blocks: arrayValue(item.blocks),
      toolCalls: arrayValue(item.toolCalls) ?? arrayValue(item.tool_calls),
      pendingConfirmations: arrayValue(item.pendingConfirmations) ?? arrayValue(item.pending_confirmations),
    };
  });
}

export function messageFromResponse(response: ChatResponse, id: string): ChatMessage {
  return {
    id,
    role: "assistant",
    content: response.reply,
    blocks: response.blocks,
    toolCalls: response.toolCalls,
    pendingConfirmations: response.pendingConfirmations,
  };
}

export function streamEventText(event: ChatStreamEvent): string {
  if (event.event === "text" && typeof event.data.content === "string") return event.data.content;
  return "";
}

export function streamEventChatId(event: ChatStreamEvent): string | null {
  return stringValue(event.data.chat_id) ?? stringValue(event.data.chatId);
}

export function streamEventToResponse(event: ChatStreamEvent): ChatResponse {
  return normalizeResponse(event.data);
}

export function summarizeToolCall(value: unknown): string {
  if (typeof value === "string") return value;
  if (!isRecord(value)) return String(value ?? "");
  return (
    stringValue(value.tool) ??
    stringValue(value.tool_name) ??
    stringValue(value.name) ??
    stringValue(value.operation) ??
    stringValue(value.capability) ??
    "tool"
  );
}

export function stringValue(value: unknown): string | null {
  return typeof value === "string" && value.trim() !== "" ? value : null;
}

export function arrayValue(value: unknown): unknown[] | undefined {
  return Array.isArray(value) ? value : undefined;
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
