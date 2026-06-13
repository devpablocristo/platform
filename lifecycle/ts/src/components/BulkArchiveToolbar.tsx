import { useState } from "react";
import type { BulkArchiveLabels, LifecyclePolicy } from "../types";

export type BulkArchiveToolbarProps = {
  selectedIds: string[];
  policy: LifecyclePolicy;
  labels: BulkArchiveLabels;
  onArchive: (reason?: string) => void | Promise<void>;
  onCancel: () => void;
  busy?: boolean;
};

/**
 * Toolbar that appears when a CrudPage table has rows selected. Shows the
 * selection count and an Archive button; collects a reason when the policy
 * requires it.
 *
 * The consumer is responsible for clearing the selection after a successful
 * archive (typically in the onArchive promise).
 */
export function BulkArchiveToolbar(props: BulkArchiveToolbarProps) {
  const [reason, setReason] = useState("");
  const count = props.selectedIds.length;
  if (count === 0) return null;

  const reasonRequired = props.policy.requireReason;
  const reasonInvalid = reasonRequired && reason.trim() === "";

  return (
    <div data-platform-lifecycle="BulkArchiveToolbar" role="toolbar">
      <span>{props.labels.selectionPrefix.replace("{n}", String(count))}</span>

      {(reasonRequired || props.labels.reasonLabel) && (
        <label>
          <span>{props.labels.reasonLabel ?? ""}</span>
          <input
            type="text"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={props.labels.reasonPlaceholder}
            aria-required={reasonRequired}
          />
        </label>
      )}

      <button type="button" onClick={props.onCancel} disabled={props.busy}>
        {props.labels.cancelButton}
      </button>
      <button
        type="button"
        onClick={() => void Promise.resolve(props.onArchive(reason.trim() || undefined))}
        disabled={props.busy || reasonInvalid}
      >
        {props.labels.archiveButton}
      </button>
    </div>
  );
}
