export type CursorPagerProps = {
  hasMore: boolean;
  loading?: boolean;
  disabled?: boolean;
  onLoadMore: () => void;
  loadMoreLabel?: string;
  loadingLabel?: string;
  endLabel?: string;
  className?: string;
};

export function CursorPager({
  hasMore,
  loading = false,
  disabled = false,
  onLoadMore,
  loadMoreLabel = "Cargar más",
  loadingLabel = "Cargando...",
  endLabel = "No hay más resultados",
  className,
}: CursorPagerProps) {
  if (!hasMore) {
    return (
      <div className={className} aria-live="polite">
        <span>{endLabel}</span>
      </div>
    );
  }

  return (
    <div className={className}>
      <button type="button" onClick={onLoadMore} disabled={disabled || loading} aria-busy={loading}>
        {loading ? loadingLabel : loadMoreLabel}
      </button>
    </div>
  );
}

export const LoadMoreControl = CursorPager;
