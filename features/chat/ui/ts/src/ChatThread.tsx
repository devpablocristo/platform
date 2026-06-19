import type { Ref, RefObject } from "react";
import type { ChatMessage, ChatStreamDraft } from "./types";
import { ChatMarkdown } from "./ChatMarkdown";
import { ToolCallsPanel } from "./ToolCallsPanel";

export function ChatThread({
  messages,
  streamDraft,
  loading = false,
  loadingMessage = "Loading...",
  emptyMessage = "No messages yet",
  endRef,
}: {
  messages: ChatMessage[];
  streamDraft?: ChatStreamDraft | null;
  loading?: boolean;
  loadingMessage?: string;
  emptyMessage?: string;
  endRef?: RefObject<HTMLDivElement | null>;
}) {
  return (
    <div className="m-chat-thread" role="log" aria-live="polite" aria-busy={loading}>
      {loading ? <div className="m-chat-thread__loading">{loadingMessage}</div> : null}
      {messages.length === 0 && !streamDraft ? <div className="m-chat-thread__empty">{emptyMessage}</div> : null}
      {messages.map((message, index) => {
        const fromUser = message.role === "user";
        return (
          <article
            key={message.id ?? `${message.role}-${index}`}
            className={`m-chat-message ${fromUser ? "m-chat-message--user" : "m-chat-message--assistant"}`}
          >
            <ChatMarkdown content={message.content} />
            <ToolCallsPanel toolCalls={message.toolCalls} />
            {message.timestamp ? <time>{message.timestamp}</time> : null}
          </article>
        );
      })}
      {streamDraft ? (
        <article className="m-chat-message m-chat-message--assistant m-chat-message--stream">
          {streamDraft.activity.length > 0 ? (
            <ul className="m-chat-activity">
              {streamDraft.activity.map((item, index) => (
                <li key={`${item}-${index}`}>{item}</li>
              ))}
            </ul>
          ) : null}
          <ChatMarkdown content={streamDraft.text} />
        </article>
      ) : null}
      <div ref={endRef as Ref<HTMLDivElement> | undefined} />
    </div>
  );
}
