import { useEffect, useState, type FormEvent } from 'react';
import { confirmAction } from '@devpablocristo/core-browser';
import { formatSchedulingDateTime, formatSchedulingWeekdayNarrow } from './locale';
import { SchedulingDateInput } from './SchedulingDateInput';
import type { Booking, SchedulingCalendarCopy, SchedulingEntryType, Service, TimeSlot } from './types';
import { SchedulingEntryTypeSwitcher } from './SchedulingEntryTypeSwitcher';

export type SchedulingBookingRecurrenceDraft = {
  mode: 'none' | 'daily' | 'weekly' | 'monthly' | 'custom';
  frequency: 'daily' | 'weekly' | 'monthly';
  interval: string;
  count: string;
  byWeekday: number[];
};

export type SchedulingBookingDraft = {
  title: string;
  customerName: string;
  customerPhone: string;
  customerEmail: string;
  notes: string;
  recurrence: SchedulingBookingRecurrenceDraft;
};

export type SchedulingBookingCreateEditor = {
  date: string;
  startTime: string;
  endTime: string;
  resourceId: string;
};

export type SchedulingBookingCreateResourceOption = {
  id: string;
  name: string;
  timezone?: string;
};

export type SchedulingBookingModalState =
  | {
      open: false;
    }
  | {
      open: true;
      mode: 'create';
      slot: TimeSlot | null;
      slotOptions: TimeSlot[];
      resourceOptions: SchedulingBookingCreateResourceOption[];
      editor: SchedulingBookingCreateEditor;
      validationMessage?: string | null;
      service?: Service;
      resourceName?: string;
      draft: SchedulingBookingDraft;
    }
  | {
      open: true;
      mode: 'details';
      booking: Booking;
      service?: Service;
      resourceName?: string;
    };

export type SchedulingBookingAction =
  | 'confirm'
  | 'cancel'
  | 'check_in'
  | 'start_service'
  | 'complete'
  | 'no_show';

type Props = {
  state: SchedulingBookingModalState;
  copy: SchedulingCalendarCopy;
  locale?: string;
  saving?: boolean;
  slotLoading?: boolean;
  onClose: () => void;
  onEditorChange?: (editor: Partial<SchedulingBookingCreateEditor>) => void;
  onCreate: (draft: SchedulingBookingDraft) => Promise<void> | void;
  onAction: (action: SchedulingBookingAction, booking: Booking) => Promise<void> | void;
  onSwitchType?: (type: SchedulingEntryType) => void;
};

function actionButtons(status: Booking['status']): SchedulingBookingAction[] {
  switch (status) {
    case 'hold':
    case 'pending_confirmation':
      return ['confirm', 'cancel'];
    case 'confirmed':
      return ['check_in', 'cancel', 'no_show'];
    case 'checked_in':
      return ['start_service', 'cancel'];
    case 'in_service':
      return ['complete'];
    default:
      return [];
  }
}

function actionLabel(copy: SchedulingCalendarCopy, action: SchedulingBookingAction): string {
  switch (action) {
    case 'confirm':
      return copy.confirmBooking;
    case 'cancel':
      return copy.cancelBooking;
    case 'check_in':
      return copy.checkInBooking;
    case 'start_service':
      return copy.startService;
    case 'complete':
      return copy.completeBooking;
    case 'no_show':
      return copy.noShowBooking;
  }
}

function defaultRecurrenceDraft(): SchedulingBookingRecurrenceDraft {
  return {
    mode: 'none',
    frequency: 'weekly',
    interval: '1',
    count: '8',
    byWeekday: [],
  };
}

