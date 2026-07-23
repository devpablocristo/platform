// @vitest-environment jsdom

import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './AppShell';

const searchMock = vi.hoisted(() => vi.fn());

vi.mock('@devpablocristo/platform-browser/search', () => ({
  search: (...args: unknown[]) => searchMock(...args),
}));

const sections = [
  {
    label: 'Operations',
    items: [
      { to: '/calendar', label: 'Calendar', icon: 'C' },
      { to: '/customers', label: 'Customers', icon: 'U' },
    ],
  },
  {
    label: 'Admin',
    items: [{ to: '/settings', label: 'Settings', icon: 'S' }],
  },
];

function renderShell() {
  return render(
    <AppShell
      brandTitle={<img alt="Pymes" src="/brand.svg" />}
      brandSubtitle="Operations"
      sections={sections}
      labels={{
        clearSearch: 'Clear search',
        noSearchResults: 'No results',
        collapseSidebar: 'Collapse navigation',
        expandSidebar: 'Expand navigation',
        openNavigation: 'Open navigation',
        closeNavigation: 'Close navigation',
        navigation: 'Primary navigation',
      }}
      renderLink={(item, className) => (
        <a key={item.to} href={item.to} className={className}>
          {item.label}
        </a>
      )}
    >
      <div>content</div>
    </AppShell>,
  );
}

describe('AppShell', () => {
  beforeEach(() => {
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      value: vi.fn(),
    });
    Object.defineProperty(window, 'requestAnimationFrame', {
      configurable: true,
      value: (callback: FrameRequestCallback) => {
        callback(0);
        return 1;
      },
    });
  });

  it('filters navigation results and shows the empty state when there are no matches', () => {
    searchMock.mockReset();
    searchMock
      .mockReturnValueOnce([{ item: { sectionLabel: 'Operations', item: sections[0].items[0] }, score: 0.9 }])
      .mockReturnValueOnce([]);

    renderShell();

    fireEvent.change(screen.getByLabelText('Buscar...'), { target: { value: 'cal' } });
    expect(screen.getByRole('link', { name: 'Calendar' })).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'Customers' })).toBeNull();

    fireEvent.change(screen.getByLabelText('Buscar...'), { target: { value: 'missing' } });
    expect(screen.getByText('No results')).toBeTruthy();
  });

  it('clears the search input and restores focus when the clear button is used', () => {
    searchMock.mockReset();
    searchMock.mockReturnValue([{ item: { sectionLabel: 'Operations', item: sections[0].items[0] }, score: 0.9 }]);

    renderShell();

    const input = screen.getByLabelText('Buscar...');
    fireEvent.change(input, { target: { value: 'cal' } });
    fireEvent.click(screen.getByRole('button', { name: 'Clear search' }));

    expect((input as HTMLInputElement).value).toBe('');
    expect(document.activeElement).toBe(input);
  });

  it('uses native accessible controls for desktop collapse', () => {
    searchMock.mockReset();
    renderShell();

    expect(screen.getByRole('img', { name: 'Pymes' })).toBeTruthy();
    const collapse = screen.getByRole('button', { name: 'Collapse navigation' });
    expect(collapse.getAttribute('aria-expanded')).toBe('true');

    fireEvent.click(collapse);

    expect(screen.getByRole('button', { name: 'Expand navigation' }).getAttribute('aria-expanded')).toBe(
      'false',
    );
    expect(screen.queryByLabelText('Buscar...')).toBeNull();
  });

  it('opens and closes the mobile drawer with Escape and restores trigger focus', () => {
    searchMock.mockReset();
    renderShell();

    const trigger = screen.getByRole('button', { name: 'Open navigation' });
    fireEvent.click(trigger);

    expect(screen.getByRole('complementary', { name: 'Primary navigation' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Close navigation' }).getAttribute('aria-expanded')).toBe(
      'true',
    );

    fireEvent.keyDown(document, { key: 'Escape' });

    const reopenedTrigger = screen.getByRole('button', { name: 'Open navigation' });
    expect(reopenedTrigger.getAttribute('aria-expanded')).toBe('false');
    expect(document.activeElement).toBe(reopenedTrigger);
  });

  it('does not own the main landmark', () => {
    searchMock.mockReset();
    const { container } = renderShell();

    expect(container.querySelectorAll('main')).toHaveLength(0);
  });
});
