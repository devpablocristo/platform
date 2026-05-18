import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { SchedulingClient } from './client';
import { formatSchedulingDateTime, resolveSchedulingCopyLocale } from './locale';
import type { DashboardStats, DayAgendaItem } from './types';

const summaryKeys = {
  dashboard: (date: string) => ['scheduling-summary', 'dashboard', date] as const,
  day: (date: string) => ['scheduling-summary', 'day', date] as const,
};

export const schedulingDaySummaryCopyPresets = {
  en: {
    title: "Today's schedule",
    description: 'Live bookings and queue activity for the current operating window.',
    dateLabel: 'Date',
    loading: 'Loading schedule snapshot…',
    empty: 'No schedule activity for this date.',
    bookings: 'Bookings',
    confirmed: 'Confirmed',
    activeQueues: 'Queues',
    waiting: 'Waiting',
    nextBooking: 'Next booking',
    queueFlow: 'Queue flow',
    noUpcoming: 'No upcoming bookings.',
    noQueueFlow: 'No queue tickets in the agenda.',
    statuses: {
      confirmed: 'Confirmed',
      waiting: 'Waiting',
      called: 'Called',
      serving: 'Serving',
      completed: 'Completed',
      cancelled: 'Cancelled',
      no_show: 'No-show',
    },
  },
  es: {
    title: 'Agenda de hoy',
    description: 'Reservas activas y movimiento de cola en la ventana operativa actual.',
    dateLabel: 'Fecha',
    loading: 'Cargando resumen de agenda…',
    empty: 'No hay actividad en la agenda para esta fecha.',
    bookings: 'Reservas',
    confirmed: 'Confirmadas',
    activeQueues: 'Colas',
    waiting: 'Esperando',
    nextBooking: 'Próxima reserva',
    queueFlow: 'Flujo de cola',
    noUpcoming: 'No hay reservas próximas.',
    noQueueFlow: 'No hay tickets de cola en la agenda.',
    statuses: {
      confirmed: 'Confirmada',
      waiting: 'Esperando',
      called: 'Llamado',
      serving: 'Atendiendo',
      completed: 'Completado',
      cancelled: 'Cancelado',
      no_show: 'No-show',
    },
  },
} as const;

export type SchedulingDaySummaryProps = {
  client: SchedulingClient;
  locale?: string;
  className?: string;
  initialDate?: string;
};

function todayValue(): string {
  return new Date().toISOString().slice(0, 10);
}

function isUpcomingBooking(item: DayAgendaItem): boolean {
  if (item.type !== 'booking' || !item.start_at) {
    return false;
  }
  return new Date(item.start_at).getTime() >= Date.now();
}

function isQueueItem(item: DayAgendaItem): boolean {
  return item.type === 'queue_ticket';
}

export function SchedulingDaySummary({
  client,
  locale = 'en',
  className = '',
  initialDate,
}: SchedulingDaySummaryProps) {
  const copy = schedulingDaySummaryCopyPresets[resolveSchedulingCopyLocale(locale)];
  const statusLabel = (status: string) => copy.statuses[status as keyof typeof copy.statuses] ?? status;
  const [selectedDate, setSelectedDate] = useState(initialDate ?? todayValue());

  const dashboardQuery = useQuery<DashboardStats>({
    queryKey: summaryKeys.dashboard(selectedDate),
    queryFn: () => client.getDashboard(undefined, selectedDate),
    staleTime: 20_000,
  });

  const dayQuery = useQuery<DayAgendaItem[]>({
    queryKey: summaryKeys.day(selectedDate),
    queryFn: () => client.getDayAgenda(undefined, selectedDate),
    staleTime: 20_000,
  });

  const nextBooking = useMemo(
    () => (dayQuery.data ?? []).filter(isUpcomingBooking).sort((a, b) => (a.start_at ?? '').localeCompare(b.start_at ?? ''))[0],
    [dayQuery.data],
  );
  const queueFlow = useMemo(() => (dayQuery.data ?? []).filter(isQueueItem).slice(0, 6), [dayQuery.data]);

  return (
    <div className={`card modules-scheduling__day-summary ${className}`.trim()}>
      <div className="card-header">
        <div>
          <h2>{copy.title}</h2>
          <p className="text-secondary">{copy.description}</p>
        </div>
        <div className="form-group">
          <label htmlFor="scheduling-day-summary-date">{copy.dateLabel}</label>
          <input
            id="scheduling-day-summary-date"
            type="date"
            value={selectedDate}
            onChange={(event) => setSelectedDate(event.target.value)}
          />
        </div>
      </div>

      {dashboardQuery.isLoading || dayQuery.isLoading ? (
        <div className="modules-scheduling__empty">
          <div className="spinner" />
          <p>{copy.loading}</p>
        </div>
      ) : !dashboardQuery.data ? (
        <div className="modules-scheduling__empty">{copy.empty}</div>
      ) : (
        <>
          <div className="modules-scheduling__summary stats-grid">
            <article className="stat-card">
              <div className="stat-label">{copy.bookings}</div>
              <div className="stat-value">{dashboardQuery.data.bookings_today}</div>
            </article>
            <article className="stat-card">
              <div className="stat-label">{copy.confirmed}</div>
              <div className="stat-value">{dashboardQuery.data.confirmed_bookings_today}</div>
            </article>
            <article className="stat-card">
              <div className="stat-label">{copy.activeQueues}</div>
              <div className="stat-value">{dashboardQuery.data.active_queues}</div>
            </article>
            <article className="stat-card">
              <div className="stat-label">{copy.waiting}</div>
              <div className="stat-value">{dashboardQuery.data.waiting_tickets}</div>
            </article>
          </div>

          <div className="modules-scheduling__day-summary-grid">
            <section className="modules-scheduling__day-summary-section">
              <div className="modules-scheduling__day-summary-title">{copy.nextBooking}</div>
              {nextBooking ? (
                <div className="modules-scheduling__day-summary-item">
                  <strong>{nextBooking.label}</strong>
                  <span>{formatSchedulingDateTime(nextBooking.start_at, locale)}</span>
                  <span>{statusLabel(nextBooking.status)}</span>
                </div>
              ) : (
                <div className="modules-scheduling__queue-empty">{copy.noUpcoming}</div>
              )}
            </section>
            <section className="modules-scheduling__day-summary-section">
              <div className="modules-scheduling__day-summary-title">{copy.queueFlow}</div>
              {queueFlow.length === 0 ? (
                <div className="modules-scheduling__queue-empty">{copy.noQueueFlow}</div>
              ) : (
                queueFlow.map((item) => (
                  <div key={item.id} className="modules-scheduling__day-summary-item">
                    <strong>{item.label}</strong>
                    <span>{statusLabel(item.status)}</span>
                  </div>
                ))
              )}
            </section>
          </div>
        </>
      )}
    </div>
  );
}
