// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SchedulingBookingModal, type SchedulingBookingModalState } from './SchedulingBookingModal';
import type { SchedulingCalendarCopy } from './types';

const confirmActionMock = vi.hoisted(() => vi.fn(async () => true));

vi.mock('@devpablocristo/core-browser', () => ({
  confirmAction: confirmActionMock,
}));

const copy = {
  branchLabel: 'Sucursal',
  serviceLabel: 'Servicio',
  resourceLabel: 'Recurso',
  anyResource: 'Cualquier recurso',
  focusDateLabel: 'Fecha foco',
  today: 'Hoy',
  slotsTitle: 'Slots disponibles',
  slotsDescription: 'Descripción',
  slotsLoading: 'Cargando',
  slotsEmpty: 'Vacío',
  slotRemainingLabel: 'Cupos',
  openBooking: 'Abrir reserva',
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
  bookingTitleDetails: 'Detalle de reserva',
  bookingSubtitleCreate: 'Nueva reserva',
  bookingSubtitleDetails: 'Reserva existente',
  availableSlotLabel: 'Slot disponible',
  availableSlotHint: 'Usa un slot valido',
  availableSlotLoading: 'Buscando disponibilidad',
  unavailableSlotMessage: 'No disponible',
  slotSummaryTitle: 'Resumen del slot',
  bookingPreviewTitle: 'Vista previa',
  customerNameLabel: 'Cliente',
  customerPhoneLabel: 'Teléfono',
  customerEmailLabel: 'Email',
  notesLabel: 'Notas',
  serviceNameLabel: 'Servicio',
  resourceNameLabel: 'Recurso',
  slotLabel: 'Horario',
  slotStartLabel: 'Inicio',
  slotEndLabel: 'Fin',
  durationLabel: 'Duración',
  timezoneLabel: 'Zona horaria',
  occupiesLabel: 'Bloquea',
  conflictLabel: 'Conflictos',
  referenceLabel: 'Referencia',
  statusLabel: 'Estado',
  create: 'Guardar',
  saving: 'Guardando',
  close: 'Cancelar',
  confirmBooking: 'Confirmar',
  cancelBooking: 'Cancelar reserva',
  checkInBooking: 'Check-in',
  startService: 'Iniciar',
  completeBooking: 'Completar',
  noShowBooking: 'No asistió',
  dragRescheduleTitle: 'Reprogramar',
  dragRescheduleDescription: 'Mover turno',
  rescheduleBooking: 'Reprogramar reserva',
  destructiveTitle: 'Acción destructiva',
  cancelActionDescription: 'Cancelar turno',
  noShowActionDescription: 'Marcar no show',
  closeDirtyTitle: 'Descartar borrador',
  closeDirtyDescription: 'Hay datos cargados sin guardar en esta reserva.',
  keepEditing: 'Seguir editando',
  discard: 'Descartar',
  resizeLockedMessage: 'Duración fija',
  searchPlaceholder: 'Buscar',
  dashboardBookings: 'Reservas',
  dashboardConfirmed: 'Confirmadas',
  dashboardQueues: 'Colas',
  dashboardWaiting: 'Esperando',
  timelineTitle: 'Agenda',
  timelineDescription: 'Descripción',
  queueTitle: 'Colas',
  queueDescription: 'Descripción',
  queueSearchLabel: 'Buscar cola',
  queueSearchPlaceholder: 'Buscar',
  queueEmpty: 'Vacío',
  queueCreateTicket: 'Crear ticket',
  queuePause: 'Pausar',
  queueResume: 'Reanudar',
  queueClose: 'Cerrar',
  queueCallNext: 'Llamar siguiente',
  queuePriorityLabel: 'Prioridad',
  queueCustomerNameLabel: 'Cliente',
  queueCustomerPhoneLabel: 'Teléfono',
  queueCustomerEmailLabel: 'Email',
  statuses: {
    hold: 'Hold',
    pending_confirmation: 'Pendiente',
    confirmed: 'Confirmada',
    checked_in: 'Check-in',
    in_service: 'En servicio',
    completed: 'Completada',
    cancelled: 'Cancelada',
    no_show: 'No asistió',
    expired: 'Expirada',
  },
} as unknown as SchedulingCalendarCopy;

