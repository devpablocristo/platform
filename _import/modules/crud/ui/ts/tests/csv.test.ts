import { describe, expect, it } from 'vitest';

import { buildCSV, normalizeCSVFieldValue, parseCSV } from '../src/csv';

describe('csv helpers', () => {
  it('builds CSV with BOM and escaped cells', () => {
    const csv = buildCSV(
      [
        { key: 'name', label: 'Nombre' },
        { key: 'notes', label: 'Notas' },
      ],
      [
        { name: 'Juan', notes: 'Linea 1\nLinea 2' },
        { name: 'Ana', notes: 'Comillas "dobles"' },
      ],
    );

    expect(csv).toBe('\uFEFFname,notes\nJuan,"Linea 1\nLinea 2"\nAna,"Comillas ""dobles"""');
  });

  it('parses quoted CSV rows and normalizes checkbox values', () => {
    const rows = parseCSV('name,active,notes\n"Juan, Perez",si,"Revisar ""frenos"""');

    expect(rows).toEqual([
      {
        name: 'Juan, Perez',
        active: 'si',
        notes: 'Revisar "frenos"',
      },
    ]);
    expect(normalizeCSVFieldValue(rows[0].active, 'checkbox')).toBe(true);
  });
});
