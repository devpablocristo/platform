// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { SchedulingClient } from './client';
import { SchedulingDaySummary } from './SchedulingDaySummary';
import type { DashboardStats, DayAgendaItem } from './types';

function createClient(overrides?: Partial<Record<keyof SchedulingClient, unknown>>): SchedulingClient {
  const dashboard: DashboardStats = {
    date: '2099-12-01',
    timezone: 'America/Argentina/Tucuman',
    bookings_today: 12,
    confirmed_bookings_today: 7,
    active_queues: 2,
    waiting_tickets: 5,
    tickets_in_service: 1,
  };
  const dayAgenda: DayAgendaItem[] = [
    {
      type: 'booking',
      id: 'booking-1',
      branch_id: 'branch-1',
      service_id: 'service-1',
      start_at: '2099-12-01T10:00:00Z',
      end_at: '2099-12-01T10:30:00Z',
      status: 'confirmed',
      label: 'Consulta Ada',
    },
    {
      type: 'queue_ticket',
      id: 'ticket-1',
      branch_id: 'branch-1',
      status: 'waiting',
      label: 'A-17',
    },
  ];

  return {
    listBranches: vi.fn(),
    listServices: vi.fn(),
    listResources: vi.fn(),
    listSlots: vi.fn(),
    listBookings: vi.fn(),
    getBooking: vi.fn(),
    createBooking: vi.fn(),
    confirmBooking: vi.fn(),
    cancelBooking: vi.fn(),
    checkInBooking: vi.fn(),
    startService: vi.fn(),
    completeBooking: vi.fn(),
    markBookingNoShow: vi.fn(),
    rescheduleBooking: vi.fn(),
    listWaitlist: vi.fn(),
    listQueues: vi.fn(),
    createQueueTicket: vi.fn(),
    getQueuePosition: vi.fn(),
    pauseQueue: vi.fn(),
    reopenQueue: vi.fn(),
    closeQueue: vi.fn(),
    callNext: vi.fn(),
    serveTicket: vi.fn(),
    completeTicket: vi.fn(),
    markTicketNoShow: vi.fn(),
    cancelTicket: vi.fn(),
    returnTicketToWaiting: vi.fn(),
    getDashboard: vi.fn(async () => dashboard),
    getDayAgenda: vi.fn(async () => dayAgenda),
    ...(overrides ?? {}),
  } as SchedulingClient;
}

function renderSummary(client: SchedulingClient) {
  cleanup();
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <SchedulingDaySummary client={client} locale="es" initialDate="2099-12-01" />
    </QueryClientProvider>,
  );
}

function renderSummaryWithLocale(client: SchedulingClient, locale: string) {
  cleanup();
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <SchedulingDaySummary client={client} locale={locale} initialDate="2099-12-01" />
    </QueryClientProvider>,
  );
}

describe('SchedulingDaySummary', () => {
  it('renders stats, next booking and queue flow from the shared client', async () => {
    const client = createClient();

    renderSummary(client);

    await waitFor(() => {
      expect(screen.getByText('Agenda de hoy')).toBeTruthy();
      expect(screen.getByText('12')).toBeTruthy();
      expect(screen.getByText('7')).toBeTruthy();
      expect(screen.getByText('2')).toBeTruthy();
      expect(screen.getByText('5')).toBeTruthy();
    });

    expect(screen.getByText('Consulta Ada')).toBeTruthy();
    expect(screen.getByText('Confirmada')).toBeTruthy();
    expect(screen.getByText('A-17')).toBeTruthy();
    expect(screen.getAllByText('Esperando').length).toBeGreaterThan(0);
  });

  it('supports regional locales while keeping the correct copy preset', async () => {
    const client = createClient();

    renderSummaryWithLocale(client, 'es-AR');

    await waitFor(() => {
      expect(screen.getByText('Agenda de hoy')).toBeTruthy();
      expect(screen.getByText(/\d{1,2}\/\d{1,2}\/\d{4}/)).toBeTruthy();
    });
  });
});
