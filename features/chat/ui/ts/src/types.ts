import type { ReactNode } from "react";

export type ChatRole = "user" | "assistant" | "system" | "tool";

export type ChatMessage = {
  id?: string;
  role: ChatRole | string;
  content: string;
  timestamp?: string | null;
  blocks?: unknown[];
  toolCalls?: unknown[];
  pendingConfirmations?: unknown[];
  metadata?: Record<string, unknown>;
  meta?: ReactNode;
};

export type ChatConversationSummary = {
  id: string;
  title?: string;
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
  messageCount?: number;
  message_count?: number;
  productSurface?: string;
  product_surface?: string;
};

export type ChatConversationDetail = {
  id: string;
  title?: string;
  messages: ChatMessage[];
  createdAt?: string;
  updatedAt?: string;
  created_at?: string;
  updated_at?: string;
};

export type ChatRequest = {
  message: string;
  chatId?: string | null;
  taskId?: string | null;
  agentId?: string | null;
  productSurface?: string;
  routeHint?: string;
  confirmedActions?: string[];
  workspace?: Record<string, unknown>;
};

export type ChatResponse = {
  chatId?: string;
  taskId?: string;
  runId?: string;
  agentId?: string;
  reply: string;
  blocks?: unknown[];
  toolCalls?: unknown[];
  pendingConfirmations?: unknown[];
  messages?: ChatMessage[];
};

export type ChatStreamDraft = {
  text: string;
  activity: string[];
};

export type ChatStreamEvent =
  | { event: "start"; data: Record<string, unknown> }
  | { event: "text"; data: { content?: string } & Record<string, unknown> }
  | { event: "tool_call"; data: Record<string, unknown> }
  | { event: "tool_result"; data: Record<string, unknown> }
  | { event: "done"; data: Record<string, unknown> }
  | { event: "error"; data: { message?: string } & Record<string, unknown> }
  | { event: string; data: Record<string, unknown> };

export type ChatAdapter = {
  sendMessage(input: ChatRequest): Promise<ChatResponse>;
  streamMessage?: (
    input: ChatRequest,
    onEvent: (event: ChatStreamEvent) => void,
    signal?: AbortSignal,
  ) => Promise<void>;
  listConversations?: (limit?: number) => Promise<{ items: ChatConversationSummary[] }>;
  getConversation?: (id: string) => Promise<ChatConversationDetail>;
};

export type ChatLabels = {
  title?: ReactNode;
  lead?: ReactNode;
  newConversation?: ReactNode;
  conversations?: ReactNode;
  emptyConversations?: ReactNode;
  emptyThread?: ReactNode;
  inputPlaceholder?: string;
  send?: ReactNode;
  sending?: ReactNode;
  loadingHistory?: ReactNode;
  confirmPending?: ReactNode;
  pendingPrefix?: ReactNode;
  tools?: ReactNode;
  refresh?: ReactNode;
};
