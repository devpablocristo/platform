import {
  default as React,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PropsWithChildren,
  type ReactNode,
} from 'react';
import { search, type SearchEntry } from '@devpablocristo/platform-browser/search';

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

export type AppShellLabels = {
  clearSearch: string;
  noSearchResults: string;
  collapseSidebar: string;
  expandSidebar: string;
  openNavigation: string;
  closeNavigation: string;
  navigation: string;
};

export type AppShellProps = PropsWithChildren<{
  brandTitle: ReactNode;
  brandSubtitle: string;
  brandIcon?: ReactNode;
  sections: AppShellNavSection[];
  footerContent?: ReactNode;
  /** Renders a navigation link. Receives the item and the shell class name. */
  renderLink: (item: AppShellNavItem, className: string) => ReactNode;
  /** Formats product-provided labels. Defaults to identity. */
  formatLabel?: (label: string) => string;
  /** Current pathname, used to reset scroll and close the mobile drawer. */
  pathname?: string;
  /** Search input placeholder. */
  searchPlaceholder?: string;
  /** Localized labels for shell-owned controls and states. */
  labels?: Partial<AppShellLabels>;
  /** Stable DOM id for the navigation drawer. */
  sidebarId?: string;
}>;

type FlatEntry = SearchEntry<{ sectionLabel: string; item: AppShellNavItem }>;

const defaultLabels: AppShellLabels = {
  clearSearch: 'Limpiar búsqueda',
  noSearchResults: 'Sin resultados',
  collapseSidebar: 'Contraer navegación',
  expandSidebar: 'Expandir navegación',
  openNavigation: 'Abrir navegación',
  closeNavigation: 'Cerrar navegación',
  navigation: 'Navegación principal',
};

