import type { ReactNode } from "react";

export interface TableSettingsColumn {
  key: string;
  label: string;
  required?: boolean;
}

interface TableSettingsModalProps {
  open: boolean;
  title: string;
  columns: TableSettingsColumn[];
  visibleColumnKeys: string[];
  onToggleColumn: (columnKey: string) => void;
  onReset: () => void;
  onClose: () => void;
  extraContent?: ReactNode;
}

export function TableSettingsModal(props: TableSettingsModalProps) {
  const {
    open,
    title,
    columns,
    visibleColumnKeys,
    onToggleColumn,
    onReset,
    onClose,
    extraContent,
  } = props;

  if (!open) return null;

  return (
    <div className="table-settings-mask" onClick={onClose}>
      <section className="table-settings-modal" onClick={(event) => event.stopPropagation()}>
        <header className="table-settings-header">
          <h4>{title}</h4>
          <button className="btn ghost cursor-pointer" type="button" onClick={onClose}>
            关闭
          </button>
        </header>

        <div className="table-settings-body">
          <div className="table-settings-section">
            <h5>显示字段</h5>
            <div className="table-settings-columns">
              {columns.map((column) => {
                const checked = visibleColumnKeys.includes(column.key);
                return (
                  <label className="table-settings-option cursor-pointer" key={column.key}>
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={column.required}
                      onChange={() => onToggleColumn(column.key)}
                    />
                    <span>{column.label}</span>
                  </label>
                );
              })}
            </div>
          </div>

          {extraContent}
        </div>

        <footer className="table-settings-footer">
          <button className="btn ghost cursor-pointer" type="button" onClick={onReset}>
            重置
          </button>
          <button className="btn primary cursor-pointer" type="button" onClick={onClose}>
            确定
          </button>
        </footer>
      </section>
    </div>
  );
}