const createState: SchedulingBookingModalState = {
  open: true,
  mode: 'create',
  slot: {
    resource_id: 'resource-1',
    resource_name: 'Demo Professional',
    start_at: '2099-04-05T10:00:00Z',
    end_at: '2099-04-05T10:30:00Z',
    occupies_from: '2099-04-05T10:00:00Z',
    occupies_until: '2099-04-05T10:30:00Z',
    timezone: 'America/Argentina/Tucuman',
    remaining: 1,
    conflict_count: 0,
    granularity_minutes: 30,
  },
  slotOptions: [
    {
      resource_id: 'resource-1',
      resource_name: 'Demo Professional',
      start_at: '2099-04-05T10:00:00Z',
      end_at: '2099-04-05T10:30:00Z',
      occupies_from: '2099-04-05T10:00:00Z',
      occupies_until: '2099-04-05T10:30:00Z',
      timezone: 'America/Argentina/Tucuman',
      remaining: 1,
      conflict_count: 0,
      granularity_minutes: 30,
    },
  ],
  resourceOptions: [
    {
      id: 'resource-1',
      name: 'Demo Professional',
      timezone: 'America/Argentina/Tucuman',
    },
  ],
  editor: {
    date: '2099-04-05',
    startTime: '10:00',
    endTime: '10:30',
    resourceId: 'resource-1',
  },
  validationMessage: null,
  draft: {
    title: '',
    customerName: '',
    customerPhone: '',
    customerEmail: '',
    notes: '',
    recurrence: {
      mode: 'none',
      frequency: 'weekly',
      interval: '1',
      count: '8',
      byWeekday: [0],
    },
  },
};

describe('SchedulingBookingModal', () => {
  beforeEach(() => {
    cleanup();
  });

  it('muestra confirmación al cerrar con Escape si hay cambios sin guardar', async () => {
    const onClose = vi.fn();
    confirmActionMock.mockResolvedValue(false);

    render(
      <SchedulingBookingModal
        state={createState}
        copy={copy}
        onClose={onClose}
        onEditorChange={vi.fn()}
        onCreate={vi.fn()}
        onAction={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText('Cliente'), { target: { value: 'Cliente editado' } });
    fireEvent.keyDown(window, { key: 'Escape' });

    await waitFor(() => {
      expect(confirmActionMock).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Descartar borrador',
          confirmLabel: 'Descartar',
          cancelLabel: 'Seguir editando',
        }),
      );
    });

    expect(onClose).not.toHaveBeenCalled();
  });

  it('muestra solo el formulario editable del creador', async () => {
    render(
      <SchedulingBookingModal
        state={createState}
        copy={copy}
        onClose={vi.fn()}
        onEditorChange={vi.fn()}
        onCreate={vi.fn()}
        onAction={vi.fn()}
      />,
    );

    expect(screen.queryByText('Resumen del slot')).toBeNull();
    expect(screen.queryByText('Vista previa')).toBeNull();
    expect(screen.queryByText('Nueva reserva')).toBeNull();
    expect(screen.queryByText('Crear reserva')).toBeNull();
    expect(screen.getByLabelText('Fecha foco')).toBeTruthy();
    expect(screen.getByLabelText('Inicio')).toBeTruthy();
    expect(screen.getByLabelText('Fin')).toBeTruthy();
    expect(screen.getByLabelText('Recurso')).toBeTruthy();
    expect(screen.getByLabelText('Título')).toBeTruthy();
    expect(screen.getByLabelText('Repetir')).toBeTruthy();

    fireEvent.change(screen.getByLabelText('Cliente'), { target: { value: 'Ada Lovelace' } });

    await waitFor(() => {
      expect(screen.getByDisplayValue('Ada Lovelace')).toBeTruthy();
    });
  });

  it('con locale es muestra Fecha foco como texto día/mes/año', () => {
    render(
      <SchedulingBookingModal
        state={createState}
        copy={copy}
        locale="es-AR"
        onClose={vi.fn()}
        onEditorChange={vi.fn()}
        onCreate={vi.fn()}
        onAction={vi.fn()}
      />,
    );

    const dateInput = screen.getByLabelText('Fecha foco') as HTMLInputElement;
    expect(dateInput.type).toBe('text');
    expect(dateInput.placeholder).toBe('DD/MM/AAAA');
    expect(dateInput.value).toMatch(/\d{1,2}\/\d{1,2}\/2099/);
    expect(dateInput.value).not.toMatch(/2099-04-05/);
  });
});
