import type { ReactNode } from "react";

export type NotificationFeedTone = "default" | "attention" | "critical" | "success";

export type NotificationFeedItem = {
  id: string;
  eyebrow?: ReactNode;
  title: ReactNode;
  body?: ReactNode;
  meta?: ReactNode;
  timestamp?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;
  extra?: ReactNode;
  unread?: boolean;
  tone?: NotificationFeedTone;
};

export type NotificationFeedProps = {
  items: NotificationFeedItem[];
  loading?: boolean;
  loadingMessage?: ReactNode;
  emptyMessage: ReactNode;
  error?: ReactNode;
  summary?: ReactNode;
  className?: string;
};

function toneClassName(tone: NotificationFeedTone | undefined): string {
  switch (tone) {
    case "attention":
      return "m-notification-feed__card--attention";
    case "critical":
      return "m-notification-feed__card--critical";
    case "success":
      return "m-notification-feed__card--success";
    default:
      return "m-notification-feed__card--default";
  }
}

export function NotificationFeed({
  items,
  loading = false,
  loadingMessage = "Cargando…",
  emptyMessage,
  error,
  summary,
  className = "",
}: NotificationFeedProps) {
  return (
    <section className={`m-notification-feed ${className}`.trim()}>
      {error ? <div className="m-notification-feed__error">{error}</div> : null}
      {summary ? <div className="m-notification-feed__summary">{summary}</div> : null}
      {loading ? (
        <div className="m-notification-feed__empty">{loadingMessage}</div>
      ) : items.length === 0 ? (
        <div className="m-notification-feed__empty">{emptyMessage}</div>
      ) : (
        <ul className="m-notification-feed__list">
          {items.map((item) => (
            <li
              key={item.id}
              className={`m-notification-feed__card ${toneClassName(item.tone)}${
                item.unread ? " m-notification-feed__card--unread" : ""
              }`}
            >
              {item.eyebrow ? <div className="m-notification-feed__eyebrow">{item.eyebrow}</div> : null}
              <div className="m-notification-feed__header">
                <div className="m-notification-feed__titleWrap">
                  <div className="m-notification-feed__title">{item.title}</div>
                  {item.meta ? <div className="m-notification-feed__meta">{item.meta}</div> : null}
                </div>
                {item.badge ? <div className="m-notification-feed__badge">{item.badge}</div> : null}
              </div>
              {item.body ? <div className="m-notification-feed__body">{item.body}</div> : null}
              {item.timestamp ? <div className="m-notification-feed__timestamp">{item.timestamp}</div> : null}
              {item.extra ? <div className="m-notification-feed__extra">{item.extra}</div> : null}
              {item.actions ? <div className="m-notification-feed__actions">{item.actions}</div> : null}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
