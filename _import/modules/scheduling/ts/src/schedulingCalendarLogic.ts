import type { EventInput } from '@fullcalendar/core';
import type {
  SchedulingBookingCreateEditor,
  SchedulingBookingCreateResourceOption,
  SchedulingBookingModalState,
  SchedulingBookingRecurrenceDraft,
} from './SchedulingBookingModal';
import type {
  AvailabilityRule,
  BlockedRange,
  Booking,
  Branch,
  CalendarEvent,
  Resource,
  Service,
  TimeSlot,
} from './types';

export type SchedulingBusinessHours = {
  daysOfWeek: number[];
  startTime: string;
  endTime: string;
};

type BuildCreateModalStateParams = {
  start: Date;
  startAt: string;
  end?: Date | null;
  endAt?: string | null;
  slots: readonly TimeSlot[];
  selectedService: Service | null;
  selectedResource: Resource | null;
  filteredResources: readonly Resource[];
  selectedBranch: Branch | null;
};

export function slotDurationMinutes(slot: TimeSlot): number {
  const start = new Date(slot.start_at).getTime();
  const end = new Date(slot.end_at).getTime();
  return Math.max(0, Math.round((end - start) / 60000));
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

function readFormatterPart(parts: Intl.DateTimeFormatPart[], type: Intl.DateTimeFormatPartTypes): string {
  return parts.find((p) => p.type === type)?.value ?? '';
}

/** Fecha calendario (YYYY-MM-DD) del instante UTC en la zona IANA indicada. */
export function utcInstantToWallDate(isoUtc: string, timeZone: string): string {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
  const parts = formatter.formatToParts(new Date(isoUtc));
  const y = readFormatterPart(parts, 'year');
  const mo = readFormatterPart(parts, 'month');
  const d = readFormatterPart(parts, 'day');
  return `${y}-${mo}-${d}`;
}

/** Hora HH:mm del instante UTC en la zona IANA indicada (24 h). */
export function utcInstantToWallClock(isoUtc: string, timeZone: string): string {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  });
  const parts = formatter.formatToParts(new Date(isoUtc));
  const h = readFormatterPart(parts, 'hour');
  const mi = readFormatterPart(parts, 'minute');
  return `${h.padStart(2, '0')}:${mi.padStart(2, '0')}`;
}

/**
 * Interpreta fecha + hora como reloj de pared en `timeZone` (sucursal/recurso) y devuelve ISO UTC.
 * Evita `new Date("YYYY-MM-DDTHH:mm:ss")`, que usa la zona del navegador y rompe validación en el servidor.
 */
export function wallDateTimeInZoneToUtcIso(ymd: string, clockHm: string, timeZone: string): string {
  const clock = normalizeClock(clockHm);
  const [y, mo, d] = ymd.split('-').map((piece) => Number(piece));
  const [H, Mi] = clock.split(':').map((piece) => Number(piece));
  if (
    !Number.isFinite(y) ||
    !Number.isFinite(mo) ||
    !Number.isFinite(d) ||
    !Number.isFinite(H) ||
    !Number.isFinite(Mi)
  ) {
    return new Date(`${ymd}T${clock}:00`).toISOString();
  }

  const center = Date.UTC(y, mo - 1, d, 12, 0, 0);
  const windowMs = 52 * 60 * 60 * 1000;
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    hourCycle: 'h23',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });

  for (let t = center - windowMs; t <= center + windowMs; t += 60_000) {
    const parts = formatter.formatToParts(new Date(t));
    const py = Number(readFormatterPart(parts, 'year'));
    const pm = Number(readFormatterPart(parts, 'month'));
    const pd = Number(readFormatterPart(parts, 'day'));
    const ph = Number(readFormatterPart(parts, 'hour'));
    const pmi = Number(readFormatterPart(parts, 'minute'));
    if (py === y && pm === mo && pd === d && ph === H && pmi === Mi) {
      return new Date(t).toISOString();
    }
  }

  return new Date(`${ymd}T${clock}:00`).toISOString();
}

