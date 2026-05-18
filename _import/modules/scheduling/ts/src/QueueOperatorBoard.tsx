import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { confirmAction } from '@devpablocristo/core-browser';
import type { SchedulingClient } from './client';
import { resolveSchedulingCopyLocale } from './locale';
import type {
  Branch,
  DayAgendaItem,
  Queue,
  QueueOperatorBoardCopy,
  QueueStatus,
  QueueTicketStatus,
} from './types';

type QueueBoardTicket = {
  id: string;
  queueId: string;
  label: string;
  status: QueueTicketStatus;
  number?: number;
};

type TicketDraft = {
  customerName: string;
  customerPhone: string;
  customerEmail: string;
  priority: number;
};

const queueKeys = {
  branches: ['scheduling', 'branches'] as const,
  queues: (branchId: string | null) => ['scheduling', 'queues', branchId ?? 'all'] as const,
  day: (branchId: string | null, date: string) => ['scheduling', 'day', branchId ?? 'all', date] as const,
};

export const queueOperatorBoardCopyPresets: Record<'en' | 'es', QueueOperatorBoardCopy> = {
  en: {
    title: 'Virtual queues',
    description: 'Manage live queues from the same workspace.',
    branchLabel: 'Branch',
    dateLabel: 'Date',
    loading: 'Loading queues…',
    noQueues: 'No queues configured for this branch.',
    issueTicketTitle: 'Create ticket',
    issueTicketDescription: 'Create a new reception or walk-in ticket.',
    customerNameLabel: 'Customer name',
    customerPhoneLabel: 'Phone',
    customerEmailLabel: 'Email',
    priorityLabel: 'Priority',
    issueTicket: 'Create ticket',
    issuingTicket: 'Issuing…',
    callNext: 'Call next',
    pauseQueue: 'Pause queue',
    reopenQueue: 'Reopen queue',
    closeQueue: 'Close queue',
    waitingColumn: 'Waiting',
    activeColumn: 'Active',
    finishedColumn: 'Finished',
    noTickets: 'No tickets in this column.',
    requestedAtLabel: 'Requested',
    statusLabel: 'Status',
    serveTicket: 'Serve',
    completeTicket: 'Complete',
    noShowTicket: 'No-show',
    cancelTicket: 'Cancel',
    returnToWaiting: 'Send back to waiting',
    nextTicketTitle: 'Next in line',
    queueMetricsIssued: 'Issued',
    queueMetricsWaiting: 'Waiting',
    queueMetricsServing: 'Serving',
    queueMetricsDone: 'Done',
    confirmDangerTitle: 'Confirm action',
    dismissConfirm: 'Cancel',
    confirmCancelDescription: 'This ticket will be removed from the active queue.',
    confirmNoShowDescription: 'This ticket will be marked as no-show.',
    closeQueueDescription: 'This queue will stop receiving new customers until you reopen it.',
    statuses: {
      active: 'Active',
      paused: 'Paused',
      closed: 'Closed',
      waiting: 'Waiting',
      called: 'Called',
      serving: 'Serving',
      completed: 'Completed',
      no_show: 'No-show',
      cancelled: 'Cancelled',
    },
  },
  es: {
    title: 'Colas virtuales',
    description: 'Operá las colas activas desde el mismo espacio de agenda.',
    branchLabel: 'Sucursal',
    dateLabel: 'Fecha',
    loading: 'Cargando colas…',
    noQueues: 'No hay colas configuradas para esta sucursal.',
    issueTicketTitle: 'Emitir ticket',
    issueTicketDescription: 'Crear un ticket nuevo de recepción o walk-in.',
    customerNameLabel: 'Cliente',
    customerPhoneLabel: 'Teléfono',
    customerEmailLabel: 'Email',
    priorityLabel: 'Prioridad',
    issueTicket: 'Emitir ticket',
    issuingTicket: 'Emitiendo…',
    callNext: 'Llamar siguiente',
    pauseQueue: 'Pausar cola',
    reopenQueue: 'Reabrir cola',
    closeQueue: 'Cerrar cola',
    waitingColumn: 'Esperando',
    activeColumn: 'En curso',
    finishedColumn: 'Finalizados',
    noTickets: 'No hay tickets en esta columna.',
    requestedAtLabel: 'Solicitado',
    statusLabel: 'Estado',
    serveTicket: 'Atender',
    completeTicket: 'Completar',
    noShowTicket: 'No-show',
    cancelTicket: 'Cancelar',
    returnToWaiting: 'Volver',
    nextTicketTitle: 'Carril de atención',
    queueMetricsIssued: 'Emitidos',
    queueMetricsWaiting: 'Esperando',
    queueMetricsServing: 'Atendiendo',
    queueMetricsDone: 'Resueltos',
    confirmDangerTitle: 'Confirmar acción',
    dismissConfirm: 'Cancelar',
    confirmCancelDescription: 'Este ticket saldrá de la cola activa.',
    confirmNoShowDescription: 'Este ticket se marcará como no-show.',
    closeQueueDescription: 'La cola dejará de recibir más flujo hasta que la reabras.',
    statuses: {
      active: 'Activa',
      paused: 'Pausada',
      closed: 'Cerrada',
      waiting: 'Esperando',
      called: 'Llamado',
      serving: 'Atendiendo',
      completed: 'Completado',
      no_show: 'No-show',
      cancelled: 'Cancelado',
    },
  },
};

