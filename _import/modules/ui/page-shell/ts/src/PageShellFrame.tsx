import type { PropsWithChildren, ReactNode } from 'react';
import { AppShell, type AppShellNavItem, type AppShellNavSection } from '@devpablocristo/modules-shell-sidebar';
import { PageSearchProvider } from '@devpablocristo/core-browser/search';

export type PageShellFrameProps = PropsWithChildren<{
  brandTitle: string;
  brandSubtitle: string;
  sections: AppShellNavSection[];
  footerContent?: ReactNode;
  pathname?: string;
  formatLabel?: (label: string) => string;
  renderLink: (item: AppShellNavItem, className: string) => ReactNode;
  searchPlaceholder?: string;
  skipLinkLabel?: string;
  mainContentId?: string;
}>;

export function PageShellFrame({
  children,
  brandTitle,
  brandSubtitle,
  sections,
  footerContent,
  pathname,
  formatLabel,
  renderLink,
  searchPlaceholder = 'Buscar...',
  skipLinkLabel = 'Ir al contenido',
  mainContentId = 'main-content',
}: PageShellFrameProps) {
  return (
    <>
      <a href={`#${mainContentId}`} className="skip-link">
        {skipLinkLabel}
      </a>
      <AppShell
        brandTitle={brandTitle}
        brandSubtitle={brandSubtitle}
        sections={sections}
        footerContent={footerContent}
        pathname={pathname}
        formatLabel={formatLabel}
        renderLink={renderLink}
        searchPlaceholder={searchPlaceholder}
      >
        <PageSearchProvider placeholder={searchPlaceholder}>
          <main id={mainContentId} className="app-shell-main" tabIndex={-1}>
            {children}
          </main>
        </PageSearchProvider>
      </AppShell>
    </>
  );
}
