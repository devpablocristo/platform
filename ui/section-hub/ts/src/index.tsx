import type { ReactNode } from 'react';
import { PageLayout } from '@devpablocristo/modules-ui-page-shell';

export type SectionHubSection<TSection extends string = string> = {
  id: TSection;
  label: string;
  desc: string;
  icon: ReactNode;
};

export function parseSectionHubSelection<TSection extends string>(
  sections: readonly SectionHubSection<TSection>[],
  raw: string | null,
): TSection | null {
  if (!raw) {
    return null;
  }
  const knownSections = new Set(sections.map((section) => section.id));
  return knownSections.has(raw as TSection) ? (raw as TSection) : null;
}

export type SectionHubPageProps<TSection extends string> = {
  className?: string;
  pageTitle: ReactNode;
  pageLead?: ReactNode;
  sections: readonly SectionHubSection<TSection>[];
  visibleSections?: readonly SectionHubSection<TSection>[];
  emptyState?: ReactNode;
  activeSectionId: TSection | null;
  onOpenSection: (id: TSection) => void;
  onBack: () => void;
  backLabel?: ReactNode;
  children?: ReactNode;
};

export function SectionHubPage<TSection extends string>({
  className,
  pageTitle,
  pageLead,
  sections,
  visibleSections,
  emptyState,
  activeSectionId,
  onOpenSection,
  onBack,
  backLabel = 'Volver',
  children,
}: SectionHubPageProps<TSection>) {
  const activeSection = sections.find((section) => section.id === activeSectionId) ?? null;
  const cards = visibleSections ?? sections;

  if (activeSection == null) {
    return (
      <PageLayout className={className} title={pageTitle} lead={pageLead}>
        {cards.length > 0 ? (
          <div className="section-hub__nav-grid">
            {cards.map((section) => (
              <button
                key={section.id}
                type="button"
                className="section-hub__nav-card"
                onClick={() => onOpenSection(section.id)}
              >
                <div className="section-hub__nav-icon">{section.icon}</div>
                <div className="section-hub__nav-info">
                  <div className="section-hub__nav-title">{section.label}</div>
                  <div className="section-hub__nav-desc">{section.desc}</div>
                </div>
              </button>
            ))}
          </div>
        ) : (
          emptyState ?? <div className="section-hub__empty">No hay secciones para mostrar.</div>
        )}
      </PageLayout>
    );
  }

  return (
    <PageLayout
      className={className}
      title={activeSection.label}
      lead={activeSection.desc}
      actions={
        <button type="button" className="section-hub__back" onClick={onBack}>
          {backLabel}
        </button>
      }
    >
      {children}
    </PageLayout>
  );
}
