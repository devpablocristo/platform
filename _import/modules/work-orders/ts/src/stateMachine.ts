/**
 * Máquina de estados para órdenes de trabajo.
 * Usa la primitiva FSM de core. Define las transiciones default
 * que aplican a cualquier vertical (auto-repair, bike-shop, etc.).
 */
import { Builder, type StringMachine } from '@devpablocristo/core-fsm';

/**
 * FSM default para órdenes de trabajo.
 *
 * Grafo:
 *   received ↔ diagnosing ↔ in_progress ↔ quality_check ↔ on_hold  (grupo libre)
 *   quote_pending ↔ awaiting_parts  (grupo libre)
 *   received/diagnosing → quote_pending  (explícita)
 *   quote_pending/awaiting_parts → in_progress  (explícita)
 *   in_progress/quality_check → ready_for_pickup  (explícita)
 *   ready_for_pickup → delivered  (explícita)
 *   delivered → invoiced  (explícita)
 *   cualquier no-terminal → cancelled  (global)
 *   invoiced, cancelled = terminales
 */
export function buildWorkOrderFSM(): StringMachine {
  return new Builder()
    .terminal('invoiced', 'cancelled')
    .freeTransitionsAmong('received', 'diagnosing', 'in_progress', 'quality_check', 'on_hold')
    .freeTransitionsAmong('quote_pending', 'awaiting_parts')
    // Ingreso → Presupuesto
    .allowFromStatesTo('quote_pending', 'received', 'diagnosing')
    // Presupuesto → Taller
    .allowFromStatesTo('in_progress', 'quote_pending', 'awaiting_parts')
    // Taller → Salida
    .allowFromStatesTo('ready_for_pickup', 'in_progress', 'quality_check')
    // Salida → Entrega → Facturado
    .allow('ready_for_pickup', 'delivered')
    .allow('delivered', 'invoiced')
    // Cancelar desde cualquier no-terminal
    .allowAnyTo('cancelled')
    .build();
}

/** Instancia singleton — inmutable, segura para compartir. */
export const workOrderFSM = buildWorkOrderFSM();
