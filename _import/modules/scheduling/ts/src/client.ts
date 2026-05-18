import type {
  AvailabilityRule,
  BlockedRange,
  BlockedRangePayload,
  CreateInternalEventPayload,
  InternalEvent,
  ListInternalEventsQuery,
  UpdateInternalEventPayload,
  Booking,
  Branch,
  CreateBookingPayload,
  DashboardStats,
  DayAgendaItem,
  ListBlockedRangesQuery,
  ListBookingsFilter,
  PublicAvailabilityQuery,
  PublicAvailabilitySlot,
  PublicBusinessInfo,
  PublicBookPayload,
  PublicBooking,
  PublicMyBookingsQuery,
  PublicQueuePosition,
  PublicQueueSummary,
  PublicQueueTicket,
  PublicQueueTicketPayload,
  PublicService,
  PublicWaitlistEntry,
  PublicWaitlistPayload,
  Queue,
  QueuePosition,
  QueueTicket,
  RescheduleBookingPayload,
  Resource,
  Service,
  SlotQuery,
  TimeSlot,
  WaitlistEntry,
  SchedulingTransport,
} from './types';

export type SchedulingClient = ReturnType<typeof createSchedulingClient>;
export type PublicSchedulingClient = ReturnType<typeof createPublicSchedulingClient>;

function appendParam(params: URLSearchParams, key: string, value: string | number | boolean | null | undefined) {
  if (value === undefined || value === null || value === '') {
    return;
  }
  params.set(key, String(value));
}

function queryString(values: Record<string, string | number | boolean | null | undefined>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    appendParam(params, key, value);
  }
  const serialized = params.toString();
  return serialized ? `?${serialized}` : '';
}