export type QueueOperatorBoardProps = {
  client: SchedulingClient;
  locale?: string;
  copy?: Partial<QueueOperatorBoardCopy>;
  searchQuery?: string;
  initialBranchId?: string;
  initialDate?: string;
  className?: string;
};

function toDateInputValue(value: Date): string {
  return value.toISOString().slice(0, 10);
}

function matchesSearch(searchQuery: string, values: Array<string | undefined | null>): boolean {
  if (!searchQuery.trim()) {
    return true;
  }
  const normalized = searchQuery.trim().toLocaleLowerCase();
  return values.some((value) => value?.toLocaleLowerCase().includes(normalized));
}

function parseQueueTicket(item: DayAgendaItem): QueueBoardTicket | null {
  if (item.type !== 'queue_ticket') {
    return null;
  }
  const queueId = typeof item.metadata?.queue_id === 'string' ? item.metadata.queue_id : '';
  if (!queueId) {
    return null;
  }
  return {
    id: item.id,
    queueId,
    label: item.label,
    status: item.status as QueueTicketStatus,
    number: typeof item.metadata?.number === 'number' ? item.metadata.number : undefined,
  };
}

function ticketActions(status: QueueTicketStatus): Array<'serve' | 'complete' | 'no_show' | 'cancel' | 'return'> {
  switch (status) {
    case 'waiting':
      return ['serve', 'no_show', 'cancel'];
    case 'called':
      return ['serve', 'return', 'no_show', 'cancel'];
    case 'serving':
      return ['complete', 'return', 'no_show'];
    default:
      return [];
  }
}

function statusTone(status: QueueStatus | QueueTicketStatus): string {
  switch (status) {
    case 'active':
    case 'serving':
    case 'completed':
      return 'modules-scheduling__badge modules-scheduling__badge--success';
    case 'waiting':
    case 'called':
    case 'paused':
      return 'modules-scheduling__badge modules-scheduling__badge--attention';
    case 'closed':
    case 'cancelled':
    case 'no_show':
      return 'modules-scheduling__badge modules-scheduling__badge--critical';
    default:
      return 'modules-scheduling__badge modules-scheduling__badge--neutral';
  }
}

