import type { ReactNode } from "react";
import type { ChatConversationSummary } from "./types";

export function ConversationList({
  conversations,
  activeId,
  loading = false,
  title = "Conversations",
  emptyMessage = "No conversations",
  onSelect,
  onNew,
  newLabel = "New",
}: {
  conversations: ChatConversationSummary[];
  activeId?: string | null;
  loading?: boolean;
  title?: ReactNode;
  emptyMessage?: ReactNode;
  onSelect: (id: string) => void;
  onNew?: () => void;
  newLabel?: ReactNode;
}) {
  return (
    <aside className="m-chat-conversations" aria-label={String(title)}>
      <div className="m-chat-conversations__header">
        <strong>{title}</strong>
        {onNew ? (
          <button type="button" onClick={onNew}>
            {newLabel}
          </button>
        ) : null}
      </div>
      {loading ? <p>Loading...</p> : null}
      {conversations.length === 0 ? <p className="m-chat-muted">{emptyMessage}</p> : null}
      <ul>
        {conversations.map((conversation) => (
          <li key={conversation.id}>
            <button
              type="button"
              className={conversation.id === activeId ? "is-active" : ""}
              onClick={() => onSelect(conversation.id)}
            >
              <span>{conversation.title || conversation.id}</span>
              <small>{conversation.updatedAt ?? conversation.updated_at ?? conversation.createdAt ?? conversation.created_at}</small>
            </button>
          </li>
        ))}
      </ul>
    </aside>
  );
}