const focusableSelector = [
  'a[href]',
  'button:not([disabled]):not(.sidebar-collapse-button)',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

export function AppShell({
  children,
  brandTitle,
  brandSubtitle,
  brandIcon,
  sections,
  footerContent,
  renderLink,
  formatLabel = (label) => label,
  pathname,
  searchPlaceholder = 'Buscar...',
  labels,
  sidebarId = 'app-shell-sidebar',
}: AppShellProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);
  const mobileTriggerRef = useRef<HTMLButtonElement>(null);
  const resolvedLabels = { ...defaultLabels, ...labels };

  useEffect(() => {
    const content = document.querySelector<HTMLElement>('.main-content');
    if (typeof content?.scrollTo === 'function') {
      content.scrollTo({ top: 0, left: 0, behavior: 'auto' });
    }
    setMobileDrawerOpen(false);
  }, [pathname]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        if (sidebarCollapsed) {
          setSidebarCollapsed(false);
          window.requestAnimationFrame(() => inputRef.current?.focus());
          return;
        }
        inputRef.current?.focus();
      }

      if (event.key === 'Escape' && document.activeElement === inputRef.current) {
        setQuery('');
        inputRef.current?.blur();
      }
    }

    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [sidebarCollapsed]);

  useEffect(() => {
    if (!mobileDrawerOpen) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.preventDefault();
        setMobileDrawerOpen(false);
        window.requestAnimationFrame(() => mobileTriggerRef.current?.focus());
        return;
      }

      if (event.key !== 'Tab') return;
      const elements = [
        mobileTriggerRef.current,
        ...Array.from(sidebarRef.current?.querySelectorAll<HTMLElement>(focusableSelector) ?? []),
      ].filter((element): element is HTMLElement => element !== null && !element.hasAttribute('disabled'));
      if (elements.length === 0) {
        event.preventDefault();
        return;
      }

      const first = elements[0];
      const last = elements[elements.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }

    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [mobileDrawerOpen]);

  const flatEntries = useMemo<FlatEntry[]>(
    () =>
      sections.flatMap((section) =>
        section.items.map((item) => ({
          item: { sectionLabel: section.label, item },
          text: `${section.label} ${item.label}`,
        })),
      ),
    [sections],
  );

  const filteredSections = useMemo(() => {
    const normalizedQuery = query.trim();
    if (normalizedQuery.length === 0) return sections;

    const matches = search(normalizedQuery, flatEntries);
    const matchedItems = new Set(matches.map((match) => match.item.item));

    return sections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => matchedItems.has(item)),
      }))
      .filter((section) => section.items.length > 0);
  }, [flatEntries, query, sections]);

  const closeMobileDrawer = () => {
    setMobileDrawerOpen(false);
    window.requestAnimationFrame(() => mobileTriggerRef.current?.focus());
  };

  return (
    <div
      className={[
        'app-layout',
        sidebarCollapsed ? 'sidebar-collapsed' : '',
        mobileDrawerOpen ? 'sidebar-mobile-open' : '',
      ].filter(Boolean).join(' ')}
    >
      <button
        ref={mobileTriggerRef}
        type="button"
        className="sidebar-mobile-trigger"
        aria-controls={sidebarId}
        aria-expanded={mobileDrawerOpen}
        aria-label={mobileDrawerOpen ? resolvedLabels.closeNavigation : resolvedLabels.openNavigation}
        onClick={() => setMobileDrawerOpen((open) => !open)}
      >
        <span aria-hidden="true">{mobileDrawerOpen ? '×' : '☰'}</span>
      </button>

      {mobileDrawerOpen && (
        <div
          className="sidebar-overlay"
          aria-hidden="true"
          onClick={closeMobileDrawer}
        />
      )}

      <aside
        ref={sidebarRef}
        id={sidebarId}
        className="sidebar"
        aria-label={resolvedLabels.navigation}
      >
        <div className="sidebar-brand">
          <div className="sidebar-brand-row">
            {brandIcon && <span className="sidebar-brand-icon">{brandIcon}</span>}
            <div className="sidebar-brand-title">{brandTitle}</div>
            <button
              type="button"
              className="sidebar-collapse-button"
              aria-controls={sidebarId}
              aria-expanded={!sidebarCollapsed}
              aria-label={
                sidebarCollapsed ? resolvedLabels.expandSidebar : resolvedLabels.collapseSidebar
              }
              onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
            >
              <span aria-hidden="true">{sidebarCollapsed ? '›' : '‹'}</span>
            </button>
          </div>
          <small>{formatLabel(brandSubtitle)}</small>
        </div>

        {!sidebarCollapsed && (
          <div className="sidebar-search">
            <input
              ref={inputRef}
              type="search"
              className="sidebar-search-input"
              placeholder={searchPlaceholder}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              aria-label={searchPlaceholder}
            />
            {query.length > 0 && (
              <button
                className="sidebar-search-clear"
                onClick={() => {
                  setQuery('');
                  inputRef.current?.focus();
                }}
                aria-label={resolvedLabels.clearSearch}
                type="button"
              >
                <span aria-hidden="true">×</span>
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
            <div className="sidebar-search-empty">{resolvedLabels.noSearchResults}</div>
          )}
        </nav>

        <div className="sidebar-footer">{footerContent ?? null}</div>
      </aside>

      <div className="main-content">{children}</div>
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
  formatLabel: (label: string) => string;
  forceOpen?: boolean;
}) {
  const [open, setOpen] = useState(true);
  const isOpen = forceOpen || open;

  return (
    <>
      <button
        type="button"
        className="sidebar-section-label"
        aria-expanded={isOpen}
        onClick={() => setOpen((current) => !current)}
      >
        <span>{formatLabel(label)}</span>
        <span
          aria-hidden="true"
          className={`sidebar-section-arrow${isOpen ? '' : ' sidebar-section-arrow--closed'}`}
        >
          ‹
        </span>
      </button>
      {isOpen && items.map((item) => renderLink(item, 'sidebar-link'))}
    </>
  );
}