export function toDateInputValue(value: string, timeZone?: string | null): string {
  const tz = timeZone?.trim();
  if (tz) {
    if (Number.isNaN(new Date(value).getTime())) {
      return value.slice(0, 10);
    }
    return utcInstantToWallDate(value, tz);
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value.slice(0, 10);
  }
  return `${parsed.getFullYear()}-${pad(parsed.getMonth() + 1)}-${pad(parsed.getDate())}`;
}

export function toTimeInputValue(value: string, timeZone?: string | null): string {
  const tz = timeZone?.trim();
  if (tz) {
    if (Number.isNaN(new Date(value).getTime())) {
      return value;
    }
    return utcInstantToWallClock(value, tz);
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return `${pad(parsed.getHours())}:${pad(parsed.getMinutes())}`;
}

/** Zona IANA efectiva: recurso, si no sucursal (evita "" y undefined que caían en UTC/local mal). */
export function effectiveSchedulingTimeZone(
  resource: Resource | null | undefined,
  branch: Branch | null,
): string {
  const r = resource?.timezone?.trim();
  if (r) {
    return r;
  }
  const b = branch?.timezone?.trim();
  if (b) {
    return b;
  }
  return 'UTC';
}

export function buildCreateEditorFromSlot(slot: TimeSlot, branchTimezone?: string | null): SchedulingBookingCreateEditor {
  const tz = slot.timezone?.trim() || branchTimezone?.trim() || null;
  return {
    date: toDateInputValue(slot.start_at, tz),
    startTime: toTimeInputValue(slot.start_at, tz),
    endTime: toTimeInputValue(slot.end_at, tz),
    resourceId: slot.resource_id,
  };
}

export function buildDefaultRecurrenceDraft(slot: TimeSlot): SchedulingBookingRecurrenceDraft {
  const weekday = new Date(slot.start_at).getUTCDay();
  return {
    mode: 'none',
    frequency: 'weekly',
    interval: '1',
    count: '8',
    byWeekday: [weekday],
  };
}

export function resolveBookingDisplayTitle(booking: Booking): string {
  const metadataTitle = booking.metadata?.title;
  return typeof metadataTitle === 'string' && metadataTitle.trim() ? metadataTitle.trim() : booking.customer_name;
}

export function buildCreateResourceOptions(resources: readonly Resource[], slot: TimeSlot): SchedulingBookingCreateResourceOption[] {
  const merged = resources.some((resource) => resource.id === slot.resource_id)
    ? resources
    : [
        ...resources,
        {
          id: slot.resource_id,
          org_id: '',
          branch_id: '',
          code: slot.resource_id,
          name: slot.resource_name,
          kind: 'generic',
          capacity: 1,
          timezone: slot.timezone,
          active: true,
          created_at: '',
          updated_at: '',
        },
      ];
  return merged.map((resource) => ({
    id: resource.id,
    name: resource.name,
    timezone: resource.timezone,
  }));
}

function slotIdentity(slot: TimeSlot): string {
  return `${slot.resource_id}:${slot.start_at}:${slot.end_at}`;
}

export function buildSlotIdentity(resourceId: string, startAt: string, endAt: string): string {
  return `${resourceId}:${startAt}:${endAt}`;
}

type ClockWindow = {
  start: string;
  end: string;
};

function normalizeClock(value: string): string {
  return value.slice(0, 5);
}

function toClockMinutes(value: string): number {
  const [hours, minutes] = normalizeClock(value).split(':').map((piece) => Number(piece));
  return hours * 60 + minutes;
}

function intersectClockWindows(left: readonly ClockWindow[], right: readonly ClockWindow[]): ClockWindow[] {
  if (!left.length || !right.length) {
    return [];
  }
  const intersections: ClockWindow[] = [];
  for (const leftWindow of left) {
    const leftStart = toClockMinutes(leftWindow.start);
    const leftEnd = toClockMinutes(leftWindow.end);
    for (const rightWindow of right) {
      const start = Math.max(leftStart, toClockMinutes(rightWindow.start));
      const end = Math.min(leftEnd, toClockMinutes(rightWindow.end));
      if (end <= start) {
        continue;
      }
      intersections.push({
        start: `${String(Math.floor(start / 60)).padStart(2, '0')}:${String(start % 60).padStart(2, '0')}`,
        end: `${String(Math.floor(end / 60)).padStart(2, '0')}:${String(end % 60).padStart(2, '0')}`,
      });
    }
  }
  return intersections;
}

export function buildSchedulingBusinessHours(
  rules: readonly AvailabilityRule[],
  selectedResourceId: string | null,
): SchedulingBusinessHours[] {
  const activeRules = rules.filter((rule) => rule.active);
  const businessHours: SchedulingBusinessHours[] = [];

  for (let weekday = 0; weekday <= 6; weekday += 1) {
    const branchWindows = activeRules
      .filter((rule) => rule.weekday === weekday && rule.kind === 'branch')
      .map((rule) => ({ start: normalizeClock(rule.start_time), end: normalizeClock(rule.end_time) }));

    if (!branchWindows.length) {
      continue;
    }

    let activeWindows = branchWindows;
    if (selectedResourceId) {
      const resourceWindows = activeRules
        .filter((rule) => rule.weekday === weekday && rule.kind === 'resource' && rule.resource_id === selectedResourceId)
        .map((rule) => ({ start: normalizeClock(rule.start_time), end: normalizeClock(rule.end_time) }));
      if (resourceWindows.length) {
        activeWindows = intersectClockWindows(branchWindows, resourceWindows);
      }
    }

    for (const window of activeWindows) {
      businessHours.push({
        daysOfWeek: [weekday],
        startTime: window.start,
        endTime: window.end,
      });
    }
  }

  return businessHours;
}

function buildCreateSlotOptions(slot: TimeSlot, slots: readonly TimeSlot[]): TimeSlot[] {
  const merged = [slot, ...slots];
  const unique = new Map<string, TimeSlot>();
  for (const item of merged) {
    unique.set(slotIdentity(item), item);
  }
  return Array.from(unique.values()).sort((left, right) => {
    const startCompare = left.start_at.localeCompare(right.start_at);
    if (startCompare !== 0) {
      return startCompare;
    }
    return left.resource_name.localeCompare(right.resource_name);
  });
}

export function buildSchedulingCalendarEvents(
  bookings: readonly Booking[],
  scheduleServices: readonly Service[],
  filteredResources: readonly Resource[],
  eventColor: (status: Booking['status']) => string,
): CalendarEvent[] {
  return bookings.map((booking) => {
    const serviceName = scheduleServices.find((service) => service.id === booking.service_id)?.name;
    const resourceName = filteredResources.find((resource) => resource.id === booking.resource_id)?.name;
    return {
      id: booking.id,
      kind: 'booking',
      sourceId: booking.id,
      sourceType: 'booking',
      title: resolveBookingDisplayTitle(booking),
      start_at: booking.start_at,
      end_at: booking.end_at,
      color: eventColor(booking.status),
      status: booking.status,
      serviceName,
      resourceName,
      sourceBooking: booking,
    };
  });
}

export function buildFullCalendarEventInputs(calendarEvents: readonly CalendarEvent[]): EventInput[] {
  return calendarEvents.map((calendarEvent) => ({
    id: calendarEvent.id,
    title: calendarEvent.title,
    start: calendarEvent.start_at,
    end: calendarEvent.end_at,
    color: calendarEvent.color,
    extendedProps: {
      calendarEvent,
    },
  }));
}

export function buildSchedulingDetailsModalState(
  booking: Booking,
  scheduleServices: readonly Service[],
  filteredResources: readonly Resource[],
): SchedulingBookingModalState {
  return {
    open: true,
    mode: 'details',
    booking,
    service: scheduleServices.find((service) => service.id === booking.service_id),
    resourceName: filteredResources.find((resource) => resource.id === booking.resource_id)?.name,
  };
}

export function buildSchedulingCreateModalStateFromSlot(
  slot: TimeSlot,
  selectedService: Service | null,
  slots: readonly TimeSlot[],
  resources: readonly Resource[],
  resourceName: string | undefined,
  branchTimezone: string | null | undefined,
): SchedulingBookingModalState {
  return {
    open: true,
    mode: 'create',
    slot,
    slotOptions: buildCreateSlotOptions(slot, slots),
    resourceOptions: buildCreateResourceOptions(resources, slot),
    editor: buildCreateEditorFromSlot(slot, branchTimezone),
    validationMessage: null,
    service: selectedService ?? undefined,
    resourceName: resourceName ?? slot.resource_name,
    draft: {
      title: '',
      customerName: '',
      customerPhone: '',
      customerEmail: '',
      notes: '',
      recurrence: buildDefaultRecurrenceDraft(slot),
    },
  };
}

export function buildSchedulingCreateModalStateFromStart({
  start,
  startAt,
  end,
  endAt,
  slots,
  selectedService,
  selectedResource,
  filteredResources,
  selectedBranch,
}: BuildCreateModalStateParams): SchedulingBookingModalState | null {
  if (!selectedService) {
    return null;
  }

  const matchingSlot =
    slots.find((slot) => slot.start_at === startAt && (!endAt || slot.end_at === endAt)) ??
    slots.find((slot) => slot.start_at === startAt);
  if (matchingSlot) {
    return buildSchedulingCreateModalStateFromSlot(
      matchingSlot,
      selectedService,
      slots,
      filteredResources,
      matchingSlot.resource_name,
      selectedBranch?.timezone,
    );
  }

  const fallbackResource = selectedResource ?? filteredResources[0];
  if (!fallbackResource) {
    return null;
  }

  const provisionalEnd =
    end && end.getTime() > start.getTime()
      ? end
      : new Date(start.getTime() + selectedService.default_duration_minutes * 60_000);

  return buildSchedulingCreateModalStateFromSlot(
    {
      resource_id: fallbackResource.id,
      resource_name: fallbackResource.name,
      start_at: start.toISOString(),
      end_at: provisionalEnd.toISOString(),
      occupies_from: start.toISOString(),
      occupies_until: provisionalEnd.toISOString(),
      timezone: effectiveSchedulingTimeZone(fallbackResource, selectedBranch),
      remaining: 1,
      conflict_count: 0,
      granularity_minutes: selectedService.slot_granularity_minutes,
    },
    selectedService,
    slots,
    filteredResources,
    fallbackResource.name,
    selectedBranch?.timezone,
  );
}

/** Slot sintético desde el editor cuando no hay coincidencia en la lista de slots del API (rango libre). */
export function buildSyntheticTimeSlotFromEditor(
  editor: SchedulingBookingCreateEditor,
  service: Service,
  resource: Resource | null | undefined,
  branch: Branch | null,
): TimeSlot | null {
  if (!resource) {
    return null;
  }
  const timeZone = effectiveSchedulingTimeZone(resource, branch);
  const startAt = wallDateTimeInZoneToUtcIso(editor.date, normalizeClock(editor.startTime), timeZone);
  const endAt = wallDateTimeInZoneToUtcIso(editor.date, normalizeClock(editor.endTime), timeZone);
  const startMs = Date.parse(startAt);
  const endMs = Date.parse(endAt);
  if (!Number.isFinite(startMs) || !Number.isFinite(endMs) || endMs <= startMs) {
    return null;
  }
  return {
    resource_id: resource.id,
    resource_name: resource.name,
    start_at: startAt,
    end_at: endAt,
    occupies_from: startAt,
    occupies_until: endAt,
    timezone: timeZone,
    remaining: 1,
    conflict_count: 0,
    granularity_minutes: service.slot_granularity_minutes,
  };
}

/** Paridad con `bookingStatusesBlocking` en el repositorio Go. */
export const SCHEDULING_BLOCKING_BOOKING_STATUSES: ReadonlySet<Booking['status']> = new Set([
  'hold',
  'pending_confirmation',
  'confirmed',
  'checked_in',
  'in_service',
]);

export function bookingBlocksCollisions(booking: Booking, nowMs: number = Date.now()): boolean {
  if (!SCHEDULING_BLOCKING_BOOKING_STATUSES.has(booking.status)) {
    return false;
  }
  if (booking.status === 'hold' && booking.hold_expires_at) {
    const exp = Date.parse(booking.hold_expires_at);
    if (Number.isFinite(exp) && exp < nowMs) {
      return false;
    }
  }
  return true;
}

export function buildOccupancyWindowFromServiceRange(
  start: Date,
  end: Date,
  service: Pick<Service, 'buffer_before_minutes' | 'buffer_after_minutes'>,
): { occupiesFrom: Date; occupiesUntil: Date } {
  const beforeMs = Math.max(0, service.buffer_before_minutes ?? 0) * 60_000;
  const afterMs = Math.max(0, service.buffer_after_minutes ?? 0) * 60_000;
  return {
    occupiesFrom: new Date(start.getTime() - beforeMs),
    occupiesUntil: new Date(end.getTime() + afterMs),
  };
}

/** Solape de intervalos [aStart,aEnd) vs [bStart,bEnd) alineado al criterio SQL del backend. */
export function schedulingIntervalsOverlap(aStart: Date, aEnd: Date, bStart: Date, bEnd: Date): boolean {
  return aStart.getTime() < bEnd.getTime() && aEnd.getTime() > bStart.getTime();
}

export function bookingOccupancyOverlapsWindow(
  booking: Booking,
  occupiesFrom: Date,
  occupiesUntil: Date,
): boolean {
  const bStart = new Date(booking.occupies_from);
  const bEnd = new Date(booking.occupies_until);
  if (!Number.isFinite(bStart.getTime()) || !Number.isFinite(bEnd.getTime())) {
    return false;
  }
  return schedulingIntervalsOverlap(occupiesFrom, occupiesUntil, bStart, bEnd);
}

export function blockedRangeOverlapsWindow(
  range: BlockedRange,
  occupiesFrom: Date,
  occupiesUntil: Date,
  resourceId: string,
): boolean {
  const rStart = new Date(range.start_at);
  const rEnd = new Date(range.end_at);
  if (!Number.isFinite(rStart.getTime()) || !Number.isFinite(rEnd.getTime())) {
    return false;
  }
  const appliesToResource = !range.resource_id || range.resource_id === resourceId;
  if (!appliesToResource) {
    return false;
  }
  return schedulingIntervalsOverlap(occupiesFrom, occupiesUntil, rStart, rEnd);
}

/**
 * Comprueba si algún recurso candidato admite la selección sin solapar ocupaciones existentes
 * (incl. buffers del servicio) ni bloqueos. Evita abrir el modal con rangos que el backend
 * rechazará con 409.
 */
export function calendarSelectionAllowedWithBuffers(params: {
  start: Date;
  end: Date;
  /** Obligatorio salvo cuando `occupancyIsExplicit` es true (solo se usa para buffers). */
  service: Service | null;
  resourceIds: readonly string[];
  bookings: readonly Booking[];
  blockedRanges: readonly BlockedRange[];
  /** Ventana de ocupación = [start,end] sin ampliar por buffers (p. ej. arrastre de bloqueo). */
  occupancyIsExplicit?: boolean;
  excludeBookingId?: string | null;
  excludeBlockedRangeId?: string | null;
}): boolean {
  const {
    start,
    end,
    service,
    resourceIds,
    bookings,
    blockedRanges,
    occupancyIsExplicit = false,
    excludeBookingId,
    excludeBlockedRangeId,
  } = params;
  if (!resourceIds.length) {
    return false;
  }
  if (!occupancyIsExplicit && !service) {
    return false;
  }
  const { occupiesFrom, occupiesUntil } = occupancyIsExplicit
    ? { occupiesFrom: start, occupiesUntil: end }
    : buildOccupancyWindowFromServiceRange(start, end, service!);
  if (occupiesUntil.getTime() <= occupiesFrom.getTime()) {
    return false;
  }

  for (const resourceId of resourceIds) {
    let collides = false;
    for (const booking of bookings) {
      if (excludeBookingId && booking.id === excludeBookingId) {
        continue;
      }
      if (!bookingBlocksCollisions(booking)) {
        continue;
      }
      if (booking.resource_id !== resourceId) {
        continue;
      }
      if (bookingOccupancyOverlapsWindow(booking, occupiesFrom, occupiesUntil)) {
        collides = true;
        break;
      }
    }
    if (collides) {
      continue;
    }
    for (const range of blockedRanges) {
      if (excludeBlockedRangeId && range.id === excludeBlockedRangeId) {
        continue;
      }
      if (blockedRangeOverlapsWindow(range, occupiesFrom, occupiesUntil, resourceId)) {
        collides = true;
        break;
      }
    }
    if (!collides) {
      return true;
    }
  }
  return false;
}
