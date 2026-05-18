// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { SchedulingClient } from './client';
import { QueueOperatorBoard } from './QueueOperatorBoard';
import type { Branch, DayAgendaItem, Queue } from './types';

const confirmActionMock = vi.hoisted(() => vi.fn(async () => true));

vi.mock('@devpablocristo/core-browser', () => ({
  confirmAction: confirmActionMock,
}));

function createClient(overrides?: Partial<Record<keyof SchedulingClient, unknown>>): SchedulingClient {
  const branches: Branch[] = [
    {
      id: 'branch-1',
      org_id: 'org-1',
      code: 'HQ',
      name: 'Casa Central',
      timezone: 'America/Argentina/Tucuman',
      address: 'Main street 123',
      active: true,
      created_at: '2099-04-01T00:00:00Z',
      updated_at: '2099-04-01T00:00:00Z',
    },
  ];

  const queues: Queue[] = [
    {
      id: 'queue-1',
      org_id: 'org-1',
      branch_id: 'branch-1',
      service_id: null,
      code: 'BOX-1',
      name: 'Mesa de entrada',
      status: 'active',
      strategy: 'fifo',
      ticket_prefix: 'A',
      avg_service_seconds: 180,
      allow_remote_join: true,
      created_at: '2099-04-01T00:00:00Z',
      updated_at: '2099-04-01T00:00:00Z',
    },
  ];

  const dayAgenda: DayAgendaItem[] = [
    {
      type: 'queue_ticket',
      id: 'ticket-1',
      branch_id: 'branch-1',
      status: 'waiting',
      label: 'A-17',
      metadata: {
        queue_id: 'queue-1',
        number: 17,
      },
    },
  ];

  return {
    listBranches: vi.fn(async () => branches),
    listQueues: vi.fn(async () => queues),
    getDayAgenda: vi.fn(async () => dayAgenda),
    createQueueTicket: vi.fn(async () => ({
      id: 'ticket-2',
      org_id: 'org-1',
      queue_id: 'queue-1',
      branch_id: 'branch-1',
      number: 18,
      display_code: 'A-18',
      customer_name: 'Ada Lovelace',
      customer_phone: '+54 381 555 0404',
      status: 'waiting',
      source: 'reception',
      priority: 2,
      notes: '',
      requested_at: '2099-04-05T10:00:00Z',
      updated_at: '2099-04-05T10:00:00Z',
    })),
    closeQueue: vi.fn(async () => queues[0]),
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
    getQueuePosition: vi.fn(),
    pauseQueue: vi.fn(),
    reopenQueue: vi.fn(),
    callNext: vi.fn(),
    serveTicket: vi.fn(),
    completeTicket: vi.fn(),
    markTicketNoShow: vi.fn(),
    cancelTicket: vi.fn(),
    returnTicketToWaiting: vi.fn(),
    getDashboard: vi.fn(),
    ...(overrides ?? {}),
  } as SchedulingClient;
}

function renderBoard(client: SchedulingClient, locale = 'es') {
  cleanup();
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <QueueOperatorBoard client={client} locale={locale} initialBranchId="branch-1" initialDate="2099-04-05" />
    </QueryClientProvider>,
  );
}

describe('QueueOperatorBoard', () => {
  it('issues a ticket with the queue operator payload', async () => {
    const client = createClient();

    renderBoard(client);

    fireEvent.change(await screen.findByLabelText('Cliente'), { target: { value: 'Ada Lovelace' } });
    fireEvent.change(screen.getByLabelText('Teléfono'), { target: { value: '+54 381 555 0404' } });
    fireEvent.change(screen.getByLabelText('Prioridad'), { target: { value: '2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Emitir ticket' }));

    await waitFor(() => {
      expect(client.createQueueTicket).toHaveBeenCalledWith('queue-1', {
        customer_name: 'Ada Lovelace',
        customer_phone: '+54 381 555 0404',
        customer_email: undefined,
        priority: 2,
      });
    });
  });

  it('confirms and closes an active queue', async () => {
    const client = createClient();

    renderBoard(client);

    fireEvent.click(await screen.findByRole('button', { name: 'Cerrar cola' }));

    await waitFor(() => {
      expect(confirmActionMock).toHaveBeenCalled();
      expect(client.closeQueue).toHaveBeenCalledWith('queue-1');
    });
  });

  it('keeps spanish copy when using a regional spanish locale', async () => {
    const client = createClient();

    renderBoard(client, 'es-AR');

    expect(await screen.findByRole('button', { name: 'Cerrar cola' })).toBeTruthy();
  });
});
