/**
 * Configuración kanban default para órdenes de trabajo.
 * Cada vertical puede sobreescribir con su propia config.
 */
import type { KanbanColumnDef, KanbanPhase, WorkOrderKanbanConfig } from './types';
import { workOrderFSM } from './stateMachine';

const DEFAULT_COLUMNS: KanbanColumnDef[] = [
  { id: 'wo_intake', label: 'Ingreso' },
  { id: 'wo_quote', label: 'Presupuesto / repuestos' },
  { id: 'wo_shop', label: 'Taller' },
  { id: 'wo_exit', label: 'Salida' },
  { id: 'wo_closed', label: 'Cerradas' },
];

function canonicalStatus(raw: string): string {
  const s = (raw || '').toLowerCase().trim();
  switch (s) {
    case '':
    case 'received':
      return 'received';
    case 'diagnosis':
    case 'diagnosing':
      return 'diagnosing';
    case 'quote_pending':
      return 'quote_pending';
    case 'awaiting_parts':
      return 'awaiting_parts';
    case 'in_progress':
      return 'in_progress';
    case 'quality_check':
      return 'quality_check';
    case 'ready':
    case 'ready_for_pickup':
      return 'ready_for_pickup';
    case 'delivered':
      return 'delivered';
    case 'invoiced':
      return 'invoiced';
    case 'cancelled':
      return 'cancelled';
    case 'on_hold':
      return 'on_hold';
    default:
      return 'received';
  }
}

function statusToPhase(raw: string): KanbanPhase {
  const s = canonicalStatus(raw);
  switch (s) {
    case 'received':
    case 'diagnosing':
      return 'wo_intake';
    case 'quote_pending':
    case 'awaiting_parts':
      return 'wo_quote';
    case 'in_progress':
    case 'quality_check':
    case 'on_hold':
      return 'wo_shop';
    case 'ready_for_pickup':
    case 'delivered':
      return 'wo_exit';
    case 'invoiced':
    case 'cancelled':
      return 'wo_closed';
    default:
      return 'wo_intake';
  }
}

function defaultStatusForPhase(phase: KanbanPhase): string | null {
  switch (phase) {
    case 'wo_intake':
      return 'received';
    case 'wo_quote':
      return 'quote_pending';
    case 'wo_shop':
      return 'in_progress';
    case 'wo_exit':
      return 'ready_for_pickup';
    case 'wo_closed':
      return null;
    default:
      return 'received';
  }
}

function isTerminal(raw: string): boolean {
  return workOrderFSM.isTerminal(canonicalStatus(raw));
}

const STATUS_LABELS: Record<string, string> = {
  received: 'Recibido',
  diagnosing: 'Diagnóstico',
  quote_pending: 'Presupuesto',
  awaiting_parts: 'Repuestos',
  in_progress: 'En taller',
  quality_check: 'Control',
  ready_for_pickup: 'Listo retiro',
  delivered: 'Entregado',
  on_hold: 'En pausa',
  invoiced: 'Facturado',
  cancelled: 'Cancelado',
};

function statusLabel(raw: string): string {
  const s = canonicalStatus(raw);
  return STATUS_LABELS[s] ?? s;
}

function canTransition(from: string, to: string): boolean {
  return workOrderFSM.canTransition(canonicalStatus(from), canonicalStatus(to));
}

export const defaultKanbanConfig: WorkOrderKanbanConfig = {
  columns: DEFAULT_COLUMNS,
  statusToPhase,
  defaultStatusForPhase,
  isTerminal,
  canonicalStatus,
  statusLabel,
  canTransition,
};

/**
 * Crea una config custom sobreescribiendo solo lo necesario.
 */
export function createKanbanConfig(overrides: Partial<WorkOrderKanbanConfig> = {}): WorkOrderKanbanConfig {
  return { ...defaultKanbanConfig, ...overrides };
}
