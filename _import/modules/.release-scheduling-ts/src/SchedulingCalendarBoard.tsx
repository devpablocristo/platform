import { useDeferredValue, useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { EventClickArg, EventContentArg, EventInput } from '@fullcalendar/core';
import FullCalendar from '@fullcalendar/react';
import { confirmAction } from '@devpablocristo/core-browser';
import {
  CalendarSurface,
  resolveInitialTimeGridScrollTime,
  resolveInitialTimeGridViewport,
  type CalendarSurfaceProps,
  type CalendarView,
} from '@devpablocristo/modules-calendar-board/next';
import type { SchedulingClient } from './client';
import {
  formatSchedulingClock,
  formatSchedulingCompactClock,
  resolveSchedulingCopyLocale,
} from './locale';
import {
  SchedulingBookingModal,
  type SchedulingBookingAction,
  type SchedulingBookingCreateEditor,
  type SchedulingBookingDraft,
  type SchedulingBookingModalState,
} from './SchedulingBookingModal';
import { SchedulingDateInput } from './SchedulingDateInput';
import {
  BlockedRangeModal,
  blockedRangeDraftFromRange,
  emptyBlockedRangeDraft,
  type BlockedRangeDraft,
  type BlockedRangeModalState,
} from './BlockedRangeModal';
import {
  emptyInternalEventDraft,
  internalEventDraftFromEntity,
  SchedulingInternalEventModal,
  type InternalEventDraft,
  type InternalEventModalState,
} from './SchedulingInternalEventModal';
import type { SchedulingEntryContext, SchedulingEntryType } from './types';
import {
  buildFullCalendarEventInputs,
  buildSchedulingCreateModalStateFromSlot,
  buildSchedulingCreateModalStateFromStart,
  buildSchedulingBusinessHours,
  buildSchedulingCalendarEvents,
  buildSchedulingDetailsModalState,
  buildSlotIdentity,
  buildSyntheticTimeSlotFromEditor,
  calendarSelectionAllowedWithBuffers,
  effectiveSchedulingTimeZone,
  resolveBookingDisplayTitle,
  toDateInputValue as toDateInputValueFromIso,
  toTimeInputValue,
} from './schedulingCalendarLogic';
import type {
  AvailabilityRule,
  BlockedRange,
  BlockedRangePayload,
  Booking,
  BookingStatus,
  Branch,
  CalendarEvent,
  CreateInternalEventPayload,
  InternalEvent,
  Resource,
  SchedulingCalendarCopy,
  Service,
  TimeSlot,
} from './types';

const schedulingKeys = {
  branches: ['scheduling', 'branches'] as const,
  services: ['scheduling', 'services'] as const,
  resources: (branchId: string | null) => ['scheduling', 'resources', branchId ?? 'all'] as const,
  availabilityRules: (branchId: string | null, resourceId: string | null) =>
    ['scheduling', 'availability-rules', branchId ?? 'none', resourceId ?? 'any'] as const,
  dashboard: (branchId: string | null, day: string) => ['scheduling', 'dashboard', branchId ?? 'all', day] as const,
  slots: (branchId: string | null, serviceId: string | null, resourceId: string | null, day: string) =>
    ['scheduling', 'slots', branchId ?? 'none', serviceId ?? 'none', resourceId ?? 'any', day] as const,
  visibleSlotsRange: (branchId: string | null, serviceId: string | null, resourceId: string | null, start: string, end: string) =>
    ['scheduling', 'visible-slots-range', branchId ?? 'none', serviceId ?? 'none', resourceId ?? 'any', start, end] as const,
  bookingsRange: (branchId: string | null, start: string, end: string) =>
    ['scheduling', 'bookings-range', branchId ?? 'none', start, end] as const,
  blockedRangesRange: (branchId: string | null, start: string, end: string) =>
    ['scheduling', 'blocked-ranges-range', branchId ?? 'none', start, end] as const,
  internalEventsRange: (branchId: string | null, start: string, end: string) =>
    ['scheduling', 'internal-events-range', branchId ?? 'none', start, end] as const,
};

/** Referencia estable cuando aún no hay datos de slots (evita loop en efecto de sync del modal). */
const emptySlotOptions: TimeSlot[] = [];

/** RFC3339 sin fracciones (alinea payloads con slots del API y expectativas de tests). */
function schedulingInstantRFC3339(iso: string): string {
  const ms = Date.parse(iso);
  if (!Number.isFinite(ms)) {
    return iso;
  }
  return new Date(ms).toISOString().replace(/\.\d{3}Z$/, 'Z');
}

export const schedulingCalendarCopyPresets: Record<'en' | 'es', SchedulingCalendarCopy> = {
  en: {
    branchLabel: 'Branch',
    serviceLabel: 'Service',
    resourceLabel: 'Resource',
    anyResource: 'Any resource',
    focusDateLabel: 'Date',
    summaryTitle: 'Daily overview',
    summaryBookings: 'Bookings',
    summaryConfirmed: 'Confirmed',
    summaryQueues: 'Active queues',
    summaryWaiting: 'Waiting tickets',
    slotsTitle: 'Available slots',
    slotsDescription: 'Slots are generated from availability rules, buffers, and conflicts.',
    slotsEmpty: 'No slots available for the current filters.',
    slotsLoading: 'Loading slots…',
    loading: 'Loading schedule…',
    unavailableTitle: 'Schedule is not configured yet',
    unavailableDescription: 'Create at least one active branch to use this calendar.',
    noServicesTitle: 'Bookings are not configured yet',
    noServicesDescription: 'The calendar is available, but you need at least one schedule-enabled service to create customer bookings.',
    loadErrorTitle: 'Could not load the schedule',
    loadErrorDescription:
      'The request for branches or services failed. Check your connection, sign in again, or confirm you have scheduling permissions (scheduling:read).',
    retryLoad: 'Retry',
    filtersTitle: 'Filters',
    filtersDescription: 'Branch and service are mandatory. Resource is optional.',
    timelineTitle: '',
    timelineDescription: '',
    openBooking: 'Book slot',
    titleLabel: 'Title',
    repeatLabel: 'Repeat',
    repeatNever: 'Does not repeat',
    repeatDaily: 'Daily',
    repeatWeekly: 'Weekly',
    repeatMonthly: 'Monthly',
    repeatCustom: 'Custom',
    repeatFrequencyLabel: 'Frequency',
    repeatIntervalLabel: 'Every',
    repeatCountLabel: 'Occurrences',
    repeatWeekdaysLabel: 'Weekdays',
    bookingTitleCreate: 'Create booking',
    bookingTitleDetails: 'Booking details',
    bookingSubtitleCreate: 'New booking',
    bookingSubtitleDetails: 'Current lifecycle state',
    availableSlotLabel: 'Available slot',
    availableSlotHint: 'Change the time, duration, or resource using an available slot.',
    availableSlotLoading: 'Checking availability…',
    unavailableSlotMessage: 'No available slot matches the selected date, time, duration, and resource.',
    slotSummaryTitle: 'Slot summary',
    bookingPreviewTitle: 'Booking preview',
    customerNameLabel: 'Customer name',
    customerPhoneLabel: 'Phone',
    customerEmailLabel: 'Email',
    notesLabel: 'Notes',
    statusLabel: 'Status',
    serviceNameLabel: 'Service',
    resourceNameLabel: 'Resource',
    slotLabel: 'Slot',
    slotStartLabel: 'Start',
    slotEndLabel: 'End',
    durationLabel: 'Duration',
    timezoneLabel: 'Timezone',
    occupiesLabel: 'Occupies',
    conflictLabel: 'Conflicts',
    slotRemainingLabel: 'Open spots',
    referenceLabel: 'Reference',
    close: 'Close',
    create: 'Save',
    saving: 'Saving…',
    cancelBooking: 'Cancel booking',
    confirmBooking: 'Confirm booking',
    checkInBooking: 'Check in',
    startService: 'Start service',
    completeBooking: 'Complete',
    noShowBooking: 'Mark as no-show',
    rescheduleBooking: 'Reschedule booking',
    dragRescheduleTitle: 'Move event',
    dragRescheduleDescription: 'Do you want to move this event to the new slot?',
    destructiveTitle: 'Confirm action',
    cancelActionDescription: 'This booking will be cancelled and removed from the active agenda.',
    noShowActionDescription: 'This booking will be marked as no-show.',
    closeDirtyTitle: 'Discard booking draft',
    closeDirtyDescription: 'You have unsaved changes in this booking draft.',
    keepEditing: 'Keep editing',
    discard: 'Discard',
    resizeLockedMessage: 'This calendar event keeps the duration defined by the service.',
    resizeBookingTitle: 'Resize booking',
    resizeBookingDescription: 'Do you want to update this booking with the new duration?',
    searchPlaceholder: 'Search events, customers, resources…',
    statuses: {
      hold: 'On hold',
      pending_confirmation: 'Pending confirmation',
      confirmed: 'Confirmed',
      checked_in: 'Checked in',
      in_service: 'In service',
      completed: 'Completed',
      cancelled: 'Cancelled',
      no_show: 'No show',
      expired: 'Expired',
    },
    blockedRangeAction: 'Block time',
    blockedRangeEyebrow: 'Calendar block',
    blockedRangeCreateTitle: 'Block time',
    blockedRangeEditTitle: 'Edit block',
    blockedRangeKindLabel: 'Type',
    blockedRangeKindOptions: {
      manual: 'Personal block',
      holiday: 'Holiday',
      maintenance: 'Maintenance',
      leave: 'Leave',
    },
    blockedRangeReasonLabel: 'Reason',
    blockedRangeReasonPlaceholder: 'Meeting with supplier, vacation…',
    blockedRangeCreate: 'Save block',
    blockedRangeUpdate: 'Update block',
    blockedRangeDelete: 'Delete block',
    blockedRangeDeleteTitle: 'Delete this block?',
    blockedRangeDeleteDescription: 'The blocked time will be removed and the slot becomes available again.',
    blockedRangeFallbackTitle: 'Blocked',
    internalEventFallbackTitle: 'Internal event',
    internalEventCreateAction: 'New event',
    internalEventModalCreateTitle: 'New internal event',
    internalEventModalEditTitle: 'Edit internal event',
    internalEventTitleLabel: 'Title',
    internalEventTitlePlaceholder: 'Team meeting, training, lunch…',
    internalEventDescriptionLabel: 'Notes',
    internalEventResourceLabel: 'Resource (optional)',
    internalEventResourceUnassigned: 'Personal time (no resource)',
    internalEventStartLabel: 'Start',
    internalEventEndLabel: 'End',
    internalEventStatusLabel: 'Status',
    internalEventStatusOptions: { scheduled: 'Scheduled', done: 'Done', cancelled: 'Cancelled' },
    internalEventVisibilityLabel: 'Visibility',
    internalEventVisibilityOptions: { team: 'Team', private: 'Private' },
    internalEventSave: 'Save event',
    internalEventDelete: 'Delete event',
    internalEventDeleteConfirm: 'Delete this event?',
    newEntryButton: 'New',
    newEntryMenuBooking: 'Customer booking',
    newEntryMenuInternalEvent: 'Internal event',
    newEntryMenuBlockedRange: 'Block schedule',
  },
  es: {
    branchLabel: 'Sucursal',
    serviceLabel: 'Servicio',
    resourceLabel: 'Recurso',
    anyResource: 'Cualquier recurso',
    focusDateLabel: 'Fecha foco',
    summaryTitle: 'Panorama diario',
    summaryBookings: 'Reservas',
    summaryConfirmed: 'Confirmadas',
    summaryQueues: 'Colas activas',
    summaryWaiting: 'Esperando',
    slotsTitle: 'Slots disponibles',
    slotsDescription: 'Los slots se calculan con disponibilidad, buffers y conflictos.',
    slotsEmpty: 'No hay slots disponibles para los filtros actuales.',
    slotsLoading: 'Cargando slots…',
    loading: 'Cargando agenda…',
    unavailableTitle: 'La agenda todavía no está configurada',
    unavailableDescription: 'Creá al menos una sucursal activa para usar este calendario.',
    noServicesTitle: 'Todavía no hay servicios configurados',
    noServicesDescription: 'La agenda ya se puede usar, pero necesitás al menos un servicio habilitado para crear turnos de clientes.',
    loadErrorTitle: 'No se pudo cargar la agenda',
    loadErrorDescription:
      'Falló la carga de sucursales o servicios. Revisá la conexión, volvé a iniciar sesión o confirmá que tu rol tenga permiso scheduling:read.',
    retryLoad: 'Reintentar',
    filtersTitle: 'Filtros',
    filtersDescription: 'Sucursal y servicio son obligatorios. Recurso es opcional.',
    timelineTitle: '',
    timelineDescription: '',
    openBooking: 'Reservar slot',
    titleLabel: 'Título',
    repeatLabel: 'Repetir',
    repeatNever: 'No se repite',
    repeatDaily: 'Diaria',
    repeatWeekly: 'Semanal',
    repeatMonthly: 'Mensual',
    repeatCustom: 'Personalizada',
    repeatFrequencyLabel: 'Frecuencia',
    repeatIntervalLabel: 'Cada',
    repeatCountLabel: 'Ocurrencias',
    repeatWeekdaysLabel: 'Días',
    bookingTitleCreate: 'Crear reserva',
    bookingTitleDetails: 'Detalle de la reserva',
    bookingSubtitleCreate: 'Nueva reserva',
    bookingSubtitleDetails: 'Estado actual del turno',
    availableSlotLabel: 'Slot disponible',
    availableSlotHint: 'Cambiá hora, duración o recurso usando un slot válido.',
    availableSlotLoading: 'Buscando disponibilidad...',
    unavailableSlotMessage: 'No hay un slot disponible para la fecha, hora, duración y recurso elegidos.',
    slotSummaryTitle: 'Resumen del slot',
    bookingPreviewTitle: 'Vista previa',
    customerNameLabel: 'Cliente',
    customerPhoneLabel: 'Teléfono',
    customerEmailLabel: 'Email',
    notesLabel: 'Notas',
    statusLabel: 'Estado',
    serviceNameLabel: 'Servicio',
    resourceNameLabel: 'Recurso',
    slotLabel: 'Slot',
    slotStartLabel: 'Inicio',
    slotEndLabel: 'Fin',
    durationLabel: 'Duración',
    timezoneLabel: 'Zona horaria',
    occupiesLabel: 'Bloquea',
    conflictLabel: 'Conflictos',
    slotRemainingLabel: 'Cupos',
    referenceLabel: 'Referencia',
    close: 'Cerrar',
    create: 'Guardar',
    saving: 'Guardando…',
    cancelBooking: 'Cancelar reserva',
    confirmBooking: 'Confirmar',
    checkInBooking: 'Check-in',
    startService: 'Iniciar atención',
    completeBooking: 'Completar',
    noShowBooking: 'Marcar no-show',
    rescheduleBooking: 'Reprogramar reserva',
    dragRescheduleTitle: 'Mover evento',
    dragRescheduleDescription: '¿Querés mover este evento al nuevo horario?',
    destructiveTitle: 'Confirmar acción',
    cancelActionDescription: 'La reserva se cancelará y saldrá de la agenda activa.',
    noShowActionDescription: 'La reserva se marcará como no-show.',
    closeDirtyTitle: 'Descartar borrador',
    closeDirtyDescription: 'Hay datos cargados sin guardar en esta reserva.',
    keepEditing: 'Seguir editando',
    discard: 'Descartar',
    resizeLockedMessage: 'Este evento del calendario mantiene la duración definida por el servicio.',
    resizeBookingTitle: 'Cambiar duración del turno',
    resizeBookingDescription: '¿Querés actualizar este turno con la nueva duración?',
    searchPlaceholder: 'Buscar eventos, clientes, recursos…',
    statuses: {
      hold: 'En espera',
      pending_confirmation: 'Pendiente de confirmación',
      confirmed: 'Confirmada',
      checked_in: 'Check-in',
      in_service: 'En atención',
      completed: 'Completada',
      cancelled: 'Cancelada',
      no_show: 'No-show',
      expired: 'Expirada',
    },
    blockedRangeAction: 'Bloquear horario',
    blockedRangeEyebrow: 'Bloqueo de agenda',
    blockedRangeCreateTitle: 'Bloquear horario',
    blockedRangeEditTitle: 'Editar bloqueo',
    blockedRangeKindLabel: 'Tipo',
    blockedRangeKindOptions: {
      manual: 'Bloqueo personal',
      holiday: 'Feriado',
      maintenance: 'Mantenimiento',
      leave: 'Licencia',
    },
    blockedRangeReasonLabel: 'Motivo',
    blockedRangeReasonPlaceholder: 'Reunión con proveedor, vacaciones…',
    blockedRangeCreate: 'Guardar bloqueo',
    blockedRangeUpdate: 'Actualizar bloqueo',
    blockedRangeDelete: 'Eliminar bloqueo',
    blockedRangeDeleteTitle: '¿Eliminar este bloqueo?',
    blockedRangeDeleteDescription: 'El horario bloqueado se eliminará y volverá a estar disponible.',
    blockedRangeFallbackTitle: 'Bloqueado',
    internalEventFallbackTitle: 'Evento interno',
    internalEventCreateAction: 'Nuevo evento',
    internalEventModalCreateTitle: 'Nuevo evento interno',
    internalEventModalEditTitle: 'Editar evento interno',
    internalEventTitleLabel: 'Título',
    internalEventTitlePlaceholder: 'Reunión equipo, capacitación, almuerzo…',
    internalEventDescriptionLabel: 'Notas',
    internalEventResourceLabel: 'Recurso (opcional)',
    internalEventResourceUnassigned: 'Tiempo personal (sin recurso)',
    internalEventStartLabel: 'Inicio',
    internalEventEndLabel: 'Fin',
    internalEventStatusLabel: 'Estado',
    internalEventStatusOptions: { scheduled: 'Programado', done: 'Realizado', cancelled: 'Cancelado' },
    internalEventVisibilityLabel: 'Visibilidad',
    internalEventVisibilityOptions: { team: 'Equipo', private: 'Privado' },
    internalEventSave: 'Guardar evento',
    internalEventDelete: 'Eliminar evento',
    internalEventDeleteConfirm: '¿Eliminar este evento?',
    newEntryButton: 'Nuevo',
    newEntryMenuBooking: 'Turno cliente',
    newEntryMenuInternalEvent: 'Evento interno',
    newEntryMenuBlockedRange: 'Bloquear horario',
  },
};

export type SchedulingCalendarProps = {
  client: SchedulingClient;
  searchQuery?: string;
  copy?: Partial<SchedulingCalendarCopy>;
  locale?: string;
  initialBranchId?: string;
  initialServiceId?: string;
  initialResourceId?: string | null;
  initialDate?: string;
  className?: string;
};

type VisibleRange = {
  start: Date;
  end: Date;
};

type CalendarDateClickArg = Parameters<NonNullable<CalendarSurfaceProps['onDateClick']>>[0];
type CalendarDateSelectArg = Parameters<NonNullable<CalendarSurfaceProps['onSelect']>>[0];
type CalendarEventDropArg = Parameters<NonNullable<CalendarSurfaceProps['onEventDrop']>>[0];
type CalendarEventResizeArg = Parameters<NonNullable<CalendarSurfaceProps['onEventResize']>>[0];
type CalendarSelectAllowArg = Parameters<NonNullable<CalendarSurfaceProps['selectAllow']>>[0];
type CalendarEventAllowArg = Parameters<NonNullable<CalendarSurfaceProps['eventAllow']>>[0];
type CalendarDraggedEventArg = Parameters<NonNullable<CalendarSurfaceProps['eventAllow']>>[1];

function startOfDay(value: Date): Date {
  return new Date(value.getFullYear(), value.getMonth(), value.getDate());
}

function toDateInputValue(value: Date): string {
  return value.toISOString().slice(0, 10);
}

// HH:MM en zona local del navegador (los modales de evento/bloqueo trabajan
// con horas locales para input type="time").
function toLocalTimeInputValue(value: Date): string {
  const hours = String(value.getHours()).padStart(2, '0');
  const minutes = String(value.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}

function statusClassName(status: BookingStatus): string {
  switch (status) {
    case 'confirmed':
      return 'modules-scheduling__badge modules-scheduling__badge--success';
    case 'checked_in':
    case 'in_service':
      return 'modules-scheduling__badge modules-scheduling__badge--attention';
    case 'cancelled':
    case 'no_show':
    case 'expired':
      return 'modules-scheduling__badge modules-scheduling__badge--critical';
    default:
      return 'modules-scheduling__badge modules-scheduling__badge--neutral';
  }
}

function eventColor(status: BookingStatus): string {
  switch (status) {
    case 'confirmed':
      return '#0f766e';
    case 'checked_in':
      return '#1d4ed8';
    case 'in_service':
      return '#7c3aed';
    case 'completed':
      return '#475569';
    case 'cancelled':
    case 'no_show':
    case 'expired':
      return '#b91c1c';
    default:
      return '#d97706';
  }
}

/** Días YYYY-MM-DD que cruzan el rango visible del calendario, en reloj de pared de la sucursal/recurso (no local del navegador). */
function buildVisibleDatesInZone(range: VisibleRange, wallTimeZone: string): string[] {
  const tz = wallTimeZone.trim() || 'UTC';
  const dates: string[] = [];
  const seen = new Set<string>();
  const stepMs = 60 * 60 * 1000;
  let t = range.start.getTime();
  const endMs = range.end.getTime();
  while (t < endMs) {
    const key = toDateInputValueFromIso(new Date(t).toISOString(), tz);
    if (!seen.has(key)) {
      seen.add(key);
      dates.push(key);
    }
    t += stepMs;
  }
  return dates.sort();
}

function uniqueBookings(items: Booking[]): Booking[] {
  const seen = new Map<string, Booking>();
  for (const item of items) {
    seen.set(item.id, item);
  }
  return Array.from(seen.values()).sort((left, right) => left.start_at.localeCompare(right.start_at));
}

function defaultVisibleRange(): VisibleRange {
  const today = new Date();
  const start = startOfDay(today);
  const end = new Date(start);
  end.setDate(end.getDate() + 7);
  return { start, end };
}

function matchesSearch(searchQuery: string, pieces: Array<string | undefined | null>): boolean {
  if (!searchQuery.trim()) {
    return true;
  }
  const normalized = searchQuery.trim().toLocaleLowerCase();
  return pieces.some((piece) => piece?.toLocaleLowerCase().includes(normalized));
}

function matchesCreateEditorSlot(
  editor: SchedulingBookingCreateEditor,
  slot: TimeSlot,
  branchTimezone?: string | null,
): boolean {
  const tz = slot.timezone?.trim() || branchTimezone?.trim() || null;
  return (
    slot.resource_id === editor.resourceId &&
    toDateInputValueFromIso(slot.start_at, tz) === editor.date &&
    toTimeInputValue(slot.start_at, tz) === editor.startTime &&
    toTimeInputValue(slot.end_at, tz) === editor.endTime
  );
}

export function SchedulingCalendar({
  client,
  searchQuery = '',
  copy: copyOverrides,
  locale = 'en',
  initialBranchId,
  initialServiceId,
  initialResourceId = null,
  initialDate,
  className = '',
}: SchedulingCalendarProps) {
  const queryClient = useQueryClient();
  const calendarRef = useRef<FullCalendar>(null);
  const [view, setView] = useState<CalendarView>('timeGridWeek');
  const [calendarTitle, setCalendarTitle] = useState('');
  const [visibleRange, setVisibleRange] = useState<VisibleRange>(defaultVisibleRange);
  const [focusedDate, setFocusedDate] = useState(initialDate ?? toDateInputValue(new Date()));
  const [selectedBranchId, setSelectedBranchId] = useState<string | null>(initialBranchId ?? null);
  const [selectedServiceId, setSelectedServiceId] = useState<string | null>(initialServiceId ?? null);
  const [selectedResourceId, setSelectedResourceId] = useState<string | null>(initialResourceId ?? null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [modalState, setModalState] = useState<SchedulingBookingModalState>({ open: false });
  const [blockedModalState, setBlockedModalState] = useState<BlockedRangeModalState>({ open: false });
  // Contexto del último gesto de creación (click/select/+Nuevo). Lo guardamos
  // en un ref para que el switcher de tipo dentro de los modales pueda preservar
  // fecha/hora al cambiar de tipo sin perder lo que el usuario eligió.
  const lastEntryContextRef = useRef<SchedulingEntryContext | null>(null);
  const [internalEventModalState, setInternalEventModalState] = useState<InternalEventModalState>({ open: false });
  const deferredSearch = useDeferredValue(searchQuery);

  // useMutation ejecuta mutationFn en un tick posterior; sin ref el closure puede ver modalState cerrado o slot null.
  const createBookingContextRef = useRef({
    branchId: selectedBranchId,
    serviceId: selectedServiceId,
    modal: modalState,
  });
  createBookingContextRef.current = {
    branchId: selectedBranchId,
    serviceId: selectedServiceId,
    modal: modalState,
  };

  const copy = { ...schedulingCalendarCopyPresets[resolveSchedulingCopyLocale(locale)], ...copyOverrides };

  const branchesQuery = useQuery<Branch[]>({
    queryKey: schedulingKeys.branches,
    queryFn: () => client.listBranches(),
    staleTime: 60_000,
  });

  const servicesQuery = useQuery<Service[]>({
    queryKey: schedulingKeys.services,
    queryFn: () => client.listServices(),
    staleTime: 60_000,
  });

  const resourcesQuery = useQuery<Resource[]>({
    queryKey: schedulingKeys.resources(selectedBranchId),
    queryFn: () => client.listResources(selectedBranchId),
    enabled: Boolean(selectedBranchId),
    staleTime: 60_000,
  });

  const schedulingCalendarTimeZone = useMemo(() => {
    const branchList = branchesQuery.data ?? [];
    const branch = branchList.find((b) => b.id === selectedBranchId) ?? null;
    if (!branch) {
      return '';
    }
    const resourceList = resourcesQuery.data ?? [];
    const resource = selectedResourceId ? resourceList.find((r) => r.id === selectedResourceId) ?? null : null;
    return effectiveSchedulingTimeZone(resource, branch);
  }, [branchesQuery.data, selectedBranchId, resourcesQuery.data, selectedResourceId]);

  const availabilityRulesQuery = useQuery<AvailabilityRule[]>({
    queryKey: schedulingKeys.availabilityRules(selectedBranchId, selectedResourceId),
    queryFn: () => client.listAvailabilityRules(selectedBranchId, selectedResourceId),
    enabled: Boolean(selectedBranchId),
    staleTime: 60_000,
  });

  const createEditorDate = modalState.open && modalState.mode === 'create' ? modalState.editor.date : focusedDate;

  const createSlotsQuery = useQuery<TimeSlot[]>({
    queryKey: schedulingKeys.slots(selectedBranchId, selectedServiceId, null, createEditorDate),
    queryFn: () =>
      client.listSlots({
        branchId: selectedBranchId ?? '',
        serviceId: selectedServiceId ?? '',
        resourceId: null,
        date: createEditorDate,
      }),
    enabled: Boolean(selectedBranchId && selectedServiceId && modalState.open && modalState.mode === 'create'),
    staleTime: 10_000,
  });

  const rangeDates = useMemo(
    () => buildVisibleDatesInZone(visibleRange, schedulingCalendarTimeZone.trim() || 'UTC'),
    [visibleRange, schedulingCalendarTimeZone],
  );

  const visibleSlotsQuery = useQuery<TimeSlot[]>({
    queryKey: schedulingKeys.visibleSlotsRange(
      selectedBranchId,
      selectedServiceId,
      selectedResourceId,
      rangeDates[0] ?? focusedDate,
      rangeDates[rangeDates.length - 1] ?? focusedDate,
    ),
    queryFn: async () => {
      if (!selectedBranchId || !selectedServiceId || rangeDates.length === 0) {
        return [];
      }
      const batches = await Promise.all(
        rangeDates.map((date) =>
          client.listSlots({
            branchId: selectedBranchId,
            serviceId: selectedServiceId,
            resourceId: selectedResourceId,
            date,
          }),
        ),
      );
      const unique = new Map<string, TimeSlot>();
      for (const slot of batches.flat()) {
        unique.set(buildSlotIdentity(slot.resource_id, slot.start_at, slot.end_at), slot);
      }
      return Array.from(unique.values()).sort((left, right) => left.start_at.localeCompare(right.start_at));
    },
    enabled: Boolean(selectedBranchId && selectedServiceId && rangeDates.length > 0),
    staleTime: 10_000,
  });

  const bookingsQuery = useQuery<Booking[]>({
    queryKey: schedulingKeys.bookingsRange(
      selectedBranchId,
      rangeDates[0] ?? focusedDate,
      rangeDates[rangeDates.length - 1] ?? focusedDate,
    ),
    queryFn: async () => {
      if (!selectedBranchId || rangeDates.length === 0) {
        return [];
      }
      const batches = await Promise.all(
        rangeDates.map((date) =>
          client.listBookings({
            branchId: selectedBranchId,
            date,
            limit: 200,
          }),
        ),
      );
      return uniqueBookings(batches.flat());
    },
    enabled: Boolean(selectedBranchId && rangeDates.length > 0),
    staleTime: 10_000,
  });

  const blockedRangesQuery = useQuery<BlockedRange[]>({
    queryKey: schedulingKeys.blockedRangesRange(
      selectedBranchId,
      rangeDates[0] ?? focusedDate,
      rangeDates[rangeDates.length - 1] ?? focusedDate,
    ),
    queryFn: async () => {
      if (!selectedBranchId || rangeDates.length === 0) {
        return [];
      }
      const batches = await Promise.all(
        rangeDates.map((date) =>
          client.listBlockedRanges({
            branchId: selectedBranchId,
            date,
          }),
        ),
      );
      const seen = new Map<string, BlockedRange>();
      for (const item of batches.flat()) {
        seen.set(item.id, item);
      }
      return Array.from(seen.values()).sort((left, right) => left.start_at.localeCompare(right.start_at));
    },
    enabled: Boolean(selectedBranchId && rangeDates.length > 0),
    staleTime: 10_000,
  });

  // Eventos internos del rango visible. La query usa from/to en RFC3339 para
  // coincidir con el shape del backend (ListCalendarEvents acepta filtros
  // temporales). Sin selectedBranchId, queda deshabilitada y no dispara fetch.
  const internalEventsQuery = useQuery<InternalEvent[]>({
    queryKey: schedulingKeys.internalEventsRange(
      selectedBranchId,
      rangeDates[0] ?? focusedDate,
      rangeDates[rangeDates.length - 1] ?? focusedDate,
    ),
    queryFn: async () => {
      if (!selectedBranchId || rangeDates.length === 0) {
        return [];
      }
      const firstDate = rangeDates[0] ?? focusedDate;
      const lastDate = rangeDates[rangeDates.length - 1] ?? focusedDate;
      const fromIso = `${firstDate}T00:00:00Z`;
      // El último día del rango debe incluirse completo, así que el `to`
      // exclusivo cae al inicio del día siguiente.
      const lastDateObj = new Date(`${lastDate}T00:00:00Z`);
      lastDateObj.setUTCDate(lastDateObj.getUTCDate() + 1);
      const toIso = lastDateObj.toISOString().replace(/\.\d{3}Z$/, 'Z');
      return client.listInternalEvents({
        branch_id: selectedBranchId,
        from: fromIso,
        to: toIso,
      });
    },
    enabled: Boolean(selectedBranchId && rangeDates.length > 0),
    staleTime: 10_000,
  });

  const branches = branchesQuery.data ?? [];
  const services = servicesQuery.data ?? [];
  const resources = resourcesQuery.data ?? [];
  const availabilityRules = availabilityRulesQuery.data ?? [];

  const scheduleServices = useMemo(
    () =>
      services.filter(
        (service) =>
          service.active && (service.fulfillment_mode === 'schedule' || service.fulfillment_mode === 'hybrid'),
      ),
    [services],
  );

  const selectedBranch = branches.find((branch) => branch.id === selectedBranchId) ?? null;

  const filteredResources = useMemo(() => {
    const branchResources = resources.filter((resource) => resource.active);
    const service = scheduleServices.find((candidate) => candidate.id === selectedServiceId);
    if (!service || !service.resource_ids?.length) {
      return branchResources;
    }
    const allowed = new Set(service.resource_ids);
    return branchResources.filter((resource) => allowed.has(resource.id));
  }, [resources, scheduleServices, selectedServiceId]);

  const selectedService = scheduleServices.find((service) => service.id === selectedServiceId) ?? null;
  const selectedResource = filteredResources.find((resource) => resource.id === selectedResourceId) ?? null;

  const visibleSlots = visibleSlotsQuery.data ?? [];
  const visibleSlotsByDate = useMemo(() => {
    const branchTz = selectedBranch?.timezone?.trim() || null;
    const grouped = new Map<string, TimeSlot[]>();
    for (const slot of visibleSlots) {
      const day = toDateInputValueFromIso(slot.start_at, slot.timezone?.trim() || branchTz);
      const items = grouped.get(day);
      if (items) {
        items.push(slot);
      } else {
        grouped.set(day, [slot]);
      }
    }
    return grouped;
  }, [visibleSlots, selectedBranch?.timezone]);

  const businessHours = useMemo(
    () => buildSchedulingBusinessHours(availabilityRules, selectedResourceId),
    [availabilityRules, selectedResourceId],
  );

  useEffect(() => {
    if (selectedBranchId) {
      return;
    }
    const preferred = branches.find((branch) => branch.active) ?? branches[0];
    if (preferred) {
      setSelectedBranchId(preferred.id);
    }
  }, [branches, selectedBranchId]);

  useEffect(() => {
    if (selectedServiceId && scheduleServices.some((service) => service.id === selectedServiceId)) {
      return;
    }
    const preferred = scheduleServices[0];
    setSelectedServiceId(preferred?.id ?? null);
  }, [scheduleServices, selectedServiceId]);

  useEffect(() => {
    if (!selectedResourceId) {
      return;
    }
    if (!filteredResources.some((resource) => resource.id === selectedResourceId)) {
      setSelectedResourceId(null);
    }
  }, [filteredResources, selectedResourceId]);

  const invalidateSchedule = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: schedulingKeys.dashboard(selectedBranchId, focusedDate) }),
      queryClient.invalidateQueries({ queryKey: ['scheduling', 'slots'] }),
      queryClient.invalidateQueries({ queryKey: ['scheduling', 'visible-slots-range'] }),
      queryClient.invalidateQueries({
        queryKey: schedulingKeys.bookingsRange(
          selectedBranchId,
          rangeDates[0] ?? focusedDate,
          rangeDates[rangeDates.length - 1] ?? focusedDate,
        ),
      }),
      queryClient.invalidateQueries({ queryKey: ['scheduling', 'blocked-ranges-range'] }),
      queryClient.invalidateQueries({ queryKey: ['scheduling', 'internal-events-range'] }),
    ]);
  };

  const buildBlockedRangePayload = (draft: BlockedRangeDraft, branchId: string): BlockedRangePayload => {
    const startAt = new Date(`${draft.date}T${draft.startTime}:00`).toISOString();
    const endAt = new Date(`${draft.date}T${draft.endTime}:00`).toISOString();
    return {
      branch_id: branchId,
      resource_id: draft.resourceId || null,
      kind: draft.kind,
      reason: draft.reason.trim(),
      start_at: startAt,
      end_at: endAt,
      all_day: false,
    };
  };

  const createBlockedRangeMutation = useMutation<BlockedRange, Error, BlockedRangeDraft>({
    mutationFn: async (draft) => {
      if (!selectedBranchId) {
        throw new Error('missing branch');
      }
      return client.createBlockedRange(buildBlockedRangePayload(draft, selectedBranchId));
    },
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setBlockedModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const updateBlockedRangeMutation = useMutation<BlockedRange, Error, { id: string; draft: BlockedRangeDraft }>({
    mutationFn: async ({ id, draft }) => {
      if (!selectedBranchId) {
        throw new Error('missing branch');
      }
      return client.updateBlockedRange(id, buildBlockedRangePayload(draft, selectedBranchId));
    },
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setBlockedModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const deleteBlockedRangeMutation = useMutation<void, Error, string>({
    mutationFn: async (id) => {
      await client.deleteBlockedRange(id);
    },
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setBlockedModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  // ── internal events (agenda interna) ─────────────────────────────────────
  // El draft viene con `date`/`startTime`/`endTime` como reloj de pared en la
  // zona del navegador (mismo patrón que BlockedRange). El backend valida en
  // UTC, así que serializamos con `new Date(...).toISOString()` para que el
  // server reciba un instante ISO inequívoco.
  const buildInternalEventPayload = (draft: InternalEventDraft): CreateInternalEventPayload => {
    const startAt = new Date(`${draft.date}T${draft.startTime}:00`).toISOString();
    const endAt = new Date(`${draft.date}T${draft.endTime}:00`).toISOString();
    return {
      branch_id: selectedBranchId || null,
      resource_id: draft.resourceId || null,
      title: draft.title.trim(),
      description: draft.description.trim() || undefined,
      start_at: startAt,
      end_at: endAt,
      status: draft.status,
      visibility: draft.visibility,
    };
  };

  const createInternalEventMutation = useMutation<InternalEvent, Error, InternalEventDraft>({
    mutationFn: async (draft) => client.createInternalEvent(buildInternalEventPayload(draft)),
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setInternalEventModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const updateInternalEventMutation = useMutation<InternalEvent, Error, { id: string; draft: InternalEventDraft }>({
    mutationFn: async ({ id, draft }) => client.updateInternalEvent(id, buildInternalEventPayload(draft)),
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setInternalEventModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const deleteInternalEventMutation = useMutation<void, Error, string>({
    mutationFn: async (id) => {
      await client.deleteInternalEvent(id);
    },
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setInternalEventModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const createBookingMutation = useMutation<Booking, Error, SchedulingBookingDraft>({
    mutationFn: async (draft: SchedulingBookingDraft) => {
      const { branchId, serviceId, modal } = createBookingContextRef.current;
      if (!branchId || !serviceId || !modal.open || modal.mode !== 'create' || !modal.slot) {
        throw new Error('missing scheduling context');
      }
      const slot = modal.slot;
      const recurrenceMode = draft.recurrence.mode;
      const interval = Number.parseInt(draft.recurrence.interval, 10);
      const count = Number.parseInt(draft.recurrence.count, 10);
      const recurrence =
        recurrenceMode === 'none'
          ? undefined
          : {
              freq: (recurrenceMode === 'custom'
                ? draft.recurrence.frequency
                : recurrenceMode) as 'daily' | 'weekly' | 'monthly',
              interval: Number.isFinite(interval) && interval > 0 ? interval : 1,
              count: Number.isFinite(count) && count > 0 ? count : 8,
              by_weekday:
                (recurrenceMode === 'custom' ? draft.recurrence.frequency : recurrenceMode) === 'weekly'
                  ? draft.recurrence.byWeekday.length
                    ? draft.recurrence.byWeekday
                    : [new Date(slot.start_at).getUTCDay()]
                  : undefined,
            };
      return client.createBooking({
        branch_id: branchId,
        service_id: serviceId,
        resource_id: slot.resource_id,
        customer_name: draft.customerName.trim(),
        customer_phone: draft.customerPhone.trim(),
        customer_email: draft.customerEmail.trim() || undefined,
        start_at: schedulingInstantRFC3339(slot.start_at),
        ...(recurrence ? {} : { end_at: schedulingInstantRFC3339(slot.end_at) }),
        notes: draft.notes.trim() || undefined,
        metadata: draft.title.trim() ? { title: draft.title.trim() } : undefined,
        recurrence,
        source: 'admin',
      });
    },
    onMutate: () => setActionError(null),
    onSuccess: async () => {
      setModalState({ open: false });
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const bookingActionMutation = useMutation<Booking, Error, { action: SchedulingBookingAction; booking: Booking }>({
    mutationFn: async ({ action, booking }) => {
      switch (action) {
        case 'confirm':
          return client.confirmBooking(booking.id);
        case 'cancel':
          return client.cancelBooking(booking.id, 'Cancelled from scheduling calendar');
        case 'check_in':
          return client.checkInBooking(booking.id);
        case 'start_service':
          return client.startService(booking.id);
        case 'complete':
          return client.completeBooking(booking.id);
        case 'no_show':
          return client.markBookingNoShow(booking.id, 'Marked as no-show from scheduling calendar');
        default:
          throw new Error(`unsupported action: ${String(action)}`);
      }
    },
    onMutate: () => setActionError(null),
    onSuccess: async (booking: Booking) => {
      setModalState(buildSchedulingDetailsModalState(booking, scheduleServices, filteredResources));
      await invalidateSchedule();
    },
    onError: (error: Error) => setActionError(error.message),
  });

  const filteredBookings = useMemo(() => {
    const source = bookingsQuery.data ?? [];
    return source.filter((booking) => {
      if (selectedServiceId && booking.service_id !== selectedServiceId) {
        return false;
      }
      if (selectedResourceId && booking.resource_id !== selectedResourceId) {
        return false;
      }
      const serviceName = scheduleServices.find((service) => service.id === booking.service_id)?.name;
      const resourceName = filteredResources.find((resource) => resource.id === booking.resource_id)?.name;
      return matchesSearch(deferredSearch, [
        resolveBookingDisplayTitle(booking),
        booking.customer_name,
        booking.customer_phone,
        booking.customer_email,
        booking.reference,
        booking.notes,
        serviceName,
        resourceName,
        copy.statuses[booking.status],
      ]);
    });
  }, [bookingsQuery.data, selectedServiceId, selectedResourceId, scheduleServices, filteredResources, deferredSearch, copy.statuses]);

  /** Reservas para colisiones en el calendario (sin filtro de búsqueda: no ocultar conflictos). */
  const collisionBookings = useMemo(() => {
    const source = bookingsQuery.data ?? [];
    return source.filter((booking) => {
      if (selectedServiceId && booking.service_id !== selectedServiceId) {
        return false;
      }
      if (selectedResourceId && booking.resource_id !== selectedResourceId) {
        return false;
      }
      return true;
    });
  }, [bookingsQuery.data, selectedServiceId, selectedResourceId]);

  const filteredFreeSlots = useMemo(() => {
    return visibleSlots.filter(
      (slot) =>
        slot.remaining > 0 &&
        matchesSearch(deferredSearch, [
          slot.resource_name,
          formatSchedulingClock(slot.start_at, locale),
          formatSchedulingClock(slot.end_at, locale),
          selectedService?.name,
        ]),
    );
  }, [visibleSlots, deferredSearch, selectedService?.name, locale]);

  // Los huecos libres no se pintan como eventos: ensucian la grilla y duplican la
  // celda vacía nativa de FullCalendar, que ya permite crear reservas con click.
  const freeSlotCalendarEvents = useMemo<EventInput[]>(() => [], []);

  const calendarEvents = useMemo(
    () => buildSchedulingCalendarEvents(filteredBookings, scheduleServices, filteredResources, eventColor),
    [filteredBookings, scheduleServices, filteredResources],
  );

  const blockedRanges = blockedRangesQuery.data ?? [];

  const blockedRangeEventInputs = useMemo<EventInput[]>(
    () =>
      blockedRanges.map((range) => ({
        id: `blocked:${range.id}`,
        title: range.reason || copy.blockedRangeKindOptions[range.kind] || copy.blockedRangeFallbackTitle,
        start: range.start_at,
        end: range.end_at,
        backgroundColor: '#9ca3af',
        borderColor: '#6b7280',
        textColor: '#111827',
        classNames: ['modules-scheduling__calendar-event--blocked'],
        editable: true,
        durationEditable: true,
        extendedProps: {
          blockedRange: range,
        },
      })),
    [blockedRanges, copy.blockedRangeKindOptions, copy.blockedRangeFallbackTitle],
  );

  // Eventos internos (reuniones, capacitaciones, tiempo personal del owner).
  // Color índigo distintivo: ni el verde/naranja de los turnos ni el gris de los
  // bloqueos. Cancelados en gris claro tachado.
  const internalEvents = internalEventsQuery.data ?? [];
  const internalEventInputs = useMemo<EventInput[]>(
    () =>
      internalEvents.map((event) => {
        const cancelled = event.status === 'cancelled';
        return {
          id: `internal-event:${event.id}`,
          title: event.title || copy.internalEventFallbackTitle,
          start: event.start_at,
          end: event.end_at,
          backgroundColor: cancelled ? '#e5e7eb' : '#6366f1',
          borderColor: cancelled ? '#9ca3af' : '#4338ca',
          textColor: cancelled ? '#6b7280' : '#ffffff',
          classNames: cancelled
            ? [
                'modules-scheduling__calendar-event--internal',
                'modules-scheduling__calendar-event--internal-cancelled',
              ]
            : ['modules-scheduling__calendar-event--internal'],
          editable: true,
          durationEditable: true,
          extendedProps: {
            internalEvent: event,
          },
        };
      }),
    [internalEvents, copy.internalEventFallbackTitle],
  );

  const fullCalendarEventInputs = useMemo(
    () => [
      ...freeSlotCalendarEvents,
      ...blockedRangeEventInputs,
      ...internalEventInputs,
      ...buildFullCalendarEventInputs(calendarEvents),
    ],
    [freeSlotCalendarEvents, blockedRangeEventInputs, internalEventInputs, calendarEvents],
  );

  const resolveDaySlots = (value: Date): TimeSlot[] =>
    visibleSlotsByDate.get(
      toDateInputValueFromIso(value.toISOString(), schedulingCalendarTimeZone.trim() || 'UTC'),
    ) ?? [];

  const handleSelectAllow = (info: CalendarSelectAllowArg) => {
    if (!selectedService || !info.start || !info.end) {
      return false;
    }
    if (info.end.getTime() <= info.start.getTime()) {
      return false;
    }
    const resourceIds = filteredResources.map((r) => r.id);
    return calendarSelectionAllowedWithBuffers({
      start: info.start,
      end: info.end,
      service: selectedService,
      resourceIds,
      bookings: collisionBookings,
      blockedRanges,
    });
  };

  const handleEventAllow = (dropInfo: CalendarEventAllowArg, draggedEvent: CalendarDraggedEventArg) => {
    if (!draggedEvent || !dropInfo.start || !dropInfo.end) {
      return false;
    }
    const ext = draggedEvent.extendedProps as {
      blockedRange?: BlockedRange;
      calendarEvent?: CalendarEvent;
    };
    const blockedRange = ext.blockedRange;
    const sourceBooking = ext.calendarEvent?.sourceBooking;
    if (blockedRange) {
      const resourceIds = blockedRange.resource_id
        ? [blockedRange.resource_id]
        : filteredResources.map((r) => r.id);
      if (!resourceIds.length) {
        return false;
      }
      return calendarSelectionAllowedWithBuffers({
        start: dropInfo.start,
        end: dropInfo.end,
        service: selectedService,
        resourceIds,
        bookings: collisionBookings,
        blockedRanges,
        occupancyIsExplicit: true,
        excludeBlockedRangeId: blockedRange.id,
      });
    }
    if (sourceBooking) {
      const dragService =
        scheduleServices.find((s) => s.id === sourceBooking.service_id) ?? selectedService;
      if (!dragService) {
        return false;
      }
      return calendarSelectionAllowedWithBuffers({
        start: dropInfo.start,
        end: dropInfo.end,
        service: dragService,
        resourceIds: [sourceBooking.resource_id],
        bookings: collisionBookings,
        blockedRanges,
        excludeBookingId: sourceBooking.id,
      });
    }
    return true;
  };

  const renderEventContent = (info: EventContentArg) => {
    const freeSlot = info.event.extendedProps.freeTimeSlot as TimeSlot | undefined;
    if (freeSlot) {
      const listView = info.view.type.startsWith('list');
      const rangeLabel = `${formatSchedulingClock(freeSlot.start_at, locale)} – ${formatSchedulingClock(freeSlot.end_at, locale)}`;
      const compactRange = `${formatSchedulingCompactClock(freeSlot.start_at, locale)}-${formatSchedulingCompactClock(freeSlot.end_at, locale)}`;
      const secondary = listView
        ? `${rangeLabel} · ${freeSlot.resource_name} · ${copy.slotRemainingLabel}: ${freeSlot.remaining}`
        : `${compactRange} · ${freeSlot.resource_name} · ${copy.slotRemainingLabel} ${freeSlot.remaining}`;
      return (
        <div
          className={`modules-scheduling__calendar-event modules-scheduling__calendar-event--free-slot${listView ? ' modules-scheduling__calendar-event--list' : ''}`}
        >
          <div className="modules-scheduling__calendar-event-top">
            <strong>{copy.openBooking}</strong>
          </div>
          <div className="modules-scheduling__calendar-event-meta">
            <span>{secondary}</span>
          </div>
        </div>
      );
    }
    const calendarEvent = info.event.extendedProps.calendarEvent as CalendarEvent | undefined;
    if (!calendarEvent) {
      return <span>{info.event.title}</span>;
    }
    const statusLabel = copy.statuses[calendarEvent.status];
    const listTimeRange = `${formatSchedulingClock(calendarEvent.start_at, locale)} - ${formatSchedulingClock(calendarEvent.end_at, locale)}`;
    const compactTimeRange = `${formatSchedulingCompactClock(calendarEvent.start_at, locale)}-${formatSchedulingCompactClock(calendarEvent.end_at, locale)}`;
    const listView = info.view.type.startsWith('list');
    const monthView = info.view.type === 'dayGridMonth';
    const contextLabel = selectedResourceId
      ? calendarEvent.serviceName ?? calendarEvent.resourceName
      : calendarEvent.resourceName ?? calendarEvent.serviceName;
    const secondaryLine = listView
      ? [listTimeRange, statusLabel, calendarEvent.resourceName, calendarEvent.serviceName].filter(Boolean).join(' · ')
      : [compactTimeRange, monthView ? undefined : contextLabel].filter(Boolean).join(' · ');

    return (
      <div className={`modules-scheduling__calendar-event ${listView ? 'modules-scheduling__calendar-event--list' : ''}`}>
        <div className="modules-scheduling__calendar-event-top">
          <strong>{calendarEvent.title}</strong>
          {listView ? <span className={statusClassName(calendarEvent.status)}>{statusLabel}</span> : null}
        </div>
        {secondaryLine ? (
          <div className="modules-scheduling__calendar-event-meta">
            <span>{secondaryLine}</span>
          </div>
        ) : null}
      </div>
    );
  };

  const openCreateBookingModal = (slot: TimeSlot, resourceName?: string) => {
    const tz = slot.timezone?.trim() || selectedBranch?.timezone?.trim() || null;
    const dayKey = toDateInputValueFromIso(slot.start_at, tz);
    const sameDaySlots = visibleSlots.filter((s) => toDateInputValueFromIso(s.start_at, tz) === dayKey);
    setModalState(
      buildSchedulingCreateModalStateFromSlot(
        slot,
        selectedService,
        sameDaySlots.length > 0 ? sameDaySlots : [slot],
        filteredResources,
        resourceName,
        selectedBranch?.timezone,
      ),
    );
  };

  const handleCreateEditorChange = (editorPatch: Partial<SchedulingBookingCreateEditor>) => {
    setModalState((current) => {
      if (!(current.open && current.mode === 'create')) {
        return current;
      }
      return {
        ...current,
        editor: {
          ...current.editor,
          ...editorPatch,
        },
      };
    });
  };

  useEffect(() => {
    if (!(modalState.open && modalState.mode === 'create')) {
      return;
    }
    const slotOptions = createSlotsQuery.data ?? emptySlotOptions;

    setModalState((current) => {
      if (!(current.open && current.mode === 'create')) {
        return current;
      }
      const editor = current.editor;
      const resourceMatchInner = filteredResources.find((resource) => resource.id === editor.resourceId);
      const branchTz = selectedBranch?.timezone;
      const matchedOrSynthetic =
        slotOptions.find((s) => matchesCreateEditorSlot(editor, s, branchTz)) ??
        (current.service
          ? buildSyntheticTimeSlotFromEditor(editor, current.service, resourceMatchInner, selectedBranch)
          : null);
      // Durante refetch: solo conservar el slot anterior si el editor sigue alineado (evita guardar horario viejo).
      const keepStaleSlot =
        createSlotsQuery.isFetching &&
        current.slot &&
        matchesCreateEditorSlot(editor, current.slot, branchTz);
      const slotToSet = matchedOrSynthetic ?? (keepStaleSlot ? current.slot : null);

      const nextResourceName =
        resourceMatchInner?.name ??
        slotOptions.find((slot) => slot.resource_id === editor.resourceId)?.resource_name ??
        current.resourceName;

      const nextValidationMessage =
        slotToSet || createSlotsQuery.isFetching ? null : copy.unavailableSlotMessage;
      const sameSlot =
        current.slot?.resource_id === slotToSet?.resource_id &&
        current.slot?.start_at === slotToSet?.start_at &&
        current.slot?.end_at === slotToSet?.end_at;
      if (
        sameSlot &&
        current.slotOptions === slotOptions &&
        current.resourceName === nextResourceName &&
        current.validationMessage === nextValidationMessage &&
        current.resourceOptions.length === filteredResources.length
      ) {
        return current;
      }
      return {
        ...current,
        slot: slotToSet,
        slotOptions,
        resourceOptions: filteredResources.map((resource) => ({
          id: resource.id,
          name: resource.name,
          timezone: resource.timezone,
        })),
        resourceName: nextResourceName,
        validationMessage: nextValidationMessage,
      };
    });
  }, [
    modalState,
    createSlotsQuery.data,
    createSlotsQuery.isFetching,
    copy.unavailableSlotMessage,
    filteredResources,
    selectedBranch,
  ]);

  const openCreateFromStart = (start: Date, startAt: string, end: Date | null = null, endAt: string | null = null) => {
    const nextState = buildSchedulingCreateModalStateFromStart({
      start,
      startAt,
      end,
      endAt,
      slots: resolveDaySlots(start),
      selectedService,
      selectedResource,
      filteredResources,
      selectedBranch,
    });
    if (nextState) {
      setModalState(nextState);
    }
  };

  const openBlockedRangeEditModal = (range: BlockedRange) => {
    setBlockedModalState({
      open: true,
      mode: 'edit',
      id: range.id,
      branchId: range.branch_id,
      resourceOptions: filteredResources.map((resource) => ({ id: resource.id, name: resource.name })),
      initial: blockedRangeDraftFromRange(range),
    });
  };

  const openBlockedRangeCreateModal = (context?: SchedulingEntryContext) => {
    if (!selectedBranchId) {
      return;
    }
    const date = context?.date ?? focusedDate;
    const base = emptyBlockedRangeDraft(date);
    const initial: typeof base = {
      ...base,
      resourceId: selectedResourceId ?? '',
      ...(context?.start ? { startTime: toLocalTimeInputValue(context.start) } : {}),
      ...(context?.end ? { endTime: toLocalTimeInputValue(context.end) } : {}),
    };
    setBlockedModalState({
      open: true,
      mode: 'create',
      branchId: selectedBranchId,
      resourceId: selectedResourceId,
      resourceOptions: filteredResources.map((resource) => ({ id: resource.id, name: resource.name })),
      initial,
    });
  };

  const openInternalEventCreateModal = (context?: SchedulingEntryContext) => {
    const date = context?.date ?? focusedDate;
    const overrides: Partial<InternalEventDraft> = { resourceId: selectedResourceId ?? '' };
    if (context?.start) overrides.startTime = toLocalTimeInputValue(context.start);
    if (context?.end) overrides.endTime = toLocalTimeInputValue(context.end);
    setInternalEventModalState({
      open: true,
      mode: 'create',
      branchId: selectedBranchId,
      resourceOptions: filteredResources.map((resource) => ({ id: resource.id, name: resource.name })),
      initial: emptyInternalEventDraft(date, overrides),
    });
  };

  // openEntryDefault — punto de entrada único: click en hueco vacío, gesto de
  // selección sobre el grid, o botón "+ Nuevo" del toolbar. Abre directamente
  // el modal de evento interno con la fecha/hora del gesto pre-cargadas, listo
  // para tipear título y guardar (UX estilo Google Calendar). Si el usuario
  // quiere otra cosa (turno o bloqueo), usa el switcher embebido en el header
  // del modal — handleSwitchType abajo lo cierra y abre el target preservando
  // el contexto.
  const openEntryDefault = (context: SchedulingEntryContext) => {
    setFocusedDate(context.date);
    lastEntryContextRef.current = context;
    openInternalEventCreateModal(context);
  };

  const handleSwitchType = (target: SchedulingEntryType) => {
    const context = lastEntryContextRef.current ?? { date: focusedDate };
    // Cerramos todos los modales de creación antes de abrir el destino, así no
    // quedan dos overlays montados a la vez.
    setInternalEventModalState({ open: false });
    setBlockedModalState({ open: false });
    setModalState({ open: false });
    if (target === 'booking') {
      const start = context.start ?? new Date(`${context.date}T09:00:00`);
      openCreateFromStart(start, start.toISOString(), context.end ?? null, context.end?.toISOString() ?? null);
      return;
    }
    if (target === 'internal_event') {
      openInternalEventCreateModal(context);
      return;
    }
    if (target === 'blocked_range') {
      openBlockedRangeCreateModal(context);
    }
  };

  const openInternalEventEditModal = (event: InternalEvent) => {
    setInternalEventModalState({
      open: true,
      mode: 'edit',
      id: event.id,
      branchId: event.branch_id ?? selectedBranchId,
      resourceOptions: filteredResources.map((resource) => ({ id: resource.id, name: resource.name })),
      initial: internalEventDraftFromEntity(event),
    });
  };

  const handleCalendarEventClick = (info: EventClickArg) => {
    const freeSlot = info.event.extendedProps.freeTimeSlot as TimeSlot | undefined;
    if (freeSlot) {
      openCreateBookingModal(freeSlot, freeSlot.resource_name);
      return;
    }
    const blockedRange = info.event.extendedProps.blockedRange as BlockedRange | undefined;
    if (blockedRange) {
      openBlockedRangeEditModal(blockedRange);
      return;
    }
    const internalEvent = info.event.extendedProps.internalEvent as InternalEvent | undefined;
    if (internalEvent) {
      openInternalEventEditModal(internalEvent);
      return;
    }
    const calendarEvent = info.event.extendedProps.calendarEvent as CalendarEvent | undefined;
    const sourceBooking = calendarEvent?.sourceBooking;
    if (!sourceBooking) {
      return;
    }
    setModalState(buildSchedulingDetailsModalState(sourceBooking, scheduleServices, filteredResources));
  };

  const handleDateClick = (info: CalendarDateClickArg) => {
    openEntryDefault({ date: info.dateStr.slice(0, 10), start: info.date });
  };

  const handleSelect = (info: CalendarDateSelectArg) => {
    calendarRef.current?.getApi().unselect();
    openEntryDefault({
      date: info.startStr.slice(0, 10),
      start: info.start,
      end: info.end,
    });
  };

  const persistBookingReschedule = async (
    info: CalendarEventDropArg | CalendarEventResizeArg,
    sourceBooking: Booking,
    confirmCopy: { title: string; description: string },
  ) => {
    const newStart = info.event.start;
    const newEnd = info.event.end;
    if (!newStart || !newEnd) {
      info.revert();
      return;
    }
    const confirmed = await confirmAction({
      title: confirmCopy.title,
      description: confirmCopy.description,
      confirmLabel: copy.rescheduleBooking,
      cancelLabel: copy.close,
    });
    if (!confirmed) {
      info.revert();
      return;
    }
    try {
      setActionError(null);
      await client.rescheduleBooking(info.event.id, {
        branch_id: selectedBranchId ?? undefined,
        resource_id: sourceBooking.resource_id,
        start_at: newStart.toISOString(),
        end_at: newEnd.toISOString(),
      });
      await invalidateSchedule();
    } catch (error) {
      info.revert();
      setActionError(error instanceof Error ? error.message : String(error));
    }
  };

  const persistBlockedRangeReschedule = async (
    info: CalendarEventDropArg | CalendarEventResizeArg,
    blockedRange: BlockedRange,
  ) => {
    const newStart = info.event.start;
    const newEnd = info.event.end;
    if (!newStart || !newEnd) {
      info.revert();
      return;
    }
    try {
      setActionError(null);
      await client.updateBlockedRange(blockedRange.id, {
        branch_id: blockedRange.branch_id,
        resource_id: blockedRange.resource_id ?? null,
        kind: blockedRange.kind,
        reason: blockedRange.reason,
        start_at: newStart.toISOString(),
        end_at: newEnd.toISOString(),
        all_day: blockedRange.all_day,
      });
      await invalidateSchedule();
    } catch (error) {
      info.revert();
      setActionError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleCalendarEventDrop = async (info: CalendarEventDropArg) => {
    const blockedRange = info.event.extendedProps.blockedRange as BlockedRange | undefined;
    if (blockedRange) {
      await persistBlockedRangeReschedule(info, blockedRange);
      return;
    }
    const calendarEvent = info.event.extendedProps.calendarEvent as CalendarEvent | undefined;
    const sourceBooking = calendarEvent?.sourceBooking;
    if (!sourceBooking) {
      info.revert();
      return;
    }
    await persistBookingReschedule(info, sourceBooking, {
      title: copy.dragRescheduleTitle,
      description: copy.dragRescheduleDescription,
    });
  };

  const handleEventResize = async (info: CalendarEventResizeArg) => {
    const blockedRange = info.event.extendedProps.blockedRange as BlockedRange | undefined;
    if (blockedRange) {
      await persistBlockedRangeReschedule(info, blockedRange);
      return;
    }
    const calendarEvent = info.event.extendedProps.calendarEvent as CalendarEvent | undefined;
    const sourceBooking = calendarEvent?.sourceBooking;
    if (!sourceBooking) {
      info.revert();
      return;
    }
    await persistBookingReschedule(info, sourceBooking, {
      title: copy.resizeBookingTitle,
      description: copy.resizeBookingDescription,
    });
  };

  const handleModalAction = async (action: SchedulingBookingAction, booking: Booking) => {
    if (action === 'cancel' || action === 'no_show') {
      const confirmed = await confirmAction({
        title: copy.destructiveTitle,
        description: action === 'cancel' ? copy.cancelActionDescription : copy.noShowActionDescription,
        confirmLabel: action === 'cancel' ? copy.cancelBooking : copy.noShowBooking,
        cancelLabel: copy.close,
        tone: 'danger',
      });
      if (!confirmed) {
        return;
      }
    }
    await bookingActionMutation.mutateAsync({ action, booking });
  };

  const updateCalendarTitle = () => {
    const api = calendarRef.current?.getApi();
    if (!api) {
      return;
    }
    setCalendarTitle(api.view.title);
  };

  const scrollCalendarToRelevantTime = () => {
    const api = calendarRef.current?.getApi();
    if (!api || !api.view.type.startsWith('timeGrid')) {
      return;
    }
    api.scrollToTime(
      resolveInitialTimeGridScrollTime({
        events: fullCalendarEventInputs,
        rangeStart: api.view.activeStart,
        rangeEnd: api.view.activeEnd,
        fallbackHour: 8,
      }),
    );
  };

  const timeGridViewport = useMemo(
    () =>
      resolveInitialTimeGridViewport({
        events: fullCalendarEventInputs,
        rangeStart: visibleRange.start,
        rangeEnd: visibleRange.end,
        fallbackHour: 8,
      }),
    [fullCalendarEventInputs, visibleRange],
  );

  const fullCalendarLocale = resolveSchedulingCopyLocale(locale) === 'es' ? 'es' : 'en';

  const fullCalendarOptions = useMemo(() => {
    if (resolveSchedulingCopyLocale(locale) !== 'es') {
      return undefined;
    }
    return {
      titleFormat: {
        day: '2-digit' as const,
        month: '2-digit' as const,
        year: 'numeric' as const,
      },
    };
  }, [locale]);

  const loading = branchesQuery.isLoading || servicesQuery.isLoading;
  const loadFailed = branchesQuery.isError || servicesQuery.isError;

  if (loading) {
    return (
      <section className={`modules-scheduling ${className}`.trim()}>
        <div className="card modules-scheduling__empty">
          <div className="spinner" />
          <p>{copy.loading}</p>
        </div>
      </section>
    );
  }

  if (loadFailed) {
    const err = branchesQuery.error ?? servicesQuery.error;
    const detail = err instanceof Error ? err.message : String(err);
    return (
      <section className={`modules-scheduling ${className}`.trim()}>
        <div className="card empty-state">
          <h3 className="text-section-title">{copy.loadErrorTitle}</h3>
          <p>{copy.loadErrorDescription}</p>
          <p className="text-secondary text-sm mono" style={{ marginTop: '0.75rem' }}>
            {detail}
          </p>
          <p style={{ marginTop: '1rem' }}>
            <button
              type="button"
              className="btn btn-secondary btn-sm"
              onClick={() => {
                void queryClient.invalidateQueries({ queryKey: schedulingKeys.branches });
                void queryClient.invalidateQueries({ queryKey: schedulingKeys.services });
              }}
            >
              {copy.retryLoad}
            </button>
          </p>
        </div>
      </section>
    );
  }

  if (!branches.length) {
    return (
      <section className={`modules-scheduling ${className}`.trim()}>
        <div className="card empty-state">
          <h3 className="text-section-title">{copy.unavailableTitle}</h3>
          <p>{copy.unavailableDescription}</p>
        </div>
      </section>
    );
  }

  return (
    <section className={`modules-scheduling ${className}`.trim()}>
      {actionError ? <div className="alert alert-error">{actionError}</div> : null}
      {!scheduleServices.length ? (
        <div className="alert alert-warning">
          <strong>{copy.noServicesTitle}</strong>
          <div>{copy.noServicesDescription}</div>
        </div>
      ) : null}

      <div className="modules-scheduling__layout">
        <div className="card modules-scheduling__calendar-card">
          {(copy.timelineTitle || copy.timelineDescription) && (
            <div className="card-header">
              <div>
                {copy.timelineTitle ? <h2>{copy.timelineTitle}</h2> : null}
                {copy.timelineDescription ? <p className="text-secondary">{copy.timelineDescription}</p> : null}
              </div>
            </div>
          )}
          <CalendarSurface
            calendarRef={calendarRef}
            view={view}
            title={calendarTitle}
            locale={fullCalendarLocale}
            calendarOptions={fullCalendarOptions ?? {}}
            loaded={!bookingsQuery.isLoading}
            onToday={() => {
              const api = calendarRef.current?.getApi();
              api?.today();
              updateCalendarTitle();
              scrollCalendarToRelevantTime();
              const tz = schedulingCalendarTimeZone.trim() || 'UTC';
              setFocusedDate(toDateInputValueFromIso(new Date().toISOString(), tz));
            }}
            onPrev={() => {
              const api = calendarRef.current?.getApi();
              api?.prev();
              updateCalendarTitle();
            }}
            onNext={() => {
              const api = calendarRef.current?.getApi();
              api?.next();
              updateCalendarTitle();
            }}
            onViewChange={(nextView: CalendarView) => {
              const api = calendarRef.current?.getApi();
              api?.changeView(nextView);
              setView(nextView);
              updateCalendarTitle();
              window.requestAnimationFrame(() => {
                scrollCalendarToRelevantTime();
              });
            }}
            toolbarTrailing={
              <div
                className="modules-scheduling__toolbar-actions"
                style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', alignItems: 'center' }}
              >
                {branches.length > 1 ? (
                  <select
                    aria-label={copy.branchLabel}
                    value={selectedBranchId ?? ''}
                    onChange={(event) => setSelectedBranchId(event.target.value || null)}
                    data-testid="scheduling-toolbar-branch"
                    style={{ minWidth: '10rem' }}
                  >
                    {branches.map((branch) => (
                      <option key={branch.id} value={branch.id}>
                        {branch.name}
                      </option>
                    ))}
                  </select>
                ) : null}
                {filteredResources.length > 1 ? (
                  <div
                    className="modules-scheduling__resource-layers"
                    role="group"
                    aria-label={copy.resourceLabel}
                    style={{ display: 'flex', gap: '0.35rem', flexWrap: 'wrap' }}
                  >
                    <button
                      type="button"
                      className={`btn-sm ${selectedResourceId === null ? 'btn-primary' : 'btn-secondary'}`}
                      onClick={() => setSelectedResourceId(null)}
                      data-testid="scheduling-resource-layer-all"
                    >
                      {copy.anyResource}
                    </button>
                    {filteredResources.map((resource) => {
                      const active = selectedResourceId === resource.id;
                      return (
                        <button
                          key={resource.id}
                          type="button"
                          className={`btn-sm ${active ? 'btn-primary' : 'btn-secondary'}`}
                          onClick={() => setSelectedResourceId(active ? null : resource.id)}
                          data-testid={`scheduling-resource-layer-${resource.id}`}
                        >
                          {resource.name}
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            }
            scrollTime={timeGridViewport.scrollTime}
            scrollTimeReset={false}
            slotMinTime={timeGridViewport.slotMinTime}
            timeZone={schedulingCalendarTimeZone.trim() || null}
            events={fullCalendarEventInputs}
            businessHours={businessHours}
            editable
            selectable={Boolean(selectedService)}
            eventDurationEditable
            selectAllow={handleSelectAllow}
            eventAllow={handleEventAllow}
            eventConstraint={businessHours.length ? 'businessHours' : undefined}
            eventContent={renderEventContent}
            dayMaxEvents
            weekends
            onEventClick={handleCalendarEventClick}
            onEventDrop={handleCalendarEventDrop}
            onEventResize={handleEventResize}
            onDateClick={handleDateClick}
            onSelect={handleSelect}
            onDatesSet={(info: { start: Date; end: Date }) => {
              setVisibleRange({ start: info.start, end: info.end });
              updateCalendarTitle();
              scrollCalendarToRelevantTime();
            }}
          />
        </div>
      </div>

      {/* Card "Filtros" eliminada: branch + resource layers ahora viven en el
          toolbarTrailing del calendario (al lado del botón "+ Nuevo"). El
          dropdown de branch solo aparece si hay >1 sucursal; los chips de
          recurso solo si hay >1 recurso. Ver deuda "branch global" en
          docs/DEUDA_TECNICA.md. */}

      <SchedulingBookingModal
        state={modalState}
        copy={copy}
        locale={locale}
        saving={createBookingMutation.isPending || bookingActionMutation.isPending}
        slotLoading={createSlotsQuery.isFetching}
        onClose={() => setModalState({ open: false })}
        onEditorChange={handleCreateEditorChange}
        onCreate={async (draft) => {
          await createBookingMutation.mutateAsync(draft);
        }}
        onAction={async (action, booking) => {
          await handleModalAction(action, booking);
        }}
        onSwitchType={handleSwitchType}
      />

      <BlockedRangeModal
        state={blockedModalState}
        copy={copy}
        locale={locale}
        saving={
          createBlockedRangeMutation.isPending ||
          updateBlockedRangeMutation.isPending ||
          deleteBlockedRangeMutation.isPending
        }
        onClose={() => setBlockedModalState({ open: false })}
        onSave={async (draft) => {
          if (blockedModalState.open && blockedModalState.mode === 'edit') {
            await updateBlockedRangeMutation.mutateAsync({ id: blockedModalState.id, draft });
          } else {
            await createBlockedRangeMutation.mutateAsync(draft);
          }
        }}
        onDelete={async (id) => {
          await deleteBlockedRangeMutation.mutateAsync(id);
        }}
        onSwitchType={handleSwitchType}
        bookingEnabled={Boolean(selectedService && selectedBranchId)}
      />

      <SchedulingInternalEventModal
        state={internalEventModalState}
        copy={copy}
        locale={locale}
        saving={
          createInternalEventMutation.isPending ||
          updateInternalEventMutation.isPending ||
          deleteInternalEventMutation.isPending
        }
        onClose={() => setInternalEventModalState({ open: false })}
        onSave={async (draft) => {
          if (internalEventModalState.open && internalEventModalState.mode === 'edit') {
            await updateInternalEventMutation.mutateAsync({ id: internalEventModalState.id, draft });
          } else {
            await createInternalEventMutation.mutateAsync(draft);
          }
        }}
        onDelete={async (id) => {
          await deleteInternalEventMutation.mutateAsync(id);
        }}
        onSwitchType={handleSwitchType}
        bookingEnabled={Boolean(selectedService && selectedBranchId)}
      />
    </section>
  );
}
