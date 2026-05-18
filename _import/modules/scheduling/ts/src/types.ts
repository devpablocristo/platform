export type SchedulingRequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
};

export type SchedulingTransport = <T>(path: string, options?: SchedulingRequestOptions) => Promise<T>;

export type FulfillmentMode = 'schedule' | 'queue' | 'hybrid';
export type ResourceKind = 'professional' | 'desk' | 'counter' | 'box' | 'room' | 'generic';
export type AvailabilityRuleKind = 'branch' | 'resource';
export type BookingStatus =
  | 'hold'
  | 'pending_confirmation'
  | 'confirmed'
  | 'checked_in'
  | 'in_service'
  | 'completed'
  | 'cancelled'
  | 'no_show'
  | 'expired';
export type BookingSource = 'admin' | 'public_web' | 'whatsapp' | 'api';
export type QueueStatus = 'active' | 'paused' | 'closed';
export type QueueStrategy = 'fifo' | 'priority';
export type QueueTicketStatus = 'waiting' | 'called' | 'serving' | 'completed' | 'no_show' | 'cancelled';
export type QueueTicketSource = 'reception' | 'web' | 'whatsapp' | 'api';
export type WaitlistStatus = 'pending' | 'notified' | 'booked' | 'cancelled' | 'expired';
export type WaitlistSource = 'admin' | 'public_web' | 'whatsapp' | 'api';

