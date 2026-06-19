import type { Dispatch, SetStateAction } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  ChatAdapter,
  ChatConversationSummary,
  ChatMessage,
  ChatRequest,
  ChatResponse,
  ChatStreamDraft,
  ChatStreamEvent,
} from "./types";
import {
  messageFromResponse,
  normalizeMessages,
  responseChatId,
  responseRunId,
  responseTaskId,
  streamEventChatId,
  streamEventText,
  streamEventToResponse,
  summarizeToolCall,
} from "./utils";

export type UseChatSessionOptions = {
  adapter: ChatAdapter;
  initialChatId?: string | null;
  baseRequest?: Partial<ChatRequest>;
  loadInitialConversation?: boolean;
  conversationLimit?: number;
  nowLabel?: () => string;
  onAssistantDone?: (message: ChatMessage, response: ChatResponse) => void;
};

export function useChatSession({
  adapter,
  initialChatId = null,
  baseRequest = {},
  loadInitialConversation = true,
  conversationLimit = 30,
  nowLabel,
  onAssistantDone,
}: UseChatSessionOptions) {
  const [chatId, setChatId] = useState<string | null>(initialChatId);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [runId, setRunId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<ChatConversationSummary[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [error, setError] = useState("");
  const [streamDraft, setStreamDraft] = useState<ChatStreamDraft | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const pendingConfirmations = useMemo(
    () => messages.flatMap((message) => message.pendingConfirmations ?? []).filter(Boolean),
    [messages],
  );

  const refreshConversations = useCallback(async () => {
    if (!adapter.listConversations) return;
    try {
      const response = await adapter.listConversations(conversationLimit);
      setConversations(response.items ?? []);
    } catch {
      setConversations([]);
    }
  }, [adapter, conversationLimit]);

  useEffect(() => {
    void refreshConversations();
  }, [refreshConversations]);

  useEffect(() => {
    if (!loadInitialConversation || chatId || conversations.length === 0) return;
    const latest = conversations[0];
    if (latest?.id) {
      void loadConversation(latest.id);
    }
    // loadConversation depends on conversations state; keep this one-shot shape simple.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadInitialConversation, conversations.length]);

  const loadConversation = useCallback(
    async (id: string) => {
      if (!adapter.getConversation) return;
      abortRef.current?.abort();
      setLoadingHistory(true);
      setError("");
      try {
        const detail = await adapter.getConversation(id);
        setChatId(detail.id);
        setMessages(normalizeMessages(detail.messages as unknown[]) ?? []);
      } catch (err) {
        setError(errorMessage(err, "No se pudo cargar la conversación"));
      } finally {
        setLoadingHistory(false);
      }
    },
    [adapter],
  );

  const newChat = useCallback(() => {
    abortRef.current?.abort();
    setChatId(null);
    setTaskId(null);
    setRunId(null);
    setMessages([]);
    setInput("");
    setError("");
    setStreamDraft(null);
  }, []);

  const send = useCallback(
    async (overrides?: Partial<ChatRequest>) => {
      const text = (overrides?.message ?? input).trim();
      const confirmedActions = overrides?.confirmedActions ?? [];
      if (!text && confirmedActions.length === 0) return;
      if (loading) return;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      setLoading(true);
      setError("");
      setInput("");
      const userMessage: ChatMessage = {
        id: nextID("user"),
        role: "user",
        content: text || "Confirmar acciones",
        timestamp: nowLabel?.(),
      };
      setMessages((prev) => [...prev, userMessage]);
      const request: ChatRequest = {
        ...baseRequest,
        ...overrides,
        message: text,
        chatId: overrides?.chatId ?? chatId,
        taskId: overrides?.taskId ?? taskId,
        confirmedActions,
      };

      try {
        if (adapter.streamMessage) {
          await sendStream(adapter, request, controller.signal, {
            setChatId,
            setTaskId,
            setRunId,
            setMessages,
            setStreamDraft,
            onAssistantDone,
            nowLabel,
          });
        } else {
          const response = await adapter.sendMessage(request);
          const nextChatID = responseChatId(response);
          const nextTaskID = responseTaskId(response);
          const nextRunID = responseRunId(response);
          if (nextChatID) setChatId(nextChatID);
          if (nextTaskID) setTaskId(nextTaskID);
          if (nextRunID) setRunId(nextRunID);
          const assistantMessage = messageFromResponse(response, nextID("assistant"));
          assistantMessage.timestamp = nowLabel?.();
          setMessages((prev) => [...prev, assistantMessage]);
          onAssistantDone?.(assistantMessage, response);
        }
        void refreshConversations();
      } catch (err) {
        if (controller.signal.aborted) return;
        setError(errorMessage(err, "No se pudo enviar el mensaje"));
        setMessages((prev) => prev.filter((message) => message.id !== userMessage.id));
        setInput(text);
      } finally {
        setLoading(false);
        setStreamDraft(null);
      }
    },
    [adapter, baseRequest, chatId, input, loading, nowLabel, onAssistantDone, refreshConversations, taskId],
  );

  const confirmPending = useCallback(async () => {
    const ids = pendingConfirmations.map((item) => itemID(item)).filter(Boolean);
    if (ids.length === 0) return;
    await send({ message: "", confirmedActions: ids });
  }, [pendingConfirmations, send]);

  return {
    chatId,
    taskId,
    runId,
    messages,
    conversations,
    input,
    setInput,
    loading,
    loadingHistory,
    error,
    streamDraft,
    pendingConfirmations,
    send,
    confirmPending,
    loadConversation,
    refreshConversations,
    newChat,
  };
}

type StreamSetters = {
  setChatId: (value: string) => void;
  setTaskId: (value: string) => void;
  setRunId: (value: string) => void;
  setMessages: Dispatch<SetStateAction<ChatMessage[]>>;
  setStreamDraft: Dispatch<SetStateAction<ChatStreamDraft | null>>;
  onAssistantDone?: (message: ChatMessage, response: ChatResponse) => void;
  nowLabel?: () => string;
};

async function sendStream(adapter: ChatAdapter, request: ChatRequest, signal: AbortSignal, setters: StreamSetters) {
  let finalResponse: ChatResponse | null = null;
  setters.setStreamDraft({ text: "", activity: [] });
  await adapter.streamMessage?.(
    request,
    (event: ChatStreamEvent) => {
      const nextChatID = streamEventChatId(event);
      if (nextChatID) setters.setChatId(nextChatID);
      if (event.event === "text") {
        const chunk = streamEventText(event);
        if (chunk) {
          setters.setStreamDraft((draft) => (draft ? { ...draft, text: draft.text + chunk } : draft));
        }
        return;
      }
      if (event.event === "tool_call" || event.event === "tool_result") {
        const prefix = event.event === "tool_call" ? "Consultando" : "Listo";
        const label = summarizeToolCall(event.data);
        setters.setStreamDraft((draft) =>
          draft ? { ...draft, activity: [...draft.activity, `${prefix}: ${label}`] } : draft,
        );
        return;
      }
      if (event.event === "done") {
        finalResponse = streamEventToResponse(event);
      }
    },
    signal,
  );
  if (finalResponse) {
    const response = finalResponse;
    const nextChatID = responseChatId(response);
    const nextTaskID = responseTaskId(response);
    const nextRunID = responseRunId(response);
    if (nextChatID) setters.setChatId(nextChatID);
    if (nextTaskID) setters.setTaskId(nextTaskID);
    if (nextRunID) setters.setRunId(nextRunID);
    const assistantMessage = messageFromResponse(response, nextID("assistant"));
    assistantMessage.timestamp = setters.nowLabel?.();
    setters.setMessages((prev) => [...prev, assistantMessage]);
    setters.onAssistantDone?.(assistantMessage, response);
  }
}

function itemID(item: unknown): string {
  if (typeof item === "string") return item;
  if (item && typeof item === "object" && "id" in item && typeof item.id === "string") return item.id;
  return "";
}

function nextID(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
