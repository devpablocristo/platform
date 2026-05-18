import { useEffect, useMemo, useRef, useState, type PropsWithChildren, type ReactNode } from 'react';
import { search, type SearchEntry } from '@devpablocristo/core-browser/search';

export type AppShellNavItem = {
  to: string;
  label: string;
  icon: ReactNode;
  end?: boolean;
  isActive?: (pathname: string, search: string) => boolean;
};

export type AppShellNavSection = {
  label: string;
  items: AppShellNavItem[];
};

export type AppShellProps = PropsWithChildren<{
  brandTitle: string;
  brandSubtitle: string;
  brandIcon?: ReactNode;
  sections: AppShellNavSection[];
  footerContent?: ReactNode;
  /** Renderiza cada link de navegación. Recibe el item y si está activo según isActive custom. */
  renderLink: (item: AppShellNavItem, className: string) => ReactNode;
  /** Formatea labels (ej. sentenceCase). Default: identidad. */
  formatLabel?: (label: string) => string;
  /** Pathname actual para scroll-to-top. */
  pathname?: string;
  /** Placeholder del buscador. Default: "Buscar..." */
  searchPlaceholder?: string;
}>;

type FlatEntry = SearchEntry<{ sectionLabel: string; item: AppShellNavItem }>;

export function AppShell({
  children,
  brandTitle,
  brandSubtitle,
  brandIcon,
  sections,
  footerContent,
  renderLink,
  formatLabel = (s) => s,
  pathname,
  searchPlaceholder = 'Buscar...',
}: AppShellProps) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const main = document.querySelector<HTMLElement>('.main-content');
    main?.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }, [pathname]);

  // Ctrl+K / Cmd+K para enfocar el buscador
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
      }
      if (e.key === 'Escape' && document.activeElement === inputRef.current) {
        setQuery('');
        inputRef.current?.blur();
      }
    }
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, []);

  // Índice plano para búsqueda
  const flatEntries = useMemo<FlatEntry[]>(
    () =>
      sections.flatMap((s) =>
        s.items.map((item) => ({
          item: { sectionLabel: s.label, item },
          text: `${s.label} ${item.label}`,
        })),
      ),
    [sections],
  );

  // Secciones filtradas por trigramas
  const filteredSections = useMemo(() => {
    const q = query.trim();
    if (q.length === 0) return sections;

    const matches = search(q, flatEntries);
    const matchedItems = new Set(matches.map((m) => m.item.item));

    return sections
      .map((s) => ({
        ...s,
        items: s.items.filter((item) => matchedItems.has(item)),
      }))
      .filter((s) => s.items.length > 0);
  }, [query, sections, flatEntries]);

  return (
    <div className={`app-layout${sidebarOpen ? '' : ' sidebar-collapsed'}`}>
      <aside className="sidebar">
        <div
          className="sidebar-brand"
          onClick={() => setSidebarOpen((v) => !v)}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && setSidebarOpen((v) => !v)}
        >
          <div className="sidebar-brand-row">
            {brandIcon && <span className="sidebar-brand-icon">{brandIcon}</span>}
            <h1>{brandTitle}</h1>
            <span className="sidebar-brand-arrow">{sidebarOpen ? '‹' : '›'}</span>
          </div>
          <small>{formatLabel(brandSubtitle)}</small>
        </div>

        {sidebarOpen && (
          <div className="sidebar-search">
            <input
              ref={inputRef}
              type="text"
              className="sidebar-search-input"
              placeholder={searchPlaceholder}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              aria-label={searchPlaceholder}
            />
            {query.length > 0 && (
              <button
                className="sidebar-search-clear"
                onClick={() => { setQuery(''); inputRef.current?.focus(); }}
                aria-label="Limpiar búsqueda"
                type="button"
              >
                ×
              </button>
            )}
          </div>
        )}

        <nav className="sidebar-nav">
          {filteredSections.map((section) => (
            <NavSection
              key={section.label}
              label={section.label}
              items={section.items}
              renderLink={renderLink}
              formatLabel={formatLabel}
              forceOpen={query.trim().length > 0}
            />
          ))}
          {query.trim().length > 0 && filteredSections.length === 0 && (
            <div className="sidebar-search-empty">Sin resultados</div>
          )}
        </nav>

        <div className="sidebar-footer">
          {footerContent ?? null}
        </div>
      </aside>

      <main className="main-content">{children}</main>
    </div>
  );
}

function NavSection({
  label,
  items,
  renderLink,
  formatLabel,
  forceOpen,
}: AppShellNavSection & {
  renderLink: AppShellProps['renderLink'];
  formatLabel: (s: string) => string;
  forceOpen?: boolean;
}) {
  const [open, setOpen] = useState(true);
  const isOpen = forceOpen || open;

  return (
    <>
      <div
        className="sidebar-section-label"
        onClick={() => setOpen((v) => !v)}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && setOpen((v) => !v)}
      >
        <span>{formatLabel(label)}</span>
        <span className={`sidebar-section-arrow${isOpen ? '' : ' sidebar-section-arrow--closed'}`}>‹</span>
      </div>
      {isOpen && items.map((item) => (
        renderLink(item, 'sidebar-link')
      ))}
    </>
  );
}