export function SchedulingBookingModal({
  state,
  copy,
  locale,
  saving = false,
  slotLoading = false,
  onClose,
  onEditorChange,
  onCreate,
  onAction,
  onSwitchType,
}: Props) {
  const [draft, setDraft] = useState<SchedulingBookingDraft>({
    title: '',
    customerName: '',
    customerPhone: '',
    customerEmail: '',
    notes: '',
    recurrence: defaultRecurrenceDraft(),
  });

  const closeWithGuard = async () => {
    if (saving) {
      return;
    }
    if (state.open && state.mode === 'create') {
      const dirty =
        draft.title.trim() !== state.draft.title.trim() ||
        draft.customerName.trim() !== state.draft.customerName.trim() ||
        draft.customerPhone.trim() !== state.draft.customerPhone.trim() ||
        draft.customerEmail.trim() !== state.draft.customerEmail.trim() ||
        draft.notes.trim() !== state.draft.notes.trim() ||
        JSON.stringify(draft.recurrence) !== JSON.stringify(state.draft.recurrence);
      if (dirty) {
        const confirmed = await confirmAction({
          title: copy.closeDirtyTitle,
          description: copy.closeDirtyDescription,
          confirmLabel: copy.discard,
          cancelLabel: copy.keepEditing,
          tone: 'danger',
        });
        if (!confirmed) {
          return;
        }
      }
    }
    onClose();
  };

  useEffect(() => {
    if (state.open && state.mode === 'create') {
      setDraft(state.draft);
    }
  }, [state]);

  useEffect(() => {
    if (!state.open) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') {
        return;
      }
      event.preventDefault();
      void closeWithGuard();
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [state.open, closeWithGuard]);

  if (!state.open) {
    return null;
  }

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (state.open && state.mode === 'create' && !state.slot) {
      return;
    }
    await onCreate(draft);
  };

  const renderCreateSummary = () => {
    if (!(state.open && state.mode === 'create')) {
      return null;
    }

    const selectedSlot = state.slot;
    const activeFrequency = draft.recurrence.mode === 'custom' ? draft.recurrence.frequency : draft.recurrence.mode;

    return (
      <div className="modules-scheduling__creator-grid">
        <div className="modules-scheduling__creator-main">
          <div className="form-group">
            <label htmlFor="scheduling-booking-title">{copy.titleLabel}</label>
            <input
              id="scheduling-booking-title"
              value={draft.title}
              onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
              autoFocus
            />
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="scheduling-slot-date">{copy.focusDateLabel}</label>
              <SchedulingDateInput
                id="scheduling-slot-date"
                locale={locale}
                value={state.editor.date}
                onValueChange={(iso) => onEditorChange?.({ date: iso })}
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="scheduling-slot-resource">{copy.resourceNameLabel}</label>
              <select
                id="scheduling-slot-resource"
                value={state.editor.resourceId}
                onChange={(event) => onEditorChange?.({ resourceId: event.target.value })}
              >
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
              <label htmlFor="scheduling-slot-start">{copy.slotStartLabel}</label>
              <input
                id="scheduling-slot-start"
                type="time"
                step={60 * ((selectedSlot?.granularity_minutes ?? state.service?.slot_granularity_minutes ?? 30))}
                value={state.editor.startTime}
                onChange={(event) => onEditorChange?.({ startTime: event.target.value })}
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="scheduling-slot-end">{copy.slotEndLabel}</label>
              <input
                id="scheduling-slot-end"
                type="time"
                step={60 * ((selectedSlot?.granularity_minutes ?? state.service?.slot_granularity_minutes ?? 30))}
                value={state.editor.endTime}
                onChange={(event) => onEditorChange?.({ endTime: event.target.value })}
              />
            </div>
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="scheduling-repeat-mode">{copy.repeatLabel}</label>
              <select
                id="scheduling-repeat-mode"
                value={draft.recurrence.mode}
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    recurrence: {
                      ...current.recurrence,
                      mode: event.target.value as SchedulingBookingRecurrenceDraft['mode'],
                    },
                  }))
                }
              >
                <option value="none">{copy.repeatNever}</option>
                <option value="daily">{copy.repeatDaily}</option>
                <option value="weekly">{copy.repeatWeekly}</option>
                <option value="monthly">{copy.repeatMonthly}</option>
                <option value="custom">{copy.repeatCustom}</option>
              </select>
            </div>
            {draft.recurrence.mode !== 'none' ? (
              <div className="form-group grow">
                <label htmlFor="scheduling-repeat-interval">{copy.repeatIntervalLabel}</label>
                <input
                  id="scheduling-repeat-interval"
                  type="number"
                  min={1}
                  max={60}
                  value={draft.recurrence.interval}
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      recurrence: {
                        ...current.recurrence,
                        interval: event.target.value,
                      },
                    }))
                  }
                />
              </div>
            ) : null}
            {draft.recurrence.mode !== 'none' ? (
              <div className="form-group grow">
                <label htmlFor="scheduling-repeat-count">{copy.repeatCountLabel}</label>
                <input
                  id="scheduling-repeat-count"
                  type="number"
                  min={1}
                  max={60}
                  value={draft.recurrence.count}
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      recurrence: {
                        ...current.recurrence,
                        count: event.target.value,
                      },
                    }))
                  }
                />
              </div>
            ) : null}
          </div>

          {draft.recurrence.mode === 'custom' ? (
            <div className="form-group">
              <label htmlFor="scheduling-repeat-frequency">{copy.repeatFrequencyLabel}</label>
              <select
                id="scheduling-repeat-frequency"
                value={draft.recurrence.frequency}
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    recurrence: {
                      ...current.recurrence,
                      frequency: event.target.value as SchedulingBookingRecurrenceDraft['frequency'],
                    },
                  }))
                }
              >
                <option value="daily">{copy.repeatDaily}</option>
                <option value="weekly">{copy.repeatWeekly}</option>
                <option value="monthly">{copy.repeatMonthly}</option>
              </select>
            </div>
          ) : null}

          {draft.recurrence.mode !== 'none' && activeFrequency === 'weekly' ? (
            <div className="form-group">
              <label>{copy.repeatWeekdaysLabel}</label>
              <div className="modules-scheduling__weekday-picker">
                {[0, 1, 2, 3, 4, 5, 6].map((weekday) => {
                  const active = draft.recurrence.byWeekday.includes(weekday);
                  return (
                    <button
                      key={weekday}
                      type="button"
                      className={`modules-scheduling__weekday-btn ${active ? 'modules-scheduling__weekday-btn--active' : ''}`}
                      onClick={() =>
                        setDraft((current) => ({
                          ...current,
                          recurrence: {
                            ...current.recurrence,
                            byWeekday: active
                              ? current.recurrence.byWeekday.filter((item) => item !== weekday)
                              : [...current.recurrence.byWeekday, weekday].sort((left, right) => left - right),
                          },
                        }))
                      }
                    >
                      {formatSchedulingWeekdayNarrow(weekday, locale)}
                    </button>
                  );
                })}
              </div>
            </div>
          ) : null}

          {slotLoading ? <div className="modules-scheduling__validation-message">{copy.availableSlotLoading}</div> : null}
          {!slotLoading && state.validationMessage ? (
            <div className="modules-scheduling__validation-message">{state.validationMessage}</div>
          ) : null}

          <div className="form-group">
            <label htmlFor="scheduling-customer-name">{copy.customerNameLabel}</label>
            <input
              id="scheduling-customer-name"
              value={draft.customerName}
              onChange={(event) => setDraft((current) => ({ ...current, customerName: event.target.value }))}
              required
              autoComplete="name"
            />
          </div>

          <div className="modules-scheduling__form-row">
            <div className="form-group grow">
              <label htmlFor="scheduling-customer-phone">{copy.customerPhoneLabel}</label>
              <input
                id="scheduling-customer-phone"
                type="tel"
                value={draft.customerPhone}
                onChange={(event) => setDraft((current) => ({ ...current, customerPhone: event.target.value }))}
                required
                autoComplete="tel"
              />
            </div>
            <div className="form-group grow">
              <label htmlFor="scheduling-customer-email">{copy.customerEmailLabel}</label>
              <input
                id="scheduling-customer-email"
                type="email"
                value={draft.customerEmail}
                onChange={(event) => setDraft((current) => ({ ...current, customerEmail: event.target.value }))}
                autoComplete="email"
              />
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="scheduling-notes">{copy.notesLabel}</label>
            <textarea
              id="scheduling-notes"
              rows={4}
              value={draft.notes}
              onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))}
            />
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="modules-scheduling__backdrop app-modal-backdrop" role="presentation" onClick={() => void closeWithGuard()}>
      <div className="modules-scheduling__modal app-modal" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
        <div className="app-modal__header">
          {state.mode === 'details' ? (
            <div className="app-modal__title-block">
              <p className="app-modal__eyebrow">{copy.bookingSubtitleDetails}</p>
              <h3 className="app-modal__title">{copy.bookingTitleDetails}</h3>
              <p className="app-modal__subtitle">
                {`${formatSchedulingDateTime(state.booking.start_at, locale)} • ${state.resourceName ?? state.booking.resource_id}`}
              </p>
            </div>
          ) : (
            <div className="app-modal__footer-spacer" aria-hidden />
          )}
          <button type="button" className="app-modal__close" aria-label={copy.close} onClick={() => void closeWithGuard()}>
            ×
          </button>
        </div>

        {state.mode === 'create' ? (
          <form className="app-modal__body modules-scheduling__modal-form" onSubmit={handleCreate}>
            {onSwitchType ? (
              <SchedulingEntryTypeSwitcher
                active="booking"
                copy={copy}
                bookingEnabled
                onSwitch={onSwitchType}
              />
            ) : null}
            {renderCreateSummary()}

            <div className="app-modal__footer">
              <div className="app-modal__footer-spacer" aria-hidden />
              <button type="button" className="btn-secondary btn-sm app-modal__action" disabled={saving} onClick={() => void closeWithGuard()}>
                {copy.close}
              </button>
              <button type="submit" className="btn-primary btn-sm app-modal__action" disabled={saving || slotLoading || !state.slot}>
                {saving ? copy.saving : copy.create}
              </button>
            </div>
          </form>
        ) : (
          <>
            <div className="app-modal__body modules-scheduling__modal-body">
              <div className="modules-scheduling__detail-grid">
                <div className="modules-scheduling__detail">
                  <span>{copy.customerNameLabel}</span>
                  <strong>{state.booking.customer_name}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.customerPhoneLabel}</span>
                  <strong>{state.booking.customer_phone || '—'}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.customerEmailLabel}</span>
                  <strong>{state.booking.customer_email || '—'}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.statusLabel}</span>
                  <strong>{copy.statuses[state.booking.status]}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.serviceNameLabel}</span>
                  <strong>{state.service?.name ?? state.booking.service_id}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.resourceNameLabel}</span>
                  <strong>{state.resourceName ?? state.booking.resource_id}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.slotLabel}</span>
                  <strong>{formatSchedulingDateTime(state.booking.start_at, locale)}</strong>
                </div>
                <div className="modules-scheduling__detail">
                  <span>{copy.referenceLabel}</span>
                  <strong>{state.booking.reference}</strong>
                </div>
              </div>
              {state.booking.notes ? (
                <div className="modules-scheduling__notes">
                  <span>{copy.notesLabel}</span>
                  <p>{state.booking.notes}</p>
                </div>
              ) : null}
            </div>
            <div className="app-modal__footer">
              {actionButtons(state.booking.status).map((action) => (
                <button
                  key={action}
                  type="button"
                  className={action === 'cancel' || action === 'no_show' ? 'btn-danger btn-sm app-modal__action' : 'btn-secondary btn-sm app-modal__action'}
                  onClick={() => void onAction(action, state.booking)}
                  disabled={saving}
                >
                  {actionLabel(copy, action)}
                </button>
              ))}
              <div className="app-modal__footer-spacer" aria-hidden />
              <button type="button" className="btn-secondary btn-sm app-modal__action" disabled={saving} onClick={onClose}>
                {copy.close}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
