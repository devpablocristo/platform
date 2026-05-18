import { useState } from "react";
import type { ArchiveLabels, ArchivePolicy } from "../types";

export type ArchiveConfirmDialogProps = {
  open: boolean;
  policy: ArchivePolicy;
  labels: ArchiveLabels;
  onConfirm: (reason?: string) => void | Promise<void>;
  onCancel: () => void;
  /** Disables the confirm button while async work is in flight. */
  busy?: boolean;
};

/**
 * Minimal, unstyled confirm dialog for archive operations.
 *
 * Visual decisions (theme, layout, animation) belong to the consumer.
 * This component focuses on the *mechanics*: rendering a modal, capturing an
 * optional reason, and enforcing policy.requireReason at submit time.
 *
 * Consumers that need a polished look can wrap this dialog in their design
 * system (e.g. a Pymes Modal component) or copy this minimal markup and
 * style it. The library never imports a CSS file.
 */
export function ArchiveConfirmDialog(props: ArchiveConfirmDialogProps) {
  const [reason, setReason] = useState("");
  const [showRequiredHint, setShowRequiredHint] = useState(false);

  if (!props.open) return null;

  const reasonRequired = props.policy.requireReason;
  const reasonInvalid = reasonRequired && reason.trim() === "";

  const handleConfirm = () => {
    if (reasonInvalid) {
      setShowRequiredHint(true);
      return;
    }
    void Promise.resolve(props.onConfirm(reason.trim() || undefined));
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="archive-confirm-title"
      data-platform-lifecycle="ArchiveConfirmDialog"
    >
      <h2 id="archive-confirm-title">{props.labels.title}</h2>
      {props.labels.description && <p>{props.labels.description}</p>}

      {(reasonRequired || props.labels.reasonLabel) && (
        <label>
          <span>{props.labels.reasonLabel ?? ""}</span>
          <textarea
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              if (showRequiredHint) setShowRequiredHint(false);
            }}
            placeholder={props.labels.reasonPlaceholder}
            aria-required={reasonRequired}
            aria-invalid={showRequiredHint && reasonInvalid}
          />
          {showRequiredHint && reasonInvalid && props.labels.reasonRequiredHint && (
            <span role="alert">{props.labels.reasonRequiredHint}</span>
          )}
        </label>
      )}

      <div>
        <button type="button" onClick={props.onCancel} disabled={props.busy}>
          {props.labels.cancelButton}
        </button>
        <button
          type="button"
          onClick={handleConfirm}
          disabled={props.busy || reasonInvalid}
        >
          {props.labels.confirmButton}
        </button>
      </div>
    </div>
  );
}
