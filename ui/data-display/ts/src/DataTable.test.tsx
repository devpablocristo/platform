// @vitest-environment jsdom

import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { DataTable, type DataTableColumn } from './DataTable';

type Row = {
  name: string;
  age: number;
  cost?: string;
};

const rows: Row[] = [
  { name: 'Grace', age: 41, cost: '100 Kg' },
  { name: 'Ada', age: 30, cost: '20 Kg' },
  { name: 'Linus', age: 21, cost: '3 Kg' },
];

const columns: DataTableColumn<Row>[] = [
  { key: 'name', header: 'Name', filterType: 'text' },
  { key: 'age', header: 'Age' },
];

function visibleNames(limit = 2) {
  return screen
    .getAllByRole('row')
    .slice(1, 1 + limit)
    .map((row) => within(row).getAllByRole('cell')[0]?.textContent);
}

describe('DataTable', () => {
  it('sorts rows and forwards pagination changes', () => {
    const onPageChange = vi.fn();

    render(
      <DataTable
        data={rows}
        columns={columns}
        pagination={{ page: 1, perPage: 2, total: rows.length, onPageChange }}
      />,
    );

    const ageHeader = screen.getByText('Age').closest('th');
    const sortButton = within(ageHeader as HTMLElement).getByRole('button');

    fireEvent.click(sortButton);
    expect(visibleNames()).toEqual(['Linus', 'Ada']);

    fireEvent.click(sortButton);
    expect(visibleNames()).toEqual(['Grace', 'Ada']);

    fireEvent.click(screen.getByRole('button', { name: '2' }));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it('applies and clears text filters through the callback contract', () => {
    const onFilterChange = vi.fn();

    render(
      <DataTable
        data={rows}
        columns={columns}
        filters={{}}
        onFilterChange={onFilterChange}
        enableFilters
      />,
    );

    const nameHeader = screen.getByText('Name').closest('th');
    fireEvent.click(within(nameHeader as HTMLElement).getByTitle('Filtrar'));
    fireEvent.change(screen.getByPlaceholderText('Buscar...'), { target: { value: 'Ada' } });
    expect(onFilterChange).toHaveBeenCalledWith({ name: 'Ada' });

    fireEvent.click(screen.getByRole('button', { name: 'Limpiar' }));
    expect(onFilterChange).toHaveBeenLastCalledWith({});
  });

  it('does not slice already-paginated server-side rows', () => {
    render(
      <DataTable
        data={rows}
        columns={columns}
        pagination={{
          page: 2,
          perPage: 2,
          total: 5,
          serverSide: true,
          onPageChange: vi.fn(),
        }}
      />,
    );

    expect(visibleNames(3)).toEqual(['Grace', 'Ada', 'Linus']);
    const navigationText = screen.getByLabelText('Table navigation').textContent ?? '';
    expect(navigationText).toMatch(/Mostrar\s*3\s*-\s*4\s*de\s*5/);
  });

  it('renders custom actions with a custom header', () => {
    const onOpen = vi.fn();

    render(
      <DataTable
        data={rows}
        columns={columns}
        actionsHeader="Tools"
        renderActions={(row) => (
          <button type="button" onClick={() => onOpen(row.name)}>
            Open {row.name}
          </button>
        )}
      />,
    );

    expect(screen.getByText('Tools')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Open Ada' }));
    expect(onOpen).toHaveBeenCalledWith('Ada');
  });

  it('sorts and filters select options with numeric-aware matching', () => {
    const onFilterChange = vi.fn();
    const costColumns: DataTableColumn<Row>[] = [
      {
        key: 'cost',
        header: 'Cost',
        filterType: 'select',
        filterOptions: ['3 Kg', '100 Kg', '20 Kg', 'Alpha'],
      },
    ];

    render(
      <DataTable
        data={rows}
        columns={costColumns}
        filters={{}}
        onFilterChange={onFilterChange}
        enableFilters
      />,
    );

    const costHeader = screen.getByText('Cost').closest('th');
    fireEvent.click(within(costHeader as HTMLElement).getByTitle('Filtrar'));

    const optionLabels = screen
      .getAllByLabelText(/Kg|Alpha/)
      .map((input) => input.closest('label')?.textContent?.trim());
    expect(optionLabels).toEqual(['100 Kg', '20 Kg', '3 Kg', 'Alpha']);

    fireEvent.change(screen.getByPlaceholderText('Buscar opción...'), { target: { value: '2' } });
    expect(screen.getByLabelText('20 Kg')).toBeTruthy();
    expect(screen.queryByLabelText('100 Kg')).toBeNull();
    expect(screen.queryByLabelText('3 Kg')).toBeNull();
  });
});
