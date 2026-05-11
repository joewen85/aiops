interface PaginationProps {
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  pageSizeOptions: number[];
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

export function Pagination({
  total,
  page,
  pageSize,
  totalPages,
  pageSizeOptions,
  onPageChange,
  onPageSizeChange,
}: PaginationProps) {
  return (
    <footer className="rbac-pagination">
      <div className="rbac-pagination-group">
        <span className="muted">共 {total} 条</span>
      </div>
      <div className="rbac-pagination-group">
        <select
          className="rbac-pagination-select cursor-pointer"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          {pageSizeOptions.map((item) => (
            <option key={item} value={item}>
              {item}条
            </option>
          ))}
        </select>
        <button
          className="btn ghost cursor-pointer"
          type="button"
          disabled={page <= 1}
          onClick={() => onPageChange(Math.max(1, page - 1))}
        >
          上一页
        </button>
        <span className="rbac-pagination-text">{page} / {totalPages}</span>
        <button
          className="btn ghost cursor-pointer"
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPageChange(Math.min(totalPages, page + 1))}
        >
          下一页
        </button>
      </div>
    </footer>
  );
}
