import type { ReactNode } from "react";

import type { NotificationFeedItem, NotificationFeedTone } from "./NotificationFeed";

export type NotificationChatHandoff = {
  notificationId: string;
  title?: string;
  body?: string;
  scope?: string;
  routedAgent?: string;
  contentLanguage?: string;
  suggestedUserMessage?: string;
  sourceNotificationKind?: string;
  entityType?: string;
  entityId?: string;
  chatContext?: Record<string, unknown>;
};

export function buildHandoffUserMessage(handoff: NotificationChatHandoff): string {
  const suggested = typeof handoff.suggestedUserMessage === "string" ? handoff.suggestedUserMessage.trim() : "";
  if (suggested) {
    return suggested;
  }
  const title = typeof handoff.title === "string" ? handoff.title.trim() : "";
  const body = typeof handoff.body === "string" ? handoff.body.trim() : "";
  if (title && body) {
    return `Necesito mas informacion sobre: ${title}\n\n${body}`;
  }
  if (title) {
    return `Necesito mas informacion sobre: ${title}`;
  }
  return "Necesito mas contexto sobre esta notificacion.";
}

export type AINotificationRecord = {
  id: string;
  title: ReactNode;
  body?: ReactNode;
  timestamp?: ReactNode;
  meta?: ReactNode;
  badge?: ReactNode;
  unread?: boolean;
  severity?: number | null;
  routeHint?: string | null;
  scope?: string | null;
  source?: string | null;
  extra?: ReactNode;
  actions?: ReactNode;
};

export type ApprovalNotificationRecord = {
  id: string;
  title: ReactNode;
  body?: ReactNode;
  timestamp?: ReactNode;
  meta?: ReactNode;
  unread?: boolean;
  riskLevel?: "low" | "medium" | "high" | string | null;
  status?: string | null;
  requestLabel?: ReactNode;
  actions?: ReactNode;
  extra?: ReactNode;
};

function toneForSeverity(severity: number | null | undefined): NotificationFeedTone {
  if ((severity ?? 0) >= 80) return "critical";
  if ((severity ?? 0) >= 50) return "attention";
  return "default";
}

function toneForRiskLevel(riskLevel: string | null | undefined): NotificationFeedTone {
  switch ((riskLevel || "").trim().toLowerCase()) {
    case "high":
      return "critical";
    case "medium":
      return "attention";
    case "low":
      return "default";
    default:
      return "default";
  }
}

function defaultAIBadge(severity: number | null | undefined): string | null {
  if (severity == null) return null;
  if (severity >= 80) return "Alta";
  if (severity >= 50) return "Media";
  return "Baja";
}

function defaultApprovalBadge(status: string | null | undefined): string | null {
  const normalized = (status || "").trim();
  return normalized || null;
}

export function toAINotificationFeedItem(record: AINotificationRecord): NotificationFeedItem {
  return {
    id: record.id,
    title: record.title,
    body: record.body,
    timestamp: record.timestamp,
    meta: record.meta,
    badge: record.badge ?? defaultAIBadge(record.severity),
    unread: Boolean(record.unread),
    tone: toneForSeverity(record.severity),
    eyebrow: record.scope ?? record.source ?? undefined,
    extra: record.extra,
    actions: record.actions,
  };
}

export function toApprovalNotificationFeedItem(record: ApprovalNotificationRecord): NotificationFeedItem {
  return {
    id: record.id,
    title: record.title,
    body: record.body,
    timestamp: record.timestamp,
    meta: record.meta,
    badge: record.requestLabel ?? defaultApprovalBadge(record.status),
    unread: Boolean(record.unread),
    tone: toneForRiskLevel(record.riskLevel),
    eyebrow: record.riskLevel ?? undefined,
    extra: record.extra,
    actions: record.actions,
  };
}
