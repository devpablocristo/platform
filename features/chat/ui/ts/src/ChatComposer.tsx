import type { ReactNode } from "react";
import { PendingConfirmationsPanel } from "./PendingConfirmationsPanel";

export function ChatComposer({
  input,
  onInputChange,
  onSend,
  busy = false,
  placeholder = "Write a message",
  sendLabel = "Send",
  sendingLabel = "Sending...",
  pendingConfirmations = [],
  confirmPendingLabel = "Confirm",
  onConfirmPending,
}: {
  input: string;
  onInputChange: (value: string) => void;
  onSend: () => void;
  busy?: boolean;
  placeholder?: string;
  sendLabel?: ReactNode;
  sendingLabel?: ReactNode;
  pendingConfirmations?: unknown[];
  confirmPendingLabel?: ReactNode;
  onConfirmPending?: () => void;
}) {
  return (
    <div className="m-chat-composer-wrap">
      <PendingConfirmationsPanel
        items={pendingConfirmations}
        busy={busy}
        confirmLabel={confirmPendingLabel}
        onConfirm={onConfirmPending}
      />
      <div className="m-chat-composer">
        <textarea
          aria-label={placeholder}
          placeholder={placeholder}
          value={input}
          disabled={busy}
          rows={2}
          onChange={(event) => onInputChange(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              onSend();
            }
          }}
        />
        <button type="button" disabled={busy || input.trim() === ""} onClick={onSend}>
          {busy ? sendingLabel : sendLabel}
        </button>
      </div>
    </div>
  );
}
