// @vitest-environment jsdom

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { FilterBar, type FilterItem } from './FilterBar';

describe('FilterBar', () => {
  it('selects a suggested search option and passes the full item back', () => {
    const onChange = vi.fn();
    const setData = vi.fn();
    const filters: FilterItem[] = [
      {
        type: 'search',
        name: 'customer',
        label: 'Customer',
        placeholder: 'Search customers',
        value: '',
        options: [
          { id: 1, name: 'Ada Lovelace' },
          { id: 2, name: 'Grace Hopper' },
        ],
        onChange,
        setData,
      },
    ];

    render(<FilterBar filters={filters} />);

    const input = screen.getByPlaceholderText('Search customers');
    fireEvent.focus(input);
    fireEvent.click(screen.getByText('Grace Hopper'));

    expect(setData).toHaveBeenCalledWith({ id: 2, name: 'Grace Hopper' });
    expect(onChange).toHaveBeenCalledWith('Grace Hopper');
  });

  it('maps select options and executes action buttons', () => {
    const onChange = vi.fn();
    const setData = vi.fn();
    const onExport = vi.fn();
    const filters: FilterItem[] = [
      {
        type: 'select',
        name: 'branch',
        label: 'Branch',
        placeholder: 'Pick branch',
        value: '',
        options: [
          { id: 1, name: 'North' },
          { id: 2, name: 'South' },
        ],
        onChange,
        setData,
      },
    ];

    render(
      <FilterBar
        filters={filters}
        actions={[{ label: 'Export', onClick: onExport }]}
      />,
    );

    fireEvent.change(screen.getByRole('combobox'), { target: { value: '2' } });
    expect(setData).toHaveBeenCalledWith({ id: 2, name: 'South' });
    expect(onChange).toHaveBeenCalledWith('South');

    fireEvent.click(screen.getByRole('button', { name: 'Export' }));
    expect(onExport).toHaveBeenCalledTimes(1);
  });
});
