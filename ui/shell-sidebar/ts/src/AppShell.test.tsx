// @vitest-environment jsdom

import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppShell } from './AppShell';

const searchMock = vi.hoisted(() => vi.fn());

vi.mock('@devpablocristo/core-browser/search', () => ({
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
      brandTitle="Pymes"
      brandSubtitle="Operations"
      sections={sections}
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
    expect(screen.getByText('Sin resultados')).toBeTruthy();
  });

  it('clears the search input and restores focus when the clear button is used', () => {
    searchMock.mockReset();
    searchMock.mockReturnValue([{ item: { sectionLabel: 'Operations', item: sections[0].items[0] }, score: 0.9 }]);

    renderShell();

    const input = screen.getByLabelText('Buscar...');
    fireEvent.change(input, { target: { value: 'cal' } });
    fireEvent.click(screen.getByRole('button', { name: 'Limpiar búsqueda' }));

    expect((input as HTMLInputElement).value).toBe('');
    expect(document.activeElement).toBe(input);
  });
});
