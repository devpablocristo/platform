import type { ReactNode } from "react";

export function PendingConfirmationsPanel({
  items = [],
  busy = false,
  title = "Pending",
  confirmLabel = "Confirm",
  onConfirm,
}: {
  items?: unknown[];
  busy?: boolean;
  title?: ReactNode;
  confirmLabel?: ReactNode;
  onConfirm?: () => void;
}) {
  if (items.length === 0) return null;
  return (
    <section className="m-chat-pending" aria-label={String(title)}>
      <span>
        {title}: {items.map(labelFor).join(", ")}
      </span>
      {onConfirm ? (
        <button type="button" disabled={busy} onClick={onConfirm}>
          {confirmLabel}
        </button>
      ) : null}
    </section>
  );
}

function labelFor(value: unknown): string {
  if (typeof value === "string") return value;
  if (value && typeof value === "object" && "id" in value && typeof value.id === "string") return value.id;
  return String(value ?? "");
}
