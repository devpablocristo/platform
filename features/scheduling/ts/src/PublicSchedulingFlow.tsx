import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { PublicSchedulingClient } from './client';
import { formatSchedulingDateTime, resolveSchedulingCopyLocale } from './locale';
import type {
  PublicAvailabilitySlot,
  PublicSchedulingFlowCopy,
  PublicService,
} from './types';

const publicKeys = {
  info: (orgRef: string) => ['public-scheduling', 'info', orgRef] as const,
  services: (orgRef: string) => ['public-scheduling', 'services', orgRef] as const,
  availability: (orgRef: string, serviceId: string | null, date: string) =>
    ['public-scheduling', 'availability', orgRef, serviceId ?? 'none', date] as const,
  queues: (orgRef: string) => ['public-scheduling', 'queues', orgRef] as const,
};

export const publicSchedulingFlowCopyPresets: Record<'en' | 'es', PublicSchedulingFlowCopy> = {
  en: {
    title: 'Public booking flow',
    description: 'Preview the public booking flow using the same public API contract.',
    orgRefLabel: 'Organization reference',
    orgRefHelp: 'You can use the public slug or the organization UUID.',
    loadOrg: 'Load organization',
    businessInfoTitle: 'Business profile',
    serviceLabel: 'Service',
    dateLabel: 'Date',
    phoneLabel: 'Phone',
    nameLabel: 'Customer name',
    emailLabel: 'Email',
    notesLabel: 'Notes',
    availabilityTitle: 'Availability',
    availabilityDescription: 'Customers book directly from available public slots.',
    availabilityEmpty: 'No public slots available for the selected date.',
    availabilityLoading: 'Loading availability…',
    selectSlot: 'Select slot',
    selectedSlotLabel: 'Selected slot',
    bookNow: 'Book now',
    booking: 'Booking…',
    queuesTitle: 'Remote queues',
    queuesDescription: 'Customers can also join a virtual queue without opening the dashboard.',
    joinQueue: 'Join queue',
    joiningQueue: 'Joining…',
    etaLabel: 'ETA',
    positionLabel: 'Position',
    ticketCodeLabel: 'Ticket',
    publicDisabledTitle: 'Public booking flow is disabled',
    publicDisabledDescription: 'Enable the schedule for this organization to expose the public booking flow.',
    loading: 'Loading public booking flow…',
    bookingCreatedTitle: 'Booking created',
    queueCreatedTitle: 'Queue ticket created',
    statuses: {
      hold: 'On hold',
      pending_confirmation: 'Pending confirmation',
      confirmed: 'Confirmed',
      checked_in: 'Checked in',
      in_service: 'In service',
      completed: 'Completed',
      cancelled: 'Cancelled',
      no_show: 'No-show',
      expired: 'Expired',
      active: 'Active',
      paused: 'Paused',
      closed: 'Closed',
      waiting: 'Waiting',
      called: 'Called',
      serving: 'Serving',
    },
  },
  es: {
    title: 'Reserva pública',
    description: 'Vista previa del flujo público de reservas con el mismo contrato de la API pública.',
    orgRefLabel: 'Referencia de organización',
    orgRefHelp: 'Puede ser el slug público o el UUID de la organización.',
    loadOrg: 'Cargar organización',
    businessInfoTitle: 'Perfil del negocio',
    serviceLabel: 'Servicio',
    dateLabel: 'Fecha',
    phoneLabel: 'Teléfono',
    nameLabel: 'Cliente',
    emailLabel: 'Email',
    notesLabel: 'Notas',
    availabilityTitle: 'Disponibilidad',
    availabilityDescription: 'El cliente reserva directamente desde los slots públicos.',
    availabilityEmpty: 'No hay slots públicos para la fecha seleccionada.',
    availabilityLoading: 'Cargando disponibilidad…',
    selectSlot: 'Elegir slot',
    selectedSlotLabel: 'Slot elegido',
    bookNow: 'Reservar',
    booking: 'Reservando…',
    queuesTitle: 'Colas remotas',
    queuesDescription: 'El cliente también puede sumarse a una cola virtual sin entrar al dashboard.',
    joinQueue: 'Sumarme',
    joiningQueue: 'Uniéndome…',
    etaLabel: 'Tiempo estimado',
    positionLabel: 'Posición',
    ticketCodeLabel: 'Ticket',
    publicDisabledTitle: 'La agenda pública está deshabilitada',
    publicDisabledDescription: 'Activá la agenda de esta organización para exponer el flujo público.',
    loading: 'Cargando agenda pública…',
    bookingCreatedTitle: 'Reserva creada',
    queueCreatedTitle: 'Ticket emitido',
    statuses: {
      hold: 'En espera',
      pending_confirmation: 'Pendiente',
      confirmed: 'Confirmada',
      checked_in: 'Check-in',
      in_service: 'En atención',
      completed: 'Completada',
      cancelled: 'Cancelada',
      no_show: 'No-show',
      expired: 'Expirada',
      active: 'Activa',
      paused: 'Pausada',
      closed: 'Cerrada',
      waiting: 'Esperando',
      called: 'Llamado',
      serving: 'Atendiendo',
    },
  },
};

