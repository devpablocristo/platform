import type { SchedulingCalendarCopy, SchedulingEntryType } from './types';

// Switcher de tipo de entrada (Evento / Turno / Bloqueo) que vive embebido en
// el header de los 3 modales del calendario interno. Reemplaza al chooser modal
// previo: el dueño abre directo el modal por default (evento) y desde acá puede
// cambiar a turno o bloqueo sin perder el contexto de fecha/hora del click.
//
// Reusable: las copy strings (`newEntryMenu*`) ya viven en SchedulingCalendarCopy
// y son las mismas que usaba el chooser. El styling reusa btn-primary/btn-secondary
// del design system de la app, no inventamos clases nuevas.

type Props = {
  active: SchedulingEntryType;
  copy: SchedulingCalendarCopy;
  bookingEnabled?: boolean; // false cuando no hay servicio seleccionado
  onSwitch: (type: SchedulingEntryType) => void;
};

const ORDER: SchedulingEntryType[] = ['internal_event', 'booking', 'blocked_range'];

function labelFor(type: SchedulingEntryType, copy: SchedulingCalendarCopy): string {
  switch (type) {
    case 'internal_event':
      return copy.newEntryMenuInternalEvent;
    case 'booking':
      return copy.newEntryMenuBooking;
    case 'blocked_range':
      return copy.newEntryMenuBlockedRange;
  }
}

export function SchedulingEntryTypeSwitcher({ active, copy, bookingEnabled = true, onSwitch }: Props) {
  return (
    <div
      className="modules-scheduling__entry-switcher"
      role="tablist"
      aria-label={copy.newEntryButton}
      style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', marginBottom: '0.75rem' }}
    >
      {ORDER.map((type) => {
        const isActive = type === active;
        const disabled = type === 'booking' && !bookingEnabled;
        return (
          <button
            key={type}
            type="button"
            role="tab"
            aria-selected={isActive}
            className={`btn-sm ${isActive ? 'btn-primary' : 'btn-secondary'}`}
            disabled={disabled || isActive}
            onClick={() => onSwitch(type)}
            data-testid={`scheduling-entry-switch-${type}`}
          >
            {labelFor(type, copy)}
          </button>
        );
      })}
    </div>
  );
}
