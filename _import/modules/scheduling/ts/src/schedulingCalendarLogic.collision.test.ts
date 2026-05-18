import { describe, expect, it } from 'vitest';
import type { Booking, Service } from './types';
import {
  bookingBlocksCollisions,
  bookingOccupancyOverlapsWindow,
  buildOccupancyWindowFromServiceRange,
  calendarSelectionAllowedWithBuffers,
  schedulingIntervalsOverlap,
  wallDateTimeInZoneToUtcIso,
} from './schedulingCalendarLogic';

const baseService: Service = {
  id: 'svc',
  org_id: 'o',
  code: 'c',
  name: 'N',
  description: '',
  fulfillment_mode: 'schedule',
  default_duration_minutes: 30,
  buffer_before_minutes: 0,
  buffer_after_minutes: 15,
  slot_granularity_minutes: 15,
  max_concurrent_bookings: 1,
  min_cancel_notice_minutes: 0,
  allow_waitlist: false,
  active: true,
  resource_ids: [],
  metadata: {},
  created_at: '',
  updated_at: '',
};

describe('schedulingIntervalsOverlap', () => {
  it('detecta solape y permite límite contiguo sin solape', () => {
    const a0 = new Date('2026-04-09T15:30:00Z');
    const a1 = new Date('2026-04-09T16:00:00Z');
    const b0 = new Date('2026-04-09T16:00:00Z');
    const b1 = new Date('2026-04-09T16:30:00Z');
    expect(schedulingIntervalsOverlap(a0, a1, b0, b1)).toBe(false);
    expect(schedulingIntervalsOverlap(a0, a1, new Date('2026-04-09T15:45:00Z'), b1)).toBe(true);
  });
});

describe('buildOccupancyWindowFromServiceRange', () => {
  it('extiende el fin con buffer_after', () => {
    const start = new Date('2026-04-09T16:00:00Z');
    const end = new Date('2026-04-09T16:30:00Z');
    const { occupiesFrom, occupiesUntil } = buildOccupancyWindowFromServiceRange(start, end, baseService);
    expect(occupiesFrom.toISOString()).toBe('2026-04-09T16:00:00.000Z');
    expect(occupiesUntil.toISOString()).toBe('2026-04-09T16:45:00.000Z');
  });
});

describe('calendarSelectionAllowedWithBuffers', () => {
  const booking: Booking = {
    id: 'b1',
    org_id: 'o',
    branch_id: 'br',
    service_id: baseService.id,
    resource_id: 'res1',
    reference: 'r',
    customer_name: 'A',
    customer_phone: '1',
    status: 'confirmed',
    source: 'admin',
    start_at: '2026-04-09T15:30:00Z',
    end_at: '2026-04-09T16:00:00Z',
    occupies_from: '2026-04-09T15:30:00Z',
    occupies_until: '2026-04-09T16:15:00Z',
    notes: '',
    created_by: 'x',
    created_at: '',
    updated_at: '',
  };

  it('rechaza selección que cae en ventana de buffer del turno previo', () => {
    const ok = calendarSelectionAllowedWithBuffers({
      start: new Date('2026-04-09T16:00:00Z'),
      end: new Date('2026-04-09T16:30:00Z'),
      service: baseService,
      resourceIds: ['res1'],
      bookings: [booking],
      blockedRanges: [],
    });
    expect(ok).toBe(false);
  });

  it('acepta selección tras el buffer del turno previo', () => {
    const ok = calendarSelectionAllowedWithBuffers({
      start: new Date('2026-04-09T16:15:00Z'),
      end: new Date('2026-04-09T16:45:00Z'),
      service: baseService,
      resourceIds: ['res1'],
      bookings: [booking],
      blockedRanges: [],
    });
    expect(ok).toBe(true);
  });
});

describe('bookingOccupancyOverlapsWindow', () => {
  it('usa occupies del booking', () => {
    const booking: Booking = {
      id: 'b1',
      org_id: 'o',
      branch_id: 'br',
      service_id: 's',
      resource_id: 'res1',
      reference: 'r',
      customer_name: 'A',
      customer_phone: '1',
      status: 'confirmed',
      source: 'admin',
      start_at: '2026-04-09T15:30:00Z',
      end_at: '2026-04-09T16:00:00Z',
      occupies_from: '2026-04-09T15:30:00Z',
      occupies_until: '2026-04-09T16:15:00Z',
      notes: '',
      created_by: 'x',
      created_at: '',
      updated_at: '',
    };
    expect(
      bookingOccupancyOverlapsWindow(
        booking,
        new Date('2026-04-09T16:00:00Z'),
        new Date('2026-04-09T16:30:00Z'),
      ),
    ).toBe(true);
  });
});

describe('bookingBlocksCollisions', () => {
  it('excluye hold vencido', () => {
    const past = new Date(Date.now() - 60_000).toISOString();
    const booking: Booking = {
      id: 'h',
      org_id: 'o',
      branch_id: 'br',
      service_id: 's',
      resource_id: 'r',
      reference: 'r',
      customer_name: 'A',
      customer_phone: '1',
      status: 'hold',
      source: 'admin',
      start_at: '2099-01-01T10:00:00Z',
      end_at: '2099-01-01T10:30:00Z',
      occupies_from: '2099-01-01T10:00:00Z',
      occupies_until: '2099-01-01T10:30:00Z',
      hold_expires_at: past,
      notes: '',
      created_by: 'x',
      created_at: '',
      updated_at: '',
    };
    expect(bookingBlocksCollisions(booking)).toBe(false);
  });
});

describe('wallDateTimeInZoneToUtcIso', () => {
  it('interpreta hora de pared en la zona del recurso (no la del navegador)', () => {
    const iso = wallDateTimeInZoneToUtcIso('2026-04-09', '10:30', 'America/Argentina/Tucuman');
    expect(iso).toMatch(/^2026-04-09T13:30:00\.000Z$/);
  });
});
