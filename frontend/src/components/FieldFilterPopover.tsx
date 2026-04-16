import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

export interface FieldFilterOption {
  value: string;
  label: string;
}

interface FieldFilterPopoverProps {
  options: FieldFilterOption[];
  selectedValues: string[];
  onChange: (nextValues: string[]) => void;
  ariaLabel: string;
}

export function FieldFilterPopover(props: FieldFilterPopoverProps) {
  const { options, selectedValues, onChange, ariaLabel } = props;
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState("");
  const rootRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [popoverStyle, setPopoverStyle] = useState<{
    top: number;
    left: number;
    width: number;
    maxHeight: number;
  }>({
    top: 0,
    left: 0,
    width: 280,
    maxHeight: 260,
  });

  const filteredOptions = useMemo(() => {
    const normalized = keyword.trim().toLowerCase();
    if (!normalized) return options;
    return options.filter((option) => option.label.toLowerCase().includes(normalized));
  }, [keyword, options]);

  useEffect(() => {
    if (!open) return;
    const updatePosition = () => {
      const trigger = triggerRef.current;
      if (!trigger) return;
      const rect = trigger.getBoundingClientRect();
      const width = Math.min(300, Math.max(220, window.innerWidth - 16));
      const left = Math.min(
        Math.max(8, rect.left),
        Math.max(8, window.innerWidth - width - 8),
      );
      const preferredTop = rect.bottom + 8;
      const fallbackTop = Math.max(8, rect.top - 260 - 8);
      const fitsBottom = preferredTop + 220 <= window.innerHeight - 8;
      const top = fitsBottom ? preferredTop : fallbackTop;
      const maxHeight = Math.max(140, window.innerHeight - top - 8);
      setPopoverStyle({
        top,
        left,
        width,
        maxHeight,
      });
    };

    updatePosition();

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (rootRef.current?.contains(target)) return;
      if (popoverRef.current?.contains(target)) return;
      setOpen(false);
    };
    const handleScrollOrResize = () => updatePosition();
    window.addEventListener("mousedown", handleClickOutside);
    window.addEventListener("resize", handleScrollOrResize);
    window.addEventListener("scroll", handleScrollOrResize, true);
    return () => {
      window.removeEventListener("mousedown", handleClickOutside);
      window.removeEventListener("resize", handleScrollOrResize);
      window.removeEventListener("scroll", handleScrollOrResize, true);
    };
  }, [open]);

  function toggleValue(value: string) {
    const selectedSet = new Set(selectedValues);
    if (selectedSet.has(value)) {
      selectedSet.delete(value);
    } else {
      selectedSet.add(value);
    }
    const nextValues = options
      .map((option) => option.value)
      .filter((optionValue) => selectedSet.has(optionValue));
    if (nextValues.length === 0) return;
    onChange(nextValues);
  }

  function selectAll() {
    onChange(options.map((option) => option.value));
  }

  function resetKeyword() {
    setKeyword("");
  }

  const selectedSet = new Set(selectedValues);

  const popoverNode = open && typeof document !== "undefined"
    ? createPortal(
      <div
        ref={popoverRef}
        className="column-filter-popover"
        style={{
          top: `${popoverStyle.top}px`,
          left: `${popoverStyle.left}px`,
          width: `${popoverStyle.width}px`,
          maxHeight: `${popoverStyle.maxHeight}px`,
        }}
      >
        <div className="column-filter-search-wrap">
          <input
            className="column-filter-search"
            type="search"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            placeholder="搜索"
          />
        </div>
        <div className="column-filter-options">
          {filteredOptions.length === 0 && <p className="muted">未匹配到选项</p>}
          {filteredOptions.map((option) => (
            <label className="column-filter-option cursor-pointer" key={option.value}>
              <input
                type="checkbox"
                checked={selectedSet.has(option.value)}
                onChange={() => toggleValue(option.value)}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
        <div className="column-filter-actions">
          <button className="btn ghost cursor-pointer" type="button" onClick={selectAll}>
            全选
          </button>
          <button className="btn ghost cursor-pointer" type="button" onClick={resetKeyword}>
            清空搜索
          </button>
        </div>
      </div>,
      document.body,
    )
    : null;

  return (
    <>
      <div className="column-filter" ref={rootRef}>
        <button
          ref={triggerRef}
          className="column-filter-trigger cursor-pointer"
          type="button"
          aria-label={ariaLabel}
          onClick={() => setOpen((prev) => !prev)}
        >
          <svg viewBox="0 0 20 20" aria-hidden="true">
            <path d="M3 4h14l-5 6v5l-4 1v-6L3 4z" />
          </svg>
        </button>
      </div>
      {popoverNode}
    </>
  );
}
