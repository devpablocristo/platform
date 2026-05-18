import { useEffect, useState, type FormEvent } from 'react';
import { confirmAction } from '@devpablocristo/core-browser';
import { SchedulingDateInput } from './SchedulingDateInput';
import type {
  InternalEvent,
  InternalEventStatus,
  InternalEventVisibility,
  SchedulingCalendarCopy,
  SchedulingEntryType,
} from './types';
import { SchedulingEntryTypeSwitcher } from './SchedulingEntryTypeSwitcher';

// Modal de eventos internos (Etapa 3 / Pymes agenda Google-like).
//
// No comparte estado con SchedulingBookingModal a propósito: los dominios son
// distintos (eventos internos vs turnos cliente) y mezclar branches de UI por
// "modo" sería el mismo error de responsabilidades que ya identificamos en el
// backend. Si en el futuro hace falta un modal unificado, lo construimos como
// shell que delega al modal correcto según el tipo, no al revés.

export type InternalEventModalState =
  | { open: false }
  | {
      open: true;
      mode: 'create';
      branchId: string | null;
      resourceOptions: Array<{ id: string; name: string }>;
      initial: InternalEventDraft;
    }
  | {
      open: true;
      mode: 'edit';
      id: string;
      branchId: string | null;
      resourceOptions: Array<{ id: string; name: string }>;
      initial: InternalEventDraft;
    };

export type InternalEventDraft = {
  title: string;
  description: string;
  date: string; // YYYY-MM-DD wall date en zona del navegador
  startTime: string; // HH:MM
  endTime: string; // HH:MM
  resourceId: string; // '' = sin recurso (tiempo personal)
  status: InternalEventStatus;
  visibility: InternalEventVisibility;
};

type Props = {
  state: InternalEventModalState;
  copy: SchedulingCalendarCopy;
  locale?: string;
  saving?: boolean;
  onClose: () => void;
  onSave: (draft: InternalEventDraft) => Promise<void> | void;
  onDelete?: (id: string) => Promise<void> | void;
  // Switcher de tipo: solo se renderiza si viene la callback (modo create).
  // En edit mode no tiene sentido cambiar de tipo a algo creado.
  onSwitchType?: (type: SchedulingEntryType) => void;
  bookingEnabled?: boolean;
};

const STATUS_OPTIONS: InternalEventStatus[] = ['scheduled', 'done', 'cancelled'];
const VISIBILITY_OPTIONS: InternalEventVisibility[] = ['team', 'private'];

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

export function internalEventDraftFromEntity(event: InternalEvent): InternalEventDraft {
  const start = new Date(event.start_at);
  const end = new Date(event.end_at);
  return {
    title: event.title ?? '',
    description: event.description ?? '',
    date: toDateInputValue(start),
    startTime: toTimeInputValue(start),
    endTime: toTimeInputValue(end),
    resourceId: event.resource_id ?? '',
    status: event.status,
    visibility: event.visibility,
  };
}

export function emptyInternalEventDraft(date: string, defaults?: Partial<InternalEventDraft>): InternalEventDraft {
  return {
    title: '',
    description: '',
    date,
    startTime: '09:00',
    endTime: '10:00',
    resourceId: '',
    status: 'scheduled',
    visibility: 'team',
    ...defaults,
  };
}

export function SchedulingInternalEventModal({
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
  const [draft, setDraft] = useState<InternalEventDraft>(() =>
    state.open ? state.initial : emptyInternalEventDraft(toDateInputValue(new Date())),
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
    if (!draft.title.trim() || !draft.date || !draft.startTime || !draft.endTime) {
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
      title: copy.internalEventDeleteConfirm,
      description: '',
      confirmLabel: copy.internalEventDelete,
      cancelLabel: copy.close,
      tone: 'danger',
    });
    if (!confirmed) {
      return;
    }
    await onDelete(state.id);
  };

  const isEdit = state.mode === 'edit';
  const title = isEdit ? copy.internalEventModalEditTitle : copy.internalEventModalCreateTitle;
  const submitLabel = saving ? copy.saving : copy.internalEventSave;
  const titleInvalid = !draft.title.trim();
  const rangeInvalid = draft.startTime >= draft.endTime;

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
            <p className="app-modal__eyebrow">{copy.internalEventCreateAction}</p>
            <h3 className="app-modal__title">{title}</h3>
          </div>
          <button type="button" className="app-modal__close" aria-label={copy.close} onClick={onClose}>
            ×
          </button>
        </div>

        <form className="app-modal__body modules-scheduling__modal-form" onSubmit={handleSubmit}>
          {!isEdit && onSwitchType ? (
            <SchedulingEntryTypeSwitcher
              active="internal_event"
              copy={copy}
              bookingEnabled={bookingEnabled}
              onSwitch={onSwitchType}
            />
          ) : null}
          <div className="form-group">
            <label htmlFor="internal-event-title">{copy.internalEventTitleLabel}</label>
            <input
              id="internal-event-title"
              value={draft.title}
              placeholder={copy.internalEventTitlePlaceholder}
              onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
              autoFocus
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="internal-event-description">{copy.internalEventDescriptionLabel}</label>
            <textarea
              id="internal-event-description"
              value={draft.description}
              rows={3}
              onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))}
            />
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="internal-event-date">{copy.focusDateLabel}</label>
              <SchedulingDateInput
                id="internal-event-date"
                locale={locale}
                value={draft.date}
                onValueChange={(iso) => setDraft((current) => ({ ...current, date: iso }))}
                required
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="internal-event-resource">{copy.internalEventResourceLabel}</label>
              <select
                id="internal-event-resource"
                value={draft.resourceId}
                onChange={(event) => setDraft((current) => ({ ...current, resourceId: event.target.value }))}
              >
                <option value="">{copy.internalEventResourceUnassigned}</option>
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
              <label htmlFor="internal-event-start">{copy.internalEventStartLabel}</label>
              <input
                id="internal-event-start"
                type="time"
                value={draft.startTime}
                onChange={(event) => setDraft((current) => ({ ...current, startTime: event.target.value }))}
                required
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="internal-event-end">{copy.internalEventEndLabel}</label>
              <input
                id="internal-event-end"
                type="time"
                value={draft.endTime}
                onChange={(event) => setDraft((current) => ({ ...current, endTime: event.target.value }))}
                required
              />
            </div>
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="internal-event-status">{copy.internalEventStatusLabel}</label>
              <select
                id="internal-event-status"
                value={draft.status}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, status: event.target.value as InternalEventStatus }))
                }
              >
                {STATUS_OPTIONS.map((status) => (
                  <option key={status} value={status}>
                    {copy.internalEventStatusOptions[status]}
                  </option>
                ))}
              </select>
            </div>
            <div className="form-group grow">
              <label htmlFor="internal-event-visibility">{copy.internalEventVisibilityLabel}</label>
              <select
                id="internal-event-visibility"
                value={draft.visibility}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, visibility: event.target.value as InternalEventVisibility }))
                }
              >
                {VISIBILITY_OPTIONS.map((visibility) => (
                  <option key={visibility} value={visibility}>
                    {copy.internalEventVisibilityOptions[visibility]}
                  </option>
                ))}
              </select>
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
                {copy.internalEventDelete}
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
              disabled={saving || titleInvalid || rangeInvalid}
            >
              {submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