export type Branch = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  timezone: string;
  address: string;
  active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type Service = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  description: string;
  fulfillment_mode: FulfillmentMode;
  default_duration_minutes: number;
  buffer_before_minutes: number;
  buffer_after_minutes: number;
  slot_granularity_minutes: number;
  max_concurrent_bookings: number;
  min_cancel_notice_minutes: number;
  allow_waitlist: boolean;
  active: boolean;
  resource_ids?: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type Resource = {
  id: string;
  org_id: string;
  branch_id: string;
  code: string;
  name: string;
  kind: ResourceKind;
  capacity: number;
  timezone?: string;
  active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type AvailabilityRule = {
  id: string;
  org_id: string;
  branch_id: string;
  resource_id?: string | null;
  kind: AvailabilityRuleKind;
  weekday: number;
  start_time: string;
  end_time: string;
  slot_granularity_minutes?: number | null;
  valid_from?: string | null;
  valid_until?: string | null;
  active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type BlockedRangeKind = 'holiday' | 'manual' | 'maintenance' | 'leave';

export type BlockedRange = {
  id: string;
  org_id: string;
  branch_id: string;
  resource_id?: string | null;
  kind: BlockedRangeKind;
  reason: string;
  start_at: string;
  end_at: string;
  all_day: boolean;
  created_by?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
};

export type BlockedRangePayload = {
  branch_id: string;
  resource_id?: string | null;
  kind: BlockedRangeKind;
  reason?: string;
  start_at: string;
  end_at: string;
  all_day?: boolean;
  metadata?: Record<string, unknown>;
};

export type ListBlockedRangesQuery = {
  branchId?: string | null;
  resourceId?: string | null;
  date?: string | null;
};

export type TimeSlot = {
  resource_id: string;
  resource_name: string;
  start_at: string;
  end_at: string;
  occupies_from: string;
  occupies_until: string;
  timezone: string;
  remaining: number;
  conflict_count: number;
  granularity_minutes: number;
};

export type Booking = {
  id: string;
  org_id: string;
  branch_id: string;
  service_id: string;
  resource_id: string;
  party_id?: string | null;
  reference: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  status: BookingStatus;
  source: BookingSource;
  idempotency_key?: string;
  start_at: string;
  end_at: string;
  occupies_from: string;
  occupies_until: string;
  hold_expires_at?: string | null;
  notes: string;
  metadata?: Record<string, unknown>;
  created_by?: string;
  confirmed_at?: string | null;
  cancelled_at?: string | null;
  reminder_sent_at?: string | null;
  created_at: string;
  updated_at: string;
};

// InternalEventStatus / InternalEventVisibility espejo del dominio Go en
// modules/scheduling/go/domain/entities.go (CalendarEventStatus / Visibility).
// El nombre TS evita choque con el wrapper `CalendarEvent` de abajo, que es el
// item renderizable de la grilla de FullCalendar.
export type InternalEventStatus = 'scheduled' | 'done' | 'cancelled';
export type InternalEventVisibility = 'team' | 'private';

// InternalEvent es la entidad persistida en la tabla scheduling_calendar_events.
// No es un turno cliente: no tiene customer ni service ni lifecycle de booking.
// Si tiene `resource_id`, ocupa ese recurso y resta del slot picker externo;
// sin `resource_id` es tiempo personal del owner y no afecta disponibilidad.
export type InternalEvent = {
  id: string;
  org_id: string;
  branch_id?: string | null;
  resource_id?: string | null;
  title: string;
  description?: string;
  start_at: string;
  end_at: string;
  all_day: boolean;
  status: InternalEventStatus;
  visibility: InternalEventVisibility;
  created_by?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type CreateInternalEventPayload = {
  branch_id?: string | null;
  resource_id?: string | null;
  title: string;
  description?: string;
  start_at: string;
  end_at: string;
  all_day?: boolean;
  status?: InternalEventStatus;
  visibility?: InternalEventVisibility;
  metadata?: Record<string, unknown>;
};

export type UpdateInternalEventPayload = CreateInternalEventPayload;

export type ListInternalEventsQuery = {
  branch_id?: string | null;
  resource_id?: string | null;
  from?: string | null;
  to?: string | null;
  status?: InternalEventStatus | null;
};

// CalendarEvent es el wrapper que renderiza la grilla de FullCalendar para
// turnos cliente. Los eventos internos NO usan este wrapper: se inyectan
// directamente como `EventInput` con `extendedProps.internalEvent`. Mantener
// los dos paths separados evita meter branches por kind en cada render y deja
// el wrapper acotado al dominio que originalmente representaba.
export type CalendarEventKind = 'booking';
export type CalendarEventSourceType = 'booking';

export type CalendarEvent = {
  id: string;
  kind: CalendarEventKind;
  sourceId: string;
  sourceType: CalendarEventSourceType;
  title: string;
  start_at: string;
  end_at: string;
  color: string;
  status: BookingStatus;
  serviceName?: string;
  resourceName?: string;
  sourceBooking: Booking;
};

export type DashboardStats = {
  date: string;
  timezone: string;
  bookings_today: number;
  confirmed_bookings_today: number;
  active_queues: number;
  waiting_tickets: number;
  tickets_in_service: number;
};

export type DayAgendaItem = {
  type: string;
  id: string;
  branch_id: string;
  service_id?: string | null;
  start_at?: string | null;
  end_at?: string | null;
  status: string;
  label: string;
  metadata?: Record<string, unknown>;
};

export type Queue = {
  id: string;
  org_id: string;
  branch_id: string;
  service_id?: string | null;
  code: string;
  name: string;
  status: QueueStatus;
  strategy: QueueStrategy;
  ticket_prefix: string;
  avg_service_seconds: number;
  allow_remote_join: boolean;
  metadata?: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
};

export type QueueTicket = {
  id: string;
  org_id: string;
  queue_id: string;
  branch_id: string;
  service_id?: string | null;
  party_id?: string | null;
  number: number;
  display_code: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  status: QueueTicketStatus;
  source: QueueTicketSource;
  priority: number;
  idempotency_key?: string;
  notes: string;
  metadata?: Record<string, unknown>;
  requested_at: string;
  called_at?: string | null;
  started_at?: string | null;
  completed_at?: string | null;
  cancelled_at?: string | null;
  no_show_at?: string | null;
  serving_resource_id?: string | null;
  operator_user_id?: string | null;
  created_by?: string;
  updated_at: string;
};

export type QueuePosition = {
  ticket_id: string;
  queue_id: string;
  status: QueueTicketStatus;
  position: number;
  estimated_wait_seconds: number;
};

export type WaitlistEntry = {
  id: string;
  org_id: string;
  branch_id: string;
  service_id: string;
  resource_id?: string | null;
  party_id?: string | null;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  requested_start_at: string;
  status: WaitlistStatus;
  source: WaitlistSource;
  idempotency_key?: string;
  notes: string;
  metadata?: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
};

export type ListBookingsFilter = {
  branchId?: string | null;
  date?: string | null;
  status?: string;
  limit?: number;
};

export type SlotQuery = {
  branchId: string;
  serviceId: string;
  date: string;
  resourceId?: string | null;
};

export type BookingRecurrence = {
  freq: 'daily' | 'weekly' | 'monthly';
  interval?: number;
  count?: number;
  until?: string;
  by_weekday?: number[];
};

export type CreateBookingPayload = {
  branch_id: string;
  service_id: string;
  resource_id?: string;
  party_id?: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  start_at: string;
  end_at?: string;
  status?: BookingStatus;
  source?: BookingSource;
  idempotency_key?: string;
  hold_until?: string;
  notes?: string;
  metadata?: Record<string, unknown>;
  recurrence?: BookingRecurrence;
};

export type RescheduleBookingPayload = {
  branch_id?: string;
  resource_id?: string;
  start_at: string;
  end_at?: string;
};

export type BookingActionResult = Booking;

export type PublicService = {
  id: string;
  name: string;
  type: string;
  description: string;
  unit: string;
  price: number;
  currency: string;
};

export type PublicAvailabilitySlot = {
  start_at: string;
  end_at: string;
  remaining: number;
};

export type PublicBookingActionLinks = {
  confirm_token?: string;
  cancel_token?: string;
  confirm_path?: string;
  cancel_path?: string;
};

export type PublicBooking = {
  id: string;
  party_name: string;
  party_phone: string;
  customer_email?: string;
  title: string;
  status: BookingStatus;
  start_at: string;
  end_at: string;
  duration: number;
  actions?: PublicBookingActionLinks;
};

export type PublicBusinessInfo = {
  org_id: string;
  name: string;
  slug: string;
  business_name: string;
  business_address: string;
  business_phone: string;
  business_email: string;
  scheduling_enabled: boolean;
  appointments_enabled?: boolean;
};

export type PublicQueueSummary = {
  id: string;
  code: string;
  name: string;
  status: QueueStatus;
  allow_remote_join: boolean;
  avg_service_seconds: number;
};

export type PublicQueueTicket = {
  ticket: QueueTicket;
  position: PublicQueuePosition;
};

export type PublicQueuePosition = {
  ticket_id: string;
  queue_id: string;
  status: QueueTicketStatus;
  position: number;
  estimated_wait_seconds: number;
};

export type PublicWaitlistEntry = {
  id: string;
  branch_id: string;
  service_id: string;
  resource_id?: string | null;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  requested_start_at: string;
  status: WaitlistStatus;
};

export type PublicAvailabilityQuery = {
  branchId?: string | null;
  serviceId?: string | null;
  date: string;
  resourceId?: string | null;
  duration?: number;
};

export type PublicBookPayload = {
  branch_id?: string;
  service_id?: string;
  resource_id?: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  start_at: string;
  notes?: string;
};

export type PublicMyBookingsQuery = {
  phone: string;
  limit?: number;
};

export type PublicQueueTicketPayload = {
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  priority?: number;
  notes?: string;
};

export type PublicWaitlistPayload = {
  branch_id: string;
  service_id: string;
  resource_id?: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  requested_start_at: string;
  notes?: string;
};

export type QueueOperatorBoardStatusCopy = Record<QueueTicketStatus | QueueStatus, string>;

export type QueueOperatorBoardCopy = {
  title: string;
  description: string;
  branchLabel: string;
  dateLabel: string;
  loading: string;
  noQueues: string;
  issueTicketTitle: string;
  issueTicketDescription: string;
  customerNameLabel: string;
  customerPhoneLabel: string;
  customerEmailLabel: string;
  priorityLabel: string;
  issueTicket: string;
  issuingTicket: string;
  callNext: string;
  pauseQueue: string;
  reopenQueue: string;
  closeQueue: string;
  waitingColumn: string;
  activeColumn: string;
  finishedColumn: string;
  noTickets: string;
  requestedAtLabel: string;
  statusLabel: string;
  serveTicket: string;
  completeTicket: string;
  noShowTicket: string;
  cancelTicket: string;
  returnToWaiting: string;
  nextTicketTitle: string;
  queueMetricsIssued: string;
  queueMetricsWaiting: string;
  queueMetricsServing: string;
  queueMetricsDone: string;
  confirmDangerTitle: string;
  dismissConfirm: string;
  confirmCancelDescription: string;
  confirmNoShowDescription: string;
  closeQueueDescription: string;
  statuses: QueueOperatorBoardStatusCopy;
};

export type PublicSchedulingFlowCopy = {
  title: string;
  description: string;
  orgRefLabel: string;
  orgRefHelp: string;
  loadOrg: string;
  businessInfoTitle: string;
  serviceLabel: string;
  dateLabel: string;
  phoneLabel: string;
  nameLabel: string;
  emailLabel: string;
  notesLabel: string;
  availabilityTitle: string;
  availabilityDescription: string;
  availabilityEmpty: string;
  availabilityLoading: string;
  selectSlot: string;
  selectedSlotLabel: string;
  bookNow: string;
  booking: string;
  myBookingsTitle: string;
  myBookingsDescription: string;
  findBookings: string;
  findingBookings: string;
  noBookings: string;
  queuesTitle: string;
  queuesDescription: string;
  joinQueue: string;
  joiningQueue: string;
  etaLabel: string;
  positionLabel: string;
  ticketCodeLabel: string;
  publicDisabledTitle: string;
  publicDisabledDescription: string;
  loading: string;
  bookingCreatedTitle: string;
  queueCreatedTitle: string;
  confirmBooking: string;
  cancelBooking: string;
  cancelBookingReason: string;
  statuses: Record<string, string>;
};

export type SchedulingCalendarStatusCopy = Record<BookingStatus, string>;

export type SchedulingCalendarCopy = {
  branchLabel: string;
  serviceLabel: string;
  resourceLabel: string;
  anyResource: string;
  focusDateLabel: string;
  summaryTitle: string;
  summaryBookings: string;
  summaryConfirmed: string;
  summaryQueues: string;
  summaryWaiting: string;
  slotsTitle: string;
  slotsDescription: string;
  slotsEmpty: string;
  slotsLoading: string;
  loading: string;
  unavailableTitle: string;
  unavailableDescription: string;
  noServicesTitle: string;
  noServicesDescription: string;
  /** Fallo al cargar sucursales/servicios (red, permisos, servidor): no confundir con “sin configurar”. */
  loadErrorTitle: string;
  loadErrorDescription: string;
  retryLoad: string;
  filtersTitle: string;
  filtersDescription: string;
  timelineTitle: string;
  timelineDescription: string;
  openBooking: string;
  titleLabel: string;
  repeatLabel: string;
  repeatNever: string;
  repeatDaily: string;
  repeatWeekly: string;
  repeatMonthly: string;
  repeatCustom: string;
  repeatFrequencyLabel: string;
  repeatIntervalLabel: string;
  repeatCountLabel: string;
  repeatWeekdaysLabel: string;
  bookingTitleCreate: string;
  bookingTitleDetails: string;
  bookingSubtitleCreate: string;
  bookingSubtitleDetails: string;
  availableSlotLabel: string;
  availableSlotHint: string;
  availableSlotLoading: string;
  unavailableSlotMessage: string;
  slotSummaryTitle: string;
  bookingPreviewTitle: string;
  customerNameLabel: string;
  customerPhoneLabel: string;
  customerEmailLabel: string;
  notesLabel: string;
  statusLabel: string;
  serviceNameLabel: string;
  resourceNameLabel: string;
  slotLabel: string;
  slotStartLabel: string;
  slotEndLabel: string;
  durationLabel: string;
  timezoneLabel: string;
  occupiesLabel: string;
  conflictLabel: string;
  slotRemainingLabel: string;
  referenceLabel: string;
  close: string;
  create: string;
  saving: string;
  cancelBooking: string;
  confirmBooking: string;
  checkInBooking: string;
  startService: string;
  completeBooking: string;
  noShowBooking: string;
  rescheduleBooking: string;
  dragRescheduleTitle: string;
  dragRescheduleDescription: string;
  destructiveTitle: string;
  cancelActionDescription: string;
  noShowActionDescription: string;
  closeDirtyTitle: string;
  closeDirtyDescription: string;
  keepEditing: string;
  discard: string;
  resizeLockedMessage: string;
  resizeBookingTitle: string;
  resizeBookingDescription: string;
  searchPlaceholder: string;
  statuses: SchedulingCalendarStatusCopy;
  blockedRangeAction: string;
  blockedRangeEyebrow: string;
  blockedRangeCreateTitle: string;
  blockedRangeEditTitle: string;
  blockedRangeKindLabel: string;
  blockedRangeKindOptions: Record<BlockedRangeKind, string>;
  blockedRangeReasonLabel: string;
  blockedRangeReasonPlaceholder: string;
  blockedRangeCreate: string;
  blockedRangeUpdate: string;
  blockedRangeDelete: string;
  blockedRangeDeleteTitle: string;
  blockedRangeDeleteDescription: string;
  blockedRangeFallbackTitle: string;
  // copy del flujo de eventos internos (Etapa 3 / agenda Google-like)
  internalEventFallbackTitle: string;
  internalEventCreateAction: string;
  internalEventModalCreateTitle: string;
  internalEventModalEditTitle: string;
  internalEventTitleLabel: string;
  internalEventTitlePlaceholder: string;
  internalEventDescriptionLabel: string;
  internalEventResourceLabel: string;
  internalEventResourceUnassigned: string;
  internalEventStartLabel: string;
  internalEventEndLabel: string;
  internalEventStatusLabel: string;
  internalEventStatusOptions: { scheduled: string; done: string; cancelled: string };
  internalEventVisibilityLabel: string;
  internalEventVisibilityOptions: { team: string; private: string };
  internalEventSave: string;
  internalEventDelete: string;
  internalEventDeleteConfirm: string;
  newEntryButton: string;
  newEntryMenuBooking: string;
  newEntryMenuInternalEvent: string;
  newEntryMenuBlockedRange: string;
};

// Tipo de entrada que el dueño puede crear desde el calendario interno.
// Se usa en el switcher de tipo embebido en los 3 modales (booking, evento
// interno, bloqueo) para que el usuario pueda alternar sin volver al toolbar.
export type SchedulingEntryType = 'booking' | 'internal_event' | 'blocked_range';

// Contexto que se preserva al cambiar de tipo: fecha + rango horario opcional
// del gesto original (click o select sobre el grid).
export type SchedulingEntryContext = {
  date: string;
  start?: Date;
  end?: Date;
};
