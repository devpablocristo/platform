export type PaginationBarProps = {
  page: number;
  perPage: number;
  total: number;
  onPageChange: (page: number) => void;
  className?: string;
};

function getPaginationRange(totalPages: number, currentPage: number): (number | null)[] {
  const delta = 1;
  const range: number[] = [];
  const out: (number | null)[] = [];
  let previous: number | undefined;

  for (let page = 1; page <= totalPages; page += 1) {
    if (page === 1 || page === totalPages || (page >= currentPage - delta && page <= currentPage + delta)) {
      range.push(page);
    }
  }

  for (const page of range) {
    if (previous !== undefined) {
      if (page - previous === 2) {
        out.push(previous + 1);
      } else if (page - previous > 2) {
        out.push(null);
      }
    }

    out.push(page);
    previous = page;
  }

  return out;
}

export function PaginationBar({ page, perPage, total, onPageChange, className }: PaginationBarProps) {
  if (total <= 0 || perPage <= 0) return null;

  const totalPages = Math.max(1, Math.ceil(total / perPage));
  const currentPage = Math.min(Math.max(1, page), totalPages);
  const start = (currentPage - 1) * perPage + 1;
  const end = Math.min(currentPage * perPage, total);

  const goToPage = (nextPage: number) => {
    if (nextPage < 1 || nextPage > totalPages || nextPage === currentPage) return;
    onPageChange(nextPage);
  };

  return (
    <nav
      className={`flex flex-col items-start justify-between space-y-3 bg-white p-4 md:flex-row md:items-center md:space-y-0 ${className ?? ""}`}
      aria-label="Table navigation"
    >
      <span className="text-xs font-normal text-slate-500">
        Mostrar
        <span className="mx-1 font-semibold text-slate-800">
          {start}-{end}
        </span>
        de
        <span className="ml-1 font-semibold text-slate-800">{total}</span>
      </span>

      <ul className="inline-flex items-stretch -space-x-px">
        <li>
          <button
            type="button"
            onClick={() => goToPage(currentPage - 1)}
            disabled={currentPage === 1}
            className="ml-0 flex h-full min-h-9 min-w-9 items-center justify-center rounded-l-lg border border-slate-200 bg-white px-3 py-1.5 text-slate-400 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Página anterior"
          >
            ‹
          </button>
        </li>

        {getPaginationRange(totalPages, currentPage).map((rangePage, index) =>
          rangePage === null ? (
            <li
              key={`ellipsis-${index}`}
              className="flex min-h-9 min-w-9 select-none items-center justify-center border border-slate-200 bg-white px-2 text-slate-400"
            >
              ...
            </li>
          ) : (
            <li key={rangePage}>
              <button
                type="button"
                onClick={() => goToPage(rangePage)}
                aria-current={currentPage === rangePage ? "page" : undefined}
                className={`flex min-h-9 min-w-9 items-center justify-center border px-3 py-2 text-sm transition-colors ${
                  currentPage === rangePage
                    ? "border-primary-200 bg-primary-50 font-semibold text-primary-700"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                {rangePage}
              </button>
            </li>
          ),
        )}

        <li>
          <button
            type="button"
            onClick={() => goToPage(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="flex h-full min-h-9 min-w-9 items-center justify-center rounded-r-lg border border-slate-200 bg-white px-3 py-1.5 text-slate-400 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Página siguiente"
          >
            ›
          </button>
        </li>
      </ul>
    </nav>
  );
}
