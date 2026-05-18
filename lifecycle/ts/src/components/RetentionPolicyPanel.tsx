import { useState } from "react";
import type { ArchivePolicy, RetentionLabels } from "../types";

export type RetentionPolicyPanelProps = {
  policy: ArchivePolicy;
  onChange: (next: ArchivePolicy) => void;
  labels: RetentionLabels;
  /** When true, disables interaction (e.g. read-only view for non-admins). */
  readOnly?: boolean;
};

/**
 * Editable view of an ArchivePolicy for admin surfaces. As with
 * ArchiveConfirmDialog, this is unstyled and label-driven so the consumer can
 * wrap or restyle freely.
 */
export function RetentionPolicyPanel(props: RetentionPolicyPanelProps) {
  const [draft, setDraft] = useState<ArchivePolicy>(props.policy);

  const update = (patch: Partial<ArchivePolicy>) => {
    const next = { ...draft, ...patch };
    setDraft(next);
    props.onChange(next);
  };

  return (
    <section data-platform-lifecycle="RetentionPolicyPanel">
      <h3>{props.labels.title}</h3>
      <label>
        <span>{props.labels.daysLabel}</span>
        <input
          type="number"
          min={0}
          value={draft.retentionDays}
          onChange={(e) =>
            update({ retentionDays: Math.max(0, Number(e.target.value) || 0) })
          }
          disabled={props.readOnly}
        />
      </label>
      <label>
        <input
          type="checkbox"
          checked={draft.allowHardDelete}
          onChange={(e) => update({ allowHardDelete: e.target.checked })}
          disabled={props.readOnly}
        />
        <span>{props.labels.allowHardDeleteLabel}</span>
      </label>
      <label>
        <input
          type="checkbox"
          checked={draft.requireReason}
          onChange={(e) => update({ requireReason: e.target.checked })}
          disabled={props.readOnly}
        />
        <span>{props.labels.requireReasonLabel}</span>
      </label>
    </section>
  );
}
