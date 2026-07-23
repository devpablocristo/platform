import React, { type PropsWithChildren, type ReactNode } from 'react';
import {
  AppShell,
  type AppShellLabels,
  type AppShellNavItem,
  type AppShellNavSection,
} from '@devpablocristo/platform-shell-sidebar';
import { PageSearchProvider } from '@devpablocristo/platform-browser/search';

export type PageShellFrameProps = PropsWithChildren<{
  brandTitle: ReactNode;
  brandSubtitle: string;
  brandIcon?: ReactNode;
  sections: AppShellNavSection[];
  footerContent?: ReactNode;
  pathname?: string;
  formatLabel?: (label: string) => string;
  renderLink: (item: AppShellNavItem, className: string) => ReactNode;
  searchPlaceholder?: string;
  shellLabels?: Partial<AppShellLabels>;
  skipLinkLabel?: string;
  mainContentId?: string;
}>;

export function PageShellFrame({
  children,
  brandTitle,
  brandSubtitle,
  brandIcon,
  sections,
  footerContent,
  pathname,
  formatLabel,
  renderLink,
  searchPlaceholder = 'Buscar...',
  shellLabels,
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
        brandIcon={brandIcon}
        sections={sections}
        footerContent={footerContent}
        pathname={pathname}
        formatLabel={formatLabel}
        renderLink={renderLink}
        searchPlaceholder={searchPlaceholder}
        labels={shellLabels}
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
