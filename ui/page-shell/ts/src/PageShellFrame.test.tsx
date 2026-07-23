// @vitest-environment jsdom

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { PageShellFrame } from './PageShellFrame';

describe('PageShellFrame', () => {
  it('owns the only main landmark and accepts composed branding', () => {
    const { container } = render(
      <PageShellFrame
        brandTitle={
          <span>
            Acme <strong>Suite</strong>
          </span>
        }
        brandSubtitle="Operations"
        brandIcon={<span aria-hidden="true">A</span>}
        sections={[
          {
            label: 'General',
            items: [{ to: '/dashboard', label: 'Dashboard', icon: <span aria-hidden="true">D</span> }],
          },
        ]}
        pathname="/dashboard"
        renderLink={(item, className) => (
          <a key={item.to} className={className} href={item.to}>
            {item.label}
          </a>
        )}
        shellLabels={{ navigation: 'Primary navigation' }}
        skipLinkLabel="Skip to content"
      >
        <h1>Dashboard</h1>
      </PageShellFrame>,
    );

    expect(container.querySelectorAll('main')).toHaveLength(1);
    expect(screen.getByRole('main').getAttribute('id')).toBe('main-content');
    expect(screen.getByRole('link', { name: 'Skip to content' }).getAttribute('href')).toBe(
      '#main-content',
    );
    expect(screen.getByRole('complementary', { name: 'Primary navigation' })).toBeTruthy();
    expect(screen.getByText('Suite')).toBeTruthy();
  });
});
