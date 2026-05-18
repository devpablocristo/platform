import type { ReactNode } from "react";
import { SearchInput } from "../search";

/**
 * Layout canónico de consola: cabecera (título + acciones), error, formulario,
 * barra de herramientas (búsqueda / filtros) y cuerpo (spinner, vacío o tabla).
 * Productos (Pymes u otros) montan datos y i18n encima; las clases CSS viven en el tema de la app.
 * El host puede combinar `data-theme` y, si aplica, `data-admin-skin` en `<html>` para tokens adicionales.
 */
export type CrudPageShellProps = {
  title: ReactNode;
  subtitle?: ReactNode;
  /** Bajo el subtítulo, columna izquierda (p. ej. filtros tipo píldora). */
  headerLeadSlot?: ReactNode;
  /** Buscador canónico del header. */
  search?: {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    ariaLabel?: string;
    inputClassName?: string;
    clearLabel?: string;
  };
  /** Fila de acciones bajo el buscador compartido. */
  headerActions?: ReactNode;
  error?: ReactNode;
  /** Tarjeta de formulario alta/edición */
  form?: ReactNode;
  /** Fila búsqueda + archivados u otros filtros */
  toolbar?: ReactNode;
  children: ReactNode;
};

export function CrudPageShell({
  title,
  subtitle,
  headerLeadSlot,
  search,
  headerActions,
  error,
  form,
  toolbar,
  children,
}: CrudPageShellProps) {
  return (
    <>
      <div className="page-header crud-page-shell__header">
        <div className="crud-page-shell__header-main">
          <h1 className="crud-page-shell__title">{title}</h1>
          {subtitle != null && subtitle !== false ? (
            <p className="text-secondary">{subtitle}</p>
          ) : null}
          {headerLeadSlot != null ? headerLeadSlot : null}
        </div>
        {search != null || headerActions != null ? (
          <div className="crud-page-shell__header-actions">
            {search != null ? (
              <div className="crud-list-header-search">
                <SearchInput
                  value={search.value}
                  onChange={search.onChange}
                  placeholder={search.placeholder ?? "Buscar..."}
                  ariaLabel={search.ariaLabel}
                  inputClassName={search.inputClassName}
                  clearLabel={search.clearLabel}
                />
              </div>
            ) : null}
            {headerActions != null ? <div className="actions-row">{headerActions}</div> : null}
          </div>
        ) : null}
      </div>
      {error}
      {form}
      {toolbar}
      {children}
    </>
  );
}