export function createSchedulingClient(request: SchedulingTransport) {
  return {
    listBranches() {
      return request<{ items: Branch[] }>('/v1/scheduling/branches').then((response) => response.items ?? []);
    },
    listServices() {
      return request<{ items: Service[] }>('/v1/scheduling/services').then((response) => response.items ?? []);
    },
    listResources(branchId?: string | null) {
      return request<{ items: Resource[] }>(`/v1/scheduling/resources${queryString({ branch_id: branchId })}`).then(
        (response) => response.items ?? [],
      );
    },
    listAvailabilityRules(branchId?: string | null, resourceId?: string | null) {
      return request<{ items: AvailabilityRule[] }>(
        `/v1/scheduling/availability-rules${queryString({ branch_id: branchId, resource_id: resourceId })}`,
      ).then((response) => response.items ?? []);
    },
    listBlockedRanges(query: ListBlockedRangesQuery = {}) {
      return request<{ items: BlockedRange[] }>(
        `/v1/scheduling/blocked-ranges${queryString({
          branch_id: query.branchId,
          resource_id: query.resourceId,
          date: query.date,
        })}`,
      ).then((response) => response.items ?? []);
    },
    createBlockedRange(payload: BlockedRangePayload) {
      return request<BlockedRange>('/v1/scheduling/blocked-ranges', { method: 'POST', body: payload });
    },
    updateBlockedRange(id: string, payload: BlockedRangePayload) {
      return request<BlockedRange>(`/v1/scheduling/blocked-ranges/${id}`, { method: 'PATCH', body: payload });
    },
    deleteBlockedRange(id: string) {
      return request<void>(`/v1/scheduling/blocked-ranges/${id}`, { method: 'DELETE' });
    },
    listInternalEvents(query: ListInternalEventsQuery = {}) {
      return request<{ items: InternalEvent[] }>(
        `/v1/scheduling/calendar-events${queryString({
          branch_id: query.branch_id,
          resource_id: query.resource_id,
          from: query.from,
          to: query.to,
          status: query.status,
        })}`,
      ).then((response) => response.items ?? []);
    },
    getInternalEvent(id: string) {
      return request<InternalEvent>(`/v1/scheduling/calendar-events/${id}`);
    },
    createInternalEvent(payload: CreateInternalEventPayload) {
      return request<InternalEvent>('/v1/scheduling/calendar-events', { method: 'POST', body: payload });
    },
    updateInternalEvent(id: string, payload: UpdateInternalEventPayload) {
      return request<InternalEvent>(`/v1/scheduling/calendar-events/${id}`, { method: 'PATCH', body: payload });
    },
    deleteInternalEvent(id: string) {
      return request<void>(`/v1/scheduling/calendar-events/${id}`, { method: 'DELETE' });
    },
    listSlots(query: SlotQuery) {
      return request<{ items: TimeSlot[] }>(
        `/v1/scheduling/slots${queryString({
          branch_id: query.branchId,
          service_id: query.serviceId,
          resource_id: query.resourceId,
          date: query.date,
        })}`,
      ).then((response) => response.items ?? []);
    },
    listBookings(filter: ListBookingsFilter = {}) {
      return request<{ items: Booking[] }>(
        `/v1/scheduling/bookings${queryString({
          branch_id: filter.branchId,
          date: filter.date,
          status: filter.status,
          limit: filter.limit ?? 200,
        })}`,
      ).then((response) => response.items ?? []);
    },
    getBooking(id: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}`);
    },
    createBooking(payload: CreateBookingPayload) {
      return request<Booking>('/v1/scheduling/bookings', { method: 'POST', body: payload });
    },
    confirmBooking(id: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/confirm`, { method: 'POST', body: {} });
    },
    cancelBooking(id: string, reason?: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/cancel`, {
        method: 'POST',
        body: { reason: reason ?? '' },
      });
    },
    checkInBooking(id: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/check-in`, { method: 'POST', body: {} });
    },
    startService(id: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/start-service`, { method: 'POST', body: {} });
    },
    completeBooking(id: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/complete`, { method: 'POST', body: {} });
    },
    markBookingNoShow(id: string, reason?: string) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/no-show`, {
        method: 'POST',
        body: { reason: reason ?? '' },
      });
    },
    rescheduleBooking(id: string, payload: RescheduleBookingPayload) {
      return request<Booking>(`/v1/scheduling/bookings/${id}/reschedule`, { method: 'POST', body: payload });
    },
    listWaitlist(branchId?: string | null, serviceId?: string | null, status?: string, limit = 100) {
      return request<{ items: WaitlistEntry[] }>(
        `/v1/scheduling/waitlist${queryString({
          branch_id: branchId,
          service_id: serviceId,
          status,
          limit,
        })}`,
      ).then((response) => response.items ?? []);
    },
    listQueues(branchId?: string | null) {
      return request<{ items: Queue[] }>(`/v1/scheduling/queues${queryString({ branch_id: branchId })}`).then(
        (response) => response.items ?? [],
      );
    },
    createQueueTicket(queueId: string, payload: {
      customer_name: string;
      customer_phone: string;
      customer_email?: string;
      priority?: number;
      notes?: string;
    }) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets`, { method: 'POST', body: payload });
    },
    getQueuePosition(queueId: string, ticketId: string) {
      return request<QueuePosition>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/position`);
    },
    pauseQueue(queueId: string) {
      return request<Queue>(`/v1/scheduling/queues/${queueId}/pause`, { method: 'POST', body: {} });
    },
    reopenQueue(queueId: string) {
      return request<Queue>(`/v1/scheduling/queues/${queueId}/reopen`, { method: 'POST', body: {} });
    },
    closeQueue(queueId: string) {
      return request<Queue>(`/v1/scheduling/queues/${queueId}/close`, { method: 'POST', body: {} });
    },
    callNext(queueId: string, payload?: { serving_resource_id?: string; operator_user_id?: string }) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/next`, {
        method: 'POST',
        body: payload ?? {},
      });
    },
    serveTicket(queueId: string, ticketId: string, payload?: { serving_resource_id?: string; operator_user_id?: string }) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/serve`, {
        method: 'POST',
        body: payload ?? {},
      });
    },
    completeTicket(queueId: string, ticketId: string, payload?: { serving_resource_id?: string; operator_user_id?: string }) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/complete`, {
        method: 'POST',
        body: payload ?? {},
      });
    },
    markTicketNoShow(queueId: string, ticketId: string) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/no-show`, {
        method: 'POST',
        body: {},
      });
    },
    cancelTicket(queueId: string, ticketId: string) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/cancel`, {
        method: 'POST',
        body: {},
      });
    },
    returnTicketToWaiting(queueId: string, ticketId: string) {
      return request<QueueTicket>(`/v1/scheduling/queues/${queueId}/tickets/${ticketId}/return-to-waiting`, {
        method: 'POST',
        body: {},
      });
    },
    getDashboard(branchId: string | null | undefined, day: string) {
      return request<DashboardStats>(`/v1/scheduling/dashboard${queryString({ branch_id: branchId, day })}`);
    },
    getDayAgenda(branchId: string | null | undefined, day: string) {
      return request<{ items: DayAgendaItem[] }>(`/v1/scheduling/day${queryString({ branch_id: branchId, day })}`).then(
        (response) => response.items ?? [],
      );
    },
  };
}

