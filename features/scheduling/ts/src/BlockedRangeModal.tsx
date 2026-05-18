import { useEffect, useState, type FormEvent } from 'react';
import { confirmAction } from '@devpablocristo/core-browser';
import { SchedulingDateInput } from './SchedulingDateInput';
import type { BlockedRange, BlockedRangeKind, SchedulingCalendarCopy, SchedulingEntryType } from './types';
import { SchedulingEntryTypeSwitcher } from './SchedulingEntryTypeSwitcher';

export type BlockedRangeModalState =
  | { open: false }
  | {
      open: true;
      mode: 'create';
      branchId: string;
      resourceId: string | null;
      resourceOptions: Array<{ id: string; name: string }>;
      initial: BlockedRangeDraft;
    }
  | {
      open: true;
      mode: 'edit';
      id: string;
      branchId: string;
      resourceOptions: Array<{ id: string; name: string }>;
      initial: BlockedRangeDraft;
    };

export type BlockedRangeDraft = {
  kind: BlockedRangeKind;
  reason: string;
  date: string; // YYYY-MM-DD
  startTime: string; // HH:MM
  endTime: string; // HH:MM
  resourceId: string; // '' === all resources
};

type Props = {
  state: BlockedRangeModalState;
  copy: SchedulingCalendarCopy;
  locale?: string;
  saving?: boolean;
  onClose: () => void;
  onSave: (draft: BlockedRangeDraft) => Promise<void> | void;
  onDelete?: (id: string) => Promise<void> | void;
  onSwitchType?: (type: SchedulingEntryType) => void;
  bookingEnabled?: boolean;
};

const BLOCKED_KINDS: BlockedRangeKind[] = ['manual', 'holiday', 'maintenance', 'leave'];

export function blockedRangeDraftFromRange(range: BlockedRange): BlockedRangeDraft {
  const start = new Date(range.start_at);
  const end = new Date(range.end_at);
  return {
    kind: range.kind,
    reason: range.reason ?? '',
    date: toDateInputValue(start),
    startTime: toTimeInputValue(start),
    endTime: toTimeInputValue(end),
    resourceId: range.resource_id ?? '',
  };
}

export function emptyBlockedRangeDraft(date: string): BlockedRangeDraft {
  return {
    kind: 'manual',
    reason: '',
    date,
    startTime: '09:00',
    endTime: '10:00',
    resourceId: '',
  };
}

function toDateInputValue(value: Date): string {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, '0');
  const day = String(value.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function toTimeInputValue(value: Date): string {
  const hours = String(value.getHours()).padStart(2, '0');
  const minutes = String(value.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}

export function BlockedRangeModal({
  state,
  copy,
  locale,
  saving = false,
  onClose,
  onSave,
  onDelete,
  onSwitchType,
  bookingEnabled = true,
}: Props) {
  const [draft, setDraft] = useState<BlockedRangeDraft>(() =>
    state.open ? state.initial : emptyBlockedRangeDraft(toDateInputValue(new Date())),
  );

  useEffect(() => {
    if (state.open) {
      setDraft(state.initial);
    }
  }, [state]);

  if (!state.open) {
    return null;
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!draft.date || !draft.startTime || !draft.endTime) {
      return;
    }
    if (draft.startTime >= draft.endTime) {
      return;
    }
    await onSave(draft);
  };

  const handleDeleteClick = async () => {
    if (state.mode !== 'edit' || !onDelete) {
      return;
    }
    const confirmed = await confirmAction({
      title: copy.blockedRangeDeleteTitle,
      description: copy.blockedRangeDeleteDescription,
      confirmLabel: copy.blockedRangeDelete,
      cancelLabel: copy.close,
      tone: 'danger',
    });
    if (!confirmed) {
      return;
    }
    await onDelete(state.id);
  };

  const isEdit = state.mode === 'edit';
  const title = isEdit ? copy.blockedRangeEditTitle : copy.blockedRangeCreateTitle;
  const submitLabel = saving ? copy.saving : isEdit ? copy.blockedRangeUpdate : copy.blockedRangeCreate;

  return (
    <div className="modules-scheduling__backdrop app-modal-backdrop" role="presentation" onClick={onClose}>
      <div
        className="modules-scheduling__modal app-modal"
        role="dialog"
        aria-modal="true"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="app-modal__header">
          <div className="app-modal__title-block">
            <p className="app-modal__eyebrow">{copy.blockedRangeEyebrow}</p>
            <h3 className="app-modal__title">{title}</h3>
          </div>
          <button type="button" className="app-modal__close" aria-label={copy.close} onClick={onClose}>
            ×
          </button>
        </div>

        <form className="app-modal__body modules-scheduling__modal-form" onSubmit={handleSubmit}>
          {!isEdit && onSwitchType ? (
            <SchedulingEntryTypeSwitcher
              active="blocked_range"
              copy={copy}
              bookingEnabled={bookingEnabled}
              onSwitch={onSwitchType}
            />
          ) : null}
          <div className="form-group">
            <label htmlFor="blocked-range-kind">{copy.blockedRangeKindLabel}</label>
            <select
              id="blocked-range-kind"
              value={draft.kind}
              onChange={(event) => setDraft((current) => ({ ...current, kind: event.target.value as BlockedRangeKind }))}
            >
              {BLOCKED_KINDS.map((kind) => (
                <option key={kind} value={kind}>
                  {copy.blockedRangeKindOptions[kind]}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="blocked-range-reason">{copy.blockedRangeReasonLabel}</label>
            <input
              id="blocked-range-reason"
              value={draft.reason}
              placeholder={copy.blockedRangeReasonPlaceholder}
              onChange={(event) => setDraft((current) => ({ ...current, reason: event.target.value }))}
              autoFocus
            />
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="blocked-range-date">{copy.focusDateLabel}</label>
              <SchedulingDateInput
                id="blocked-range-date"
                locale={locale}
                value={draft.date}
                onValueChange={(iso) => setDraft((current) => ({ ...current, date: iso }))}
                required
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="blocked-range-resource">{copy.resourceNameLabel}</label>
              <select
                id="blocked-range-resource"
                value={draft.resourceId}
                onChange={(event) => setDraft((current) => ({ ...current, resourceId: event.target.value }))}
              >
                <option value="">{copy.anyResource}</option>
                {state.resourceOptions.map((resource) => (
                  <option key={resource.id} value={resource.id}>
                    {resource.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="blocked-range-start">{copy.slotStartLabel}</label>
              <input
                id="blocked-range-start"
                type="time"
                value={draft.startTime}
                onChange={(event) => setDraft((current) => ({ ...current, startTime: event.target.value }))}
                required
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="blocked-range-end">{copy.slotEndLabel}</label>
              <input
                id="blocked-range-end"
                type="time"
                value={draft.endTime}
                onChange={(event) => setDraft((current) => ({ ...current, endTime: event.target.value }))}
                required
              />
            </div>
          </div>

          <div className="app-modal__footer">
            {isEdit && onDelete ? (
              <button
                type="button"
                className="btn-danger btn-sm app-modal__action"
                onClick={() => void handleDeleteClick()}
                disabled={saving}
              >
                {copy.blockedRangeDelete}
              </button>
            ) : null}
            <div className="app-modal__footer-spacer" aria-hidden />
            <button
              type="button"
              className="btn-secondary btn-sm app-modal__action"
              disabled={saving}
              onClick={onClose}
            >
              {copy.close}
            </button>
            <button
              type="submit"
              className="btn-primary btn-sm app-modal__action"
              disabled={saving || draft.startTime >= draft.endTime}
            >
              {submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
