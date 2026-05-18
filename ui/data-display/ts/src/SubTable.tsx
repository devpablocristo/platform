import type { ReactNode } from "react";

export type SubColumn<T> = {
  key: keyof T;
  header: string;
  render?: (value: T[keyof T], item: T) => ReactNode;
};

export type SubTableProps<T> = {
  data: T[];
  columns: SubColumn<T>[];
  className?: string;
};

export function SubTable<T>({ data, columns, className }: SubTableProps<T>) {
  return (
    <div className={`overflow-x-auto rounded-xl ${className ?? ""}`.trim()}>
      <table className="w-full overflow-hidden rounded-xl border border-slate-200 bg-white text-left text-sm text-slate-700 shadow-sm">
        <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
          <tr>
            {columns.map((column) => (
              <th key={String(column.key)} className="px-4 py-2 text-[11px] font-semibold uppercase tracking-wider">
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((item, index) => (
            <tr
              key={index}
              className="border-t border-slate-100 font-normal transition-colors duration-150 hover:bg-slate-50"
            >
              {columns.map((column) => (
                <td key={String(column.key)} className="px-4 py-2 font-normal">
                  {column.render ? column.render(item[column.key], item) : String(item[column.key] ?? "")}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