export type PublicSchedulingFlowProps = {
  client: PublicSchedulingClient;
  orgRef: string;
  locale?: string;
  copy?: Partial<PublicSchedulingFlowCopy>;
  className?: string;
};

type ContactDraft = {
  name: string;
  phone: string;
  email: string;
  notes: string;
};

function todayValue(): string {
  return new Date().toISOString().slice(0, 10);
}

function statusLabel(copy: PublicSchedulingFlowCopy, status: string): string {
  return copy.statuses[status] ?? status;
}

export function PublicSchedulingFlow({
  client,
  orgRef,
  locale = 'en',
  copy: copyOverrides,
  className = '',
}: PublicSchedulingFlowProps) {
  const copy = { ...publicSchedulingFlowCopyPresets[resolveSchedulingCopyLocale(locale)], ...copyOverrides };
  const queryClient = useQueryClient();
  const [selectedServiceId, setSelectedServiceId] = useState<string>('');
  const [selectedDate, setSelectedDate] = useState(todayValue());
  const [selectedSlot, setSelectedSlot] = useState<PublicAvailabilitySlot | null>(null);
  const [contact, setContact] = useState<ContactDraft>({ name: '', phone: '', email: '', notes: '' });
  const [feedback, setFeedback] = useState<string | null>(null);
  const trimmedOrgRef = orgRef.trim();

  const infoQuery = useQuery({
    queryKey: publicKeys.info(trimmedOrgRef),
    queryFn: () => client.getBusinessInfo(trimmedOrgRef),
    enabled: trimmedOrgRef.length > 0,
    staleTime: 60_000,
  });

  const servicesQuery = useQuery({
    queryKey: publicKeys.services(trimmedOrgRef),
    queryFn: () => client.listServices(trimmedOrgRef),
    enabled: trimmedOrgRef.length > 0,
    staleTime: 60_000,
  });

  const queuesQuery = useQuery({
    queryKey: publicKeys.queues(trimmedOrgRef),
    queryFn: () => client.listQueues(trimmedOrgRef),
    enabled: trimmedOrgRef.length > 0,
    staleTime: 30_000,
  });

  const availabilityQuery = useQuery({
    queryKey: publicKeys.availability(trimmedOrgRef, selectedServiceId || null, selectedDate),
    queryFn: () =>
      client.getAvailability(trimmedOrgRef, {
        serviceId: selectedServiceId || undefined,
        date: selectedDate,
      }),
    enabled: trimmedOrgRef.length > 0 && selectedServiceId.length > 0,
    staleTime: 15_000,
  });

  useEffect(() => {
    if (selectedServiceId) {
      return;
    }
    const firstService = servicesQuery.data?.[0];
    if (firstService) {
      setSelectedServiceId(firstService.id);
    }
  }, [servicesQuery.data, selectedServiceId]);

  useEffect(() => {
    setSelectedSlot(null);
  }, [selectedDate, selectedServiceId, trimmedOrgRef]);

  const bookMutation = useMutation({
    mutationFn: async () =>
      client.book(trimmedOrgRef, {
        service_id: selectedServiceId || undefined,
        customer_name: contact.name.trim(),
        customer_phone: contact.phone.trim(),
        customer_email: contact.email.trim() || undefined,
        start_at: selectedSlot?.start_at ?? '',
        notes: contact.notes.trim() || undefined,
      }),
    onMutate: () => setFeedback(null),
    onSuccess: async () => {
      setFeedback(copy.bookingCreatedTitle);
      await queryClient.invalidateQueries({
        queryKey: publicKeys.availability(trimmedOrgRef, selectedServiceId || null, selectedDate),
      });
    },
    onError: (error: Error) => setFeedback(error.message),
  });

  const joinQueueMutation = useMutation({
    mutationFn: async (queueId: string) =>
      client.createQueueTicket(trimmedOrgRef, queueId, {
        customer_name: contact.name.trim(),
        customer_phone: contact.phone.trim(),
        customer_email: contact.email.trim() || undefined,
      }),
    onMutate: () => setFeedback(null),
    onSuccess: () => {
      setFeedback(copy.queueCreatedTitle);
    },
    onError: (error: Error) => setFeedback(error.message),
  });

  const services = servicesQuery.data ?? [];
  const selectedService = services.find((service) => service.id === selectedServiceId) ?? null;

  if (!trimmedOrgRef) {
    return (
      <section className={`modules-scheduling ${className}`.trim()}>
        <div className="card empty-state">
          <p>{copy.orgRefHelp}</p>
        </div>
      </section>
    );
  }

  if (infoQuery.isLoading || servicesQuery.isLoading) {
    return (
      <section className={`modules-scheduling ${className}`.trim()}>
        <div className="card modules-scheduling__empty">
          <div className="spinner" />
          <p>{copy.loading}</p>
        </div>
      </section>
    );
  }

  const business = infoQuery.data;

  return (
    <section className={`modules-scheduling ${className}`.trim()}>
      {feedback ? <div className="alert alert-success">{feedback}</div> : null}

      {business ? (
        <div className="card modules-scheduling__public-business">
          <div className="card-header">
            <div>
              <h2>{copy.businessInfoTitle}</h2>
              <p className="text-secondary">{business.business_name || business.name}</p>
            </div>
            <span className="modules-scheduling__public-ref">{business.slug || trimmedOrgRef}</span>
          </div>
          <div className="modules-scheduling__public-business-meta">
            {business.business_address ? <span>{business.business_address}</span> : null}
            {business.business_phone ? <span>{business.business_phone}</span> : null}
            {business.business_email ? <span>{business.business_email}</span> : null}
          </div>
        </div>
      ) : null}

      {business && !business.scheduling_enabled ? (
        <div className="alert alert-warning">
          <strong>{copy.publicDisabledTitle}</strong>
          <div>{copy.publicDisabledDescription}</div>
        </div>
      ) : null}

      <div className="modules-scheduling__public-grid">
        <div className="card">
          <div className="card-header">
            <div>
              <h2>{copy.availabilityTitle}</h2>
              <p className="text-secondary">{copy.availabilityDescription}</p>
            </div>
          </div>

          <div className="modules-scheduling__filters">
            <div className="form-group grow">
              <label htmlFor="public-scheduling-service">{copy.serviceLabel}</label>
              <select
                id="public-scheduling-service"
                value={selectedServiceId}
                onChange={(event) => setSelectedServiceId(event.target.value)}
              >
                {services.map((service: PublicService) => (
                  <option key={service.id} value={service.id}>
                    {service.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="form-group">
              <label htmlFor="public-scheduling-date">{copy.dateLabel}</label>
              <input
                id="public-scheduling-date"
                type="date"
                value={selectedDate}
                onChange={(event) => setSelectedDate(event.target.value)}
              />
            </div>
          </div>

          {availabilityQuery.isLoading ? (
            <div className="modules-scheduling__empty">{copy.availabilityLoading}</div>
          ) : (availabilityQuery.data ?? []).length === 0 ? (
            <div className="modules-scheduling__empty">{copy.availabilityEmpty}</div>
          ) : (
            <div className="modules-scheduling__public-slots">
              {(availabilityQuery.data ?? []).map((slot) => (
                <button
                  key={slot.start_at}
                  type="button"
                  className={`modules-scheduling__public-slot${
                    selectedSlot?.start_at === slot.start_at ? ' modules-scheduling__public-slot--active' : ''
                  }`}
                  onClick={() => setSelectedSlot(slot)}
                >
                  <strong>{formatSchedulingDateTime(slot.start_at, locale)}</strong>
                  <span>{copy.selectSlot}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="card">
          <div className="card-header">
            <div>
              <h2>{selectedService?.name ?? copy.bookNow}</h2>
              <p className="text-secondary">
                {copy.selectedSlotLabel}: {selectedSlot ? formatSchedulingDateTime(selectedSlot.start_at, locale) : '—'}
              </p>
            </div>
          </div>
          <div className="modules-scheduling__public-form">
            <div className="form-group">
              <label htmlFor="public-scheduling-name">{copy.nameLabel}</label>
              <input
                id="public-scheduling-name"
                value={contact.name}
                onChange={(event) => setContact((current) => ({ ...current, name: event.target.value }))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="public-scheduling-phone">{copy.phoneLabel}</label>
              <input
                id="public-scheduling-phone"
                value={contact.phone}
                onChange={(event) => setContact((current) => ({ ...current, phone: event.target.value }))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="public-scheduling-email">{copy.emailLabel}</label>
              <input
                id="public-scheduling-email"
                value={contact.email}
                onChange={(event) => setContact((current) => ({ ...current, email: event.target.value }))}
              />
            </div>
            <div className="form-group">
              <label htmlFor="public-scheduling-notes">{copy.notesLabel}</label>
              <textarea
                id="public-scheduling-notes"
                rows={3}
                value={contact.notes}
                onChange={(event) => setContact((current) => ({ ...current, notes: event.target.value }))}
              />
            </div>
          </div>
          <button
            type="button"
            className="btn-primary"
            disabled={!selectedSlot || !contact.name.trim() || !contact.phone.trim() || bookMutation.isPending}
            onClick={() => void bookMutation.mutateAsync()}
          >
            {bookMutation.isPending ? copy.booking : copy.bookNow}
          </button>
        </div>
      </div>

      <div className="modules-scheduling__public-grid">
        <div className="card">
          <div className="card-header">
            <div>
              <h2>{copy.queuesTitle}</h2>
              <p className="text-secondary">{copy.queuesDescription}</p>
            </div>
          </div>
          <div className="modules-scheduling__public-queues">
            {(queuesQuery.data ?? []).map((queue) => (
              <div key={queue.id} className="modules-scheduling__public-queue-card">
                <div className="modules-scheduling__public-booking-head">
                  <strong>{queue.name}</strong>
                  <span className="modules-scheduling__badge modules-scheduling__badge--attention">
                    {statusLabel(copy, queue.status)}
                  </span>
                </div>
                <div className="modules-scheduling__public-booking-meta">
                  <span>{queue.code}</span>
                  <span>
                    {copy.etaLabel}: {Math.max(queue.avg_service_seconds, 0)}s
                  </span>
                </div>
                <button
                  type="button"
                  className="btn-secondary btn-sm"
                  disabled={!queue.allow_remote_join || !contact.name.trim() || !contact.phone.trim() || joinQueueMutation.isPending}
                  onClick={() => void joinQueueMutation.mutateAsync(queue.id)}
                >
                  {joinQueueMutation.isPending ? copy.joiningQueue : copy.joinQueue}
                </button>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
