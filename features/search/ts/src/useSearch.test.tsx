// @vitest-environment jsdom

import { act, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useSearch } from './useSearch';

const searchMock = vi.hoisted(() => vi.fn());

vi.mock('@devpablocristo/core-browser/search', () => ({
  search: (...args: unknown[]) => searchMock(...args),
}));

function SearchProbe({
  items,
  query,
  debounceMs,
}: {
  items: string[];
  query: string;
  debounceMs?: number;
}) {
  const results = useSearch(items, (item) => item, query, { debounceMs });
  return <div data-testid="results">{results.join('|')}</div>;
}

describe('useSearch', () => {
  it('returns the full list when the query is empty', () => {
    searchMock.mockReset();

    render(<SearchProbe items={['Ada', 'Grace', 'Linus']} query="" />);

    expect(screen.getByTestId('results').textContent).toBe('Ada|Grace|Linus');
    expect(searchMock).not.toHaveBeenCalled();
  });

  it('debounces the fuzzy search query before filtering', async () => {
    vi.useFakeTimers();
    searchMock.mockReset();
    searchMock.mockReturnValue([{ item: 'Grace', score: 0.9 }]);

    const { rerender } = render(<SearchProbe items={['Ada', 'Grace', 'Linus']} query="" debounceMs={200} />);

    rerender(<SearchProbe items={['Ada', 'Grace', 'Linus']} query="gr" debounceMs={200} />);
    expect(searchMock).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(200);
    });

    expect(searchMock).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('results').textContent).toBe('Grace');
    vi.useRealTimers();
  });
});
