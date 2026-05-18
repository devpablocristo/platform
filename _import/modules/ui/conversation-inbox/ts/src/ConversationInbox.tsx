import type { ReactNode } from "react";
import {
  NotificationFeed,
  type NotificationFeedItem,
  type NotificationFeedTone,
} from "@devpablocristo/modules-ui-notification-feed";

export type ConversationInboxTone = NotificationFeedTone;

export type ConversationInboxItem = {
  id: string;
  contactName: ReactNode;
  contactDetail?: ReactNode;
  preview?: ReactNode;
  assignee?: ReactNode;
  status?: ReactNode;
  timestamp?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;
  unread?: boolean;
  tone?: ConversationInboxTone;
};

export type ConversationInboxProps = {
  items: ConversationInboxItem[];
  loading?: boolean;
  loadingMessage?: ReactNode;
  emptyMessage: ReactNode;
  error?: ReactNode;
  summary?: ReactNode;
  className?: string;
};

function toNotificationItem(item: ConversationInboxItem): NotificationFeedItem {
  return {
    id: item.id,
    title: item.contactName,
    meta: item.contactDetail,
    body: item.preview,
    extra:
      item.assignee || item.status ? (
        <div className="m-conversation-inbox__metaRow">
          {item.assignee ? <span className="m-conversation-inbox__pill">{item.assignee}</span> : null}
          {item.status ? <span className="m-conversation-inbox__pill">{item.status}</span> : null}
        </div>
      ) : undefined,
    timestamp: item.timestamp,
    badge: item.badge,
    actions: item.actions,
    unread: item.unread,
    tone: item.tone,
  };
}

export function ConversationInbox({
  items,
  loading = false,
  loadingMessage = "Cargando conversaciones…",
  emptyMessage,
  error,
  summary,
  className = "",
}: ConversationInboxProps) {
  return (
    <NotificationFeed
      items={items.map(toNotificationItem)}
      loading={loading}
      loadingMessage={loadingMessage}
      emptyMessage={emptyMessage}
      error={error}
      summary={summary}
      className={`m-conversation-inbox ${className}`.trim()}
    />
  );
}