export function QueueOperatorBoard({
  client,
  locale = 'en',
  copy: copyOverrides,
  searchQuery = '',
  initialBranchId,
  initialDate,
  className = '',
}: QueueOperatorBoardProps) {
  const copy = { ...queueOperatorBoardCopyPresets[resolveSchedulingCopyLocale(locale)], ...copyOverrides };
  const queryClient = useQueryClient();
  const [selectedBranchId, setSelectedBranchId] = useState<string | null>(initialBranchId ?? null);
  const [selectedDate, setSelectedDate] = useState(initialDate ?? toDateInputValue(new Date()));
  const [drafts, setDrafts] = useState<Record<string, TicketDraft>>({});
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const branchesQuery = useQuery<Branch[]>({
    queryKey: queueKeys.branches,
    queryFn: () => client.listBranches(),
    staleTime: 60_000,
  });

  const queuesQuery = useQuery<Queue[]>({
    queryKey: queueKeys.queues(selectedBranchId),
    queryFn: () => client.listQueues(selectedBranchId),
    enabled: Boolean(selectedBranchId),
    staleTime: 15_000,
  });

  const dayAgendaQuery = useQuery<DayAgendaItem[]>({
    queryKey: queueKeys.day(selectedBranchId, selectedDate),
    queryFn: () => client.getDayAgenda(selectedBranchId, selectedDate),
    enabled: Boolean(selectedBranchId),
    staleTime: 10_000,
  });

  useEffect(() => {
    if (selectedBranchId) {
      return;
    }
    const preferred = (branchesQuery.data ?? []).find((branch) => branch.active) ?? branchesQuery.data?.[0];
    if (preferred) {
      setSelectedBranchId(preferred.id);
    }
  }, [branchesQuery.data, selectedBranchId]);

  const invalidateBoard = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queueKeys.queues(selectedBranchId) }),
      queryClient.invalidateQueries({ queryKey: queueKeys.day(selectedBranchId, selectedDate) }),
    ]);
  };

  const issueMutation = useMutation({
    mutationFn: async ({ queueId, draft }: { queueId: string; draft: TicketDraft }) =>
      client.createQueueTicket(queueId, {
        customer_name: draft.customerName.trim(),
        customer_phone: draft.customerPhone.trim(),
        customer_email: draft.customerEmail.trim() || undefined,
        priority: draft.priority,
      }),
    onMutate: () => setErrorMessage(null),
    onSuccess: async (_item, variables) => {
      setDrafts((current) => ({
        ...current,
        [variables.queueId]: { customerName: '', customerPhone: '', customerEmail: '', priority: 0 },
      }));
      await invalidateBoard();
    },
    onError: (error: Error) => setErrorMessage(error.message),
  });

  const queueMutation = useMutation({
    mutationFn: async ({ queueId, action }: { queueId: string; action: 'pause' | 'reopen' | 'close' | 'call-next' }) => {
      switch (action) {
        case 'pause':
          return client.pauseQueue(queueId);
        case 'reopen':
          return client.reopenQueue(queueId);
        case 'close':
          return client.closeQueue(queueId);
        case 'call-next':
          return client.callNext(queueId);
      }
    },
    onMutate: () => setErrorMessage(null),
    onSuccess: async () => {
      await invalidateBoard();
    },
    onError: (error: Error) => setErrorMessage(error.message),
  });

  const ticketMutation = useMutation({
    mutationFn: async ({
      queueId,
      ticketId,
      action,
    }: {
      queueId: string;
      ticketId: string;
      action: 'serve' | 'complete' | 'no_show' | 'cancel' | 'return';
    }) => {
      switch (action) {
        case 'serve':
          return client.serveTicket(queueId, ticketId);
        case 'complete':
          return client.completeTicket(queueId, ticketId);
        case 'no_show':
          return client.markTicketNoShow(queueId, ticketId);
        case 'cancel':
          return client.cancelTicket(queueId, ticketId);
        case 'return':
          return client.returnTicketToWaiting(queueId, ticketId);
      }
    },
    onMutate: () => setErrorMessage(null),
    onSuccess: async () => {
      await invalidateBoard();
    },
    onError: (error: Error) => setErrorMessage(error.message),
  });

  const queues = useMemo(() => {
    const allQueues = queuesQuery.data ?? [];
    return allQueues.filter((queue) => matchesSearch(searchQuery, [queue.name, queue.code]));
  }, [queuesQuery.data, searchQuery]);

  const queueTickets = useMemo(() => {
    const items = (dayAgendaQuery.data ?? [])
      .map(parseQueueTicket)
      .filter((item): item is QueueBoardTicket => item !== null);
    return items.filter((item) => matchesSearch(searchQuery, [item.label, String(item.number ?? '')]));
  }, [dayAgendaQuery.data, searchQuery]);

  const groupedTickets = useMemo(() => {
    const map = new Map<string, QueueBoardTicket[]>();
    for (const ticket of queueTickets) {
      const current = map.get(ticket.queueId) ?? [];
      current.push(ticket);
      map.set(ticket.queueId, current);
    }
    return map;
  }, [queueTickets]);

  const handleQueueAction = async (queue: Queue, action: 'pause' | 'reopen' | 'close' | 'call-next') => {
    if (action === 'close') {
      const confirmed = await confirmAction({
        title: copy.confirmDangerTitle,
        description: copy.closeQueueDescription,
        confirmLabel: copy.closeQueue,
        cancelLabel: copy.dismissConfirm,
        tone: 'danger',
      });
      if (!confirmed) {
        return;
      }
    }
    await queueMutation.mutateAsync({ queueId: queue.id, action });
  };

  const handleTicketAction = async (
    queueId: string,
    ticketId: string,
    action: 'serve' | 'complete' | 'no_show' | 'cancel' | 'return',
  ) => {
    if (action === 'cancel' || action === 'no_show') {
      const confirmed = await confirmAction({
        title: copy.confirmDangerTitle,
        description: action === 'cancel' ? copy.confirmCancelDescription : copy.confirmNoShowDescription,
        confirmLabel: action === 'cancel' ? copy.cancelTicket : copy.noShowTicket,
        cancelLabel: copy.dismissConfirm,
        tone: 'danger',
      });
      if (!confirmed) {
        return;
      }
    }
    await ticketMutation.mutateAsync({ queueId, ticketId, action });
  };

  const loading = branchesQuery.isLoading || queuesQuery.isLoading || dayAgendaQuery.isLoading;

  return (
    <section className={`modules-scheduling ${className}`.trim()}>
      {errorMessage ? <div className="alert alert-error">{errorMessage}</div> : null}

      <div className="card">
        <div className="card-header">
          <div>
            <h2>{copy.title}</h2>
            <p className="text-secondary">{copy.description}</p>
          </div>
        </div>
        <div className="modules-scheduling__filters">
          <div className="form-group grow">
            <label htmlFor="queue-board-branch">{copy.branchLabel}</label>
            <select
              id="queue-board-branch"
              value={selectedBranchId ?? ''}
              onChange={(event) => setSelectedBranchId(event.target.value || null)}
            >
              {(branchesQuery.data ?? []).map((branch) => (
                <option key={branch.id} value={branch.id}>
                  {branch.name}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label htmlFor="queue-board-date">{copy.dateLabel}</label>
            <input
              id="queue-board-date"
              type="date"
              value={selectedDate}
              onChange={(event) => setSelectedDate(event.target.value)}
            />
          </div>
        </div>
      </div>

      {loading ? (
        <div className="card modules-scheduling__empty">
          <div className="spinner" />
          <p>{copy.loading}</p>
        </div>
      ) : queues.length === 0 ? (
        <div className="card empty-state">
          <p>{copy.noQueues}</p>
        </div>
      ) : (
        <div className="modules-scheduling__queue-grid">
          {queues.map((queue) => {
            const items = groupedTickets.get(queue.id) ?? [];
            const waiting = items.filter((item) => item.status === 'waiting');
            const active = items.filter((item) => item.status === 'called' || item.status === 'serving');
            const finished = items.filter(
              (item) => item.status === 'completed' || item.status === 'cancelled' || item.status === 'no_show',
            );
            const draft = drafts[queue.id] ?? { customerName: '', customerPhone: '', customerEmail: '', priority: 0 };

            return (
              <article key={queue.id} className="card modules-scheduling__queue-card">
                <div className="modules-scheduling__queue-header">
                  <div>
                    <h3 className="text-section-title">
                      {queue.name} <span className="text-muted">({queue.code})</span>
                    </h3>
                    <div className="modules-scheduling__queue-meta">
                      <span className={statusTone(queue.status)}>{copy.statuses[queue.status]}</span>
                      <span>
                        {copy.queueMetricsWaiting}: <strong>{waiting.length}</strong>
                      </span>
                      <span>
                        {copy.queueMetricsServing}: <strong>{active.length}</strong>
                      </span>
                      <span>
                        {copy.queueMetricsDone}: <strong>{finished.length}</strong>
                      </span>
                    </div>
                  </div>
                  <div className="modules-scheduling__queue-actions">
                    <button
                      type="button"
                      className="btn-primary btn-sm"
                      onClick={() => void handleQueueAction(queue, 'call-next')}
                      disabled={queue.status !== 'active' || queueMutation.isPending}
                    >
                      {copy.callNext}
                    </button>
                    {queue.status === 'active' ? (
                      <button
                        type="button"
                        className="btn-secondary btn-sm"
                        onClick={() => void handleQueueAction(queue, 'pause')}
                        disabled={queueMutation.isPending}
                      >
                        {copy.pauseQueue}
                      </button>
                    ) : null}
                    {queue.status === 'paused' ? (
                      <button
                        type="button"
                        className="btn-secondary btn-sm"
                        onClick={() => void handleQueueAction(queue, 'reopen')}
                        disabled={queueMutation.isPending}
                      >
                        {copy.reopenQueue}
                      </button>
                    ) : null}
                    {queue.status !== 'closed' ? (
                      <button
                        type="button"
                        className="btn-danger btn-sm"
                        onClick={() => void handleQueueAction(queue, 'close')}
                        disabled={queueMutation.isPending}
                      >
                        {copy.closeQueue}
                      </button>
                    ) : null}
                  </div>
                </div>

                <div className="modules-scheduling__queue-ticket-form">
                  <div className="modules-scheduling__queue-ticket-form-header">
                    <strong>{copy.issueTicketTitle}</strong>
                    <span>{copy.issueTicketDescription}</span>
                  </div>
                  <div className="modules-scheduling__queue-ticket-form-grid">
                    <div className="form-group grow">
                      <label htmlFor={`queue-name-${queue.id}`}>{copy.customerNameLabel}</label>
                      <input
                        id={`queue-name-${queue.id}`}
                        value={draft.customerName}
                        onChange={(event) =>
                          setDrafts((current) => ({
                            ...current,
                            [queue.id]: { ...draft, customerName: event.target.value },
                          }))
                        }
                      />
                    </div>
                    <div className="form-group grow">
                      <label htmlFor={`queue-phone-${queue.id}`}>{copy.customerPhoneLabel}</label>
                      <input
                        id={`queue-phone-${queue.id}`}
                        value={draft.customerPhone}
                        onChange={(event) =>
                          setDrafts((current) => ({
                            ...current,
                            [queue.id]: { ...draft, customerPhone: event.target.value },
                          }))
                        }
                      />
                    </div>
                    <div className="form-group grow">
                      <label htmlFor={`queue-email-${queue.id}`}>{copy.customerEmailLabel}</label>
                      <input
                        id={`queue-email-${queue.id}`}
                        value={draft.customerEmail}
                        onChange={(event) =>
                          setDrafts((current) => ({
                            ...current,
                            [queue.id]: { ...draft, customerEmail: event.target.value },
                          }))
                        }
                      />
                    </div>
                    <div className="form-group">
                      <label htmlFor={`queue-priority-${queue.id}`}>{copy.priorityLabel}</label>
                      <input
                        id={`queue-priority-${queue.id}`}
                        type="number"
                        min={0}
                        max={9}
                        value={draft.priority}
                        onChange={(event) =>
                          setDrafts((current) => ({
                            ...current,
                            [queue.id]: { ...draft, priority: Number(event.target.value || 0) },
                          }))
                        }
                      />
                    </div>
                  </div>
                  <button
                    type="button"
                    className="btn-secondary btn-sm"
                    disabled={!draft.customerName.trim() || !draft.customerPhone.trim() || issueMutation.isPending || queue.status === 'closed'}
                    onClick={() => void issueMutation.mutateAsync({ queueId: queue.id, draft })}
                  >
                    {issueMutation.isPending ? copy.issuingTicket : copy.issueTicket}
                  </button>
                </div>

                <div className="modules-scheduling__queue-columns">
                  {[
                    { title: copy.waitingColumn, items: waiting },
                    { title: copy.activeColumn, items: active },
                    { title: copy.finishedColumn, items: finished },
                  ].map((column) => (
                    <section key={column.title} className="modules-scheduling__queue-column">
                      <div className="modules-scheduling__queue-column-title">{column.title}</div>
                      {column.items.length === 0 ? (
                        <div className="modules-scheduling__queue-empty">{copy.noTickets}</div>
                      ) : (
                        column.items.map((ticket) => (
                          <div key={ticket.id} className="modules-scheduling__queue-ticket">
                            <div className="modules-scheduling__queue-ticket-main">
                              <strong>{ticket.label}</strong>
                              <span className={statusTone(ticket.status)}>{copy.statuses[ticket.status]}</span>
                            </div>
                            <div className="modules-scheduling__queue-ticket-meta">
                              <span>
                                #{ticket.number ?? '—'} · {copy.statusLabel}: {copy.statuses[ticket.status]}
                              </span>
                            </div>
                            {ticketActions(ticket.status).length > 0 ? (
                              <div className="modules-scheduling__queue-ticket-actions">
                                {ticketActions(ticket.status).map((action) => (
                                  <button
                                    key={action}
                                    type="button"
                                    className={
                                      action === 'cancel' || action === 'no_show'
                                        ? 'btn-danger btn-sm'
                                        : 'btn-secondary btn-sm'
                                    }
                                    onClick={() => void handleTicketAction(queue.id, ticket.id, action)}
                                    disabled={ticketMutation.isPending}
                                  >
                                    {action === 'serve'
                                      ? copy.serveTicket
                                      : action === 'complete'
                                        ? copy.completeTicket
                                        : action === 'no_show'
                                          ? copy.noShowTicket
                                          : action === 'cancel'
                                            ? copy.cancelTicket
                                            : copy.returnToWaiting}
                                  </button>
                                ))}
                              </div>
                            ) : null}
                          </div>
                        ))
                      )}
                    </section>
                  ))}
                </div>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}
