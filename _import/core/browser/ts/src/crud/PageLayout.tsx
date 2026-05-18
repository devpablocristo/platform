import type { ReactNode } from 'react';
import { CrudPageShell } from './CrudPageShell';
import { usePageSearchShellControl } from '../search/PageSearchProvider';

export type PageLayoutProps = {
  /** Título principal (h1) */
  title: ReactNode;
  /** Párrafo lead bajo el título (opcional) */
  lead?: ReactNode;
  /** Botones / enlaces a la derecha; activa layout tipo split automáticamente */
  actions?: ReactNode;
  /** Contenido entre cabecera y body (alertas, avisos) */
  banner?: ReactNode;
  /** Clases extra en el contenedor (e.g. 'dash', 'gcal') */
  className?: string;
  /** Label del botón clear de búsqueda (default: 'Clear search') */
  searchClearLabel?: string;
  children: ReactNode;
};

function isPrimitiveLead(lead: ReactNode) {
  return typeof lead === 'string' || typeof lead === 'number';
}

/**
 * Layout estándar de página: wrapper sobre CrudPageShell con integración de PageSearch.
 */
export function PageLayout({ title, lead, actions, banner, className, searchClearLabel, children }: PageLayoutProps) {
  const stackClass = ['page-stack', className].filter(Boolean).join(' ');
  const pageSearch = usePageSearchShellControl();
  const hasSearch = pageSearch.visible;
  const primitiveLead = lead != null && lead !== false && isPrimitiveLead(lead) ? lead : undefined;
  const richLead =
    lead != null && lead !== false && !isPrimitiveLead(lead) ? <div className="text-page-lead">{lead}</div> : undefined;
  return (
    <div className={stackClass}>
      <CrudPageShell
        title={title}
        subtitle={primitiveLead}
        headerLeadSlot={richLead}
        search={
          hasSearch
            ? {
                value: pageSearch.query,
                onChange: pageSearch.setQuery,
                placeholder: pageSearch.placeholder,
                clearLabel: searchClearLabel ?? 'Clear search',
              }
            : undefined
        }
        headerActions={actions}
      >
        <>
          {banner}
          {children}
        </>
      </CrudPageShell>
    </div>
  );
}