export function createPublicSchedulingClient(request: SchedulingTransport) {
  return {
    getBusinessInfo(orgId: string) {
      return request<PublicBusinessInfo>(`/v1/public/${orgId}/info`).then((response) => ({
        ...response,
        scheduling_enabled:
          typeof response.scheduling_enabled === 'boolean'
            ? response.scheduling_enabled
            : Boolean(response.appointments_enabled),
        appointments_enabled:
          typeof response.scheduling_enabled === 'boolean'
            ? response.scheduling_enabled
            : Boolean(response.appointments_enabled),
      }));
    },
    listServices(orgId: string) {
      return request<{ items: PublicService[] }>(`/v1/public/${orgId}/scheduling/services`).then(
        (response) => response.items ?? [],
      );
    },
    getAvailability(orgId: string, query: PublicAvailabilityQuery) {
      return request<{ slots?: PublicAvailabilitySlot[] }>(
        `/v1/public/${orgId}/scheduling/availability${queryString({
          branch_id: query.branchId,
          service_id: query.serviceId,
          resource_id: query.resourceId,
          date: query.date,
          duration: query.duration,
        })}`,
      ).then((response) => response.slots ?? []);
    },
    book(orgId: string, payload: PublicBookPayload) {
      return request<PublicBooking>(`/v1/public/${orgId}/scheduling/book`, { method: 'POST', body: payload });
    },
    listMyBookings(orgId: string, query: PublicMyBookingsQuery) {
      return request<{ items: PublicBooking[] }>(
        `/v1/public/${orgId}/scheduling/my-bookings${queryString({ phone: query.phone, limit: query.limit ?? 20 })}`,
      ).then((response) => response.items ?? []);
    },
    listQueues(orgId: string, branchId?: string | null) {
      return request<{ items: PublicQueueSummary[] }>(
        `/v1/public/${orgId}/scheduling/queues${queryString({ branch_id: branchId })}`,
      ).then((response) => response.items ?? []);
    },
    createQueueTicket(orgId: string, queueId: string, payload: PublicQueueTicketPayload) {
      return request<PublicQueueTicket>(`/v1/public/${orgId}/scheduling/queues/${queueId}/tickets`, {
        method: 'POST',
        body: payload,
      });
    },
    getQueuePosition(orgId: string, queueId: string, ticketId: string) {
      return request<PublicQueuePosition>(`/v1/public/${orgId}/scheduling/queues/${queueId}/tickets/${ticketId}/position`);
    },
    joinWaitlist(orgId: string, payload: PublicWaitlistPayload) {
      return request<PublicWaitlistEntry>(`/v1/public/${orgId}/scheduling/waitlist`, {
        method: 'POST',
        body: payload,
      });
    },
    confirmBooking(orgId: string, token: string) {
      return request<PublicBooking>(`/v1/public/${orgId}/scheduling/bookings/actions/confirm`, {
        method: 'POST',
        body: { token },
      });
    },
    cancelBooking(orgId: string, token: string, reason?: string) {
      return request<PublicBooking>(`/v1/public/${orgId}/scheduling/bookings/actions/cancel`, {
        method: 'POST',
        body: { token, reason: reason ?? '' },
      });
    },
  };
}
