import type { ReactNode } from "react";

export type InsightListItem = {
  id: string;
  title: string;
  summary: string;
  badge?: string;
  impact?: string | null;
  metadata?: string[];
  ctaLabel?: string | null;
  action?: ReactNode;
};

export type InsightCardsListProps = {
  title: string;
  items: InsightListItem[];
  emptyMessage: string;
  collapsedCount?: number;
  expanded?: boolean;
  onToggleExpanded?: () => void;
  showAllLabel?: string;
  showLessLabel?: string;
};

export function InsightCardsList({
  title,
  items,
  emptyMessage,
  collapsedCount,
  expanded = false,
  onToggleExpanded,
  showAllLabel = "Mostrar todos",
  showLessLabel = "Mostrar menos",
}: InsightCardsListProps) {
  const visibleItems =
    collapsedCount && !expanded ? items.slice(0, collapsedCount) : items;

  return (
    <div className="border rounded-md p-4">
      <h3 className="font-semibold mb-2">{title}</h3>
      {items.length === 0 ? (
        <div className="text-sm text-slate-500">{emptyMessage}</div>
      ) : (
        <div className="grid grid-cols-1 gap-3">
          {visibleItems.map((item) => (
            <div key={item.id} className="rounded-md border p-3">
              <div className="flex items-center justify-between gap-3">
                <div className="font-semibold">{item.title}</div>
                {item.badge ? (
                  <div className="text-xs text-slate-500">{item.badge}</div>
                ) : null}
              </div>
              <div className="text-sm text-slate-600">{item.summary}</div>
              {item.ctaLabel ? (
                <div className="mt-2 text-sm text-slate-700">CTA: {item.ctaLabel}</div>
              ) : null}
              {item.impact ? (
                <div className="mt-2 text-xs text-slate-500">{item.impact}</div>
              ) : null}
              {item.metadata?.length ? (
                <div className="mt-2 flex items-center gap-3 flex-wrap text-xs text-slate-500">
                  {item.metadata.map((entry) => (
                    <span key={entry}>{entry}</span>
                  ))}
                </div>
              ) : null}
              {item.action ? <div className="mt-2">{item.action}</div> : null}
            </div>
          ))}
          {collapsedCount && items.length > collapsedCount && onToggleExpanded ? (
            <button
              type="button"
              className="text-sm text-blue-600 hover:underline"
              onClick={onToggleExpanded}
            >
              {expanded ? showLessLabel : showAllLabel}
            </button>
          ) : null}
        </div>
      )}
    </div>
  );
}
