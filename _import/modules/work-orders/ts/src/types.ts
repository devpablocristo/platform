/**
 * Tipos genéricos para órdenes de trabajo.
 * Aplica a cualquier vertical (auto repair, bike shop, etc.).
 */

export type WorkOrderItem = {
  id?: string;
  item_type: 'service' | 'part';
  service_id?: string;
  product_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  tax_rate: number;
  sort_order?: number;
  metadata?: Record<string, unknown>;
};

export type WorkOrder = {
  id: string;
  org_id: string;
  number: string;
  /** Identificador del activo (vehículo, bicicleta, etc.) */
  asset_id?: string;
  /** Etiqueta visible del activo (patente, cuadro, etc.) */
  asset_label: string;
  customer_id?: string;
  customer_name: string;
  booking_id?: string;
  quote_id?: string;
  sale_id?: string;
  status: string;
  requested_work: string;
  diagnosis: string;
  notes: string;
  internal_notes: string;
  currency: string;
  subtotal_services: number;
  subtotal_parts: number;
  tax_total: number;
  total: number;
  opened_at: string;
  promised_at?: string;
  ready_at?: string;
  delivered_at?: string;
  created_by: string;
  archived_at?: string | null;
  created_at: string;
  updated_at: string;
  items: WorkOrderItem[];
};

/** Funciones API que el consumidor debe proveer. */
export type WorkOrderClient = {
  listAll: (opts?: { archived?: boolean }) => Promise<WorkOrder[]>;
  get: (id: string) => Promise<WorkOrder>;
  patch: (id: string, data: Partial<Record<string, unknown>>) => Promise<WorkOrder>;
  archive: (id: string) => Promise<void>;
  restore: (id: string) => Promise<void>;
};

export type KanbanPhase = string;

export type KanbanColumnDef = {
  id: KanbanPhase;
  label: string;
};

export type WorkOrderKanbanConfig = {
  columns: KanbanColumnDef[];
  /** Mapea un status canónico a su fase kanban. */
  statusToPhase: (status: string) => KanbanPhase;
  /** Status default al soltar una card en una fase. null = no permitir drop. */
  defaultStatusForPhase: (phase: KanbanPhase) => string | null;
  /** Es un status terminal (no draggable)? */
  isTerminal: (status: string) => boolean;
  /** Normaliza status raw del backend. */
  canonicalStatus: (raw: string) => string;
  /** Label para badge de status. */
  statusLabel: (status: string) => string;
  /** Valida si una transición de estado es permitida. */
  canTransition: (from: string, to: string) => boolean;
};
