import { useState } from "react";
import type { LifecyclePolicy, RetentionLabels } from "../types";

export type RetentionPolicyPanelProps = {
  policy: LifecyclePolicy;
  onChange: (next: LifecyclePolicy) => void;
  labels: RetentionLabels;
  /** When true, disables interaction (e.g. read-only view for non-admins). */
  readOnly?: boolean;
};

/**
 * Editable view of a LifecyclePolicy for admin surfaces. As with
 * ArchiveConfirmDialog, this is unstyled and label-driven so the consumer can
 * wrap or restyle freely.
 */
export function RetentionPolicyPanel(props: RetentionPolicyPanelProps) {
  const [draft, setDraft] = useState<LifecyclePolicy>(props.policy);

  const update = (patch: Partial<LifecyclePolicy>) => {
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
          checked={draft.allowPurge}
          onChange={(e) => update({ allowPurge: e.target.checked })}
          disabled={props.readOnly}
        />
        <span>{props.labels.allowPurgeLabel}</span>
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
