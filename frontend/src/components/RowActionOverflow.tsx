import { CSSProperties, useCallback, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { usePermissionBundle } from "@/hooks/usePermissionBundle";
import { hasButtonPermission } from "@/store/permission";

export interface RowActionItem {
  key: string;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  permissionKey?: string;
  className?: string;
}

export interface RowActionOverflowProps {
  actions: RowActionItem[];
  maxInline?: number;
  moreLabel?: string;
  title?: string;
}

export function RowActionOverflow({
  actions,
  maxInline = 3,
  moreLabel = "...",
  title = "更多操作",
}: RowActionOverflowProps) {
  const permissionBundle = usePermissionBundle();
  const [open, setOpen] = useState(false);
  const [popupStyle, setPopupStyle] = useState<CSSProperties>({
    position: "fixed",
    top: 0,
    left: 0,
    width: 260,
    maxHeight: 240,
  });
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const visibleActions = useMemo(
    () => actions.filter((item) => !item.permissionKey || hasButtonPermission(item.permissionKey, permissionBundle)),
    [actions, permissionBundle],
  );
  const inlineActions = visibleActions.slice(0, maxInline);
  const overflowActions = visibleActions.slice(maxInline);

  if (visibleActions.length === 0) return null;

  const updatePopupPosition = useCallback(() => {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const rect = trigger.getBoundingClientRect();
    const margin = 8;
    const gap = 6;
    const width = Math.min(320, Math.max(220, rect.width * 5));

    let left = rect.left;
    if (left + width > window.innerWidth - margin) {
      left = window.innerWidth - margin - width;
    }
    if (left < margin) {
      left = margin;
    }

    const top = rect.bottom + gap;
    const maxHeight = Math.max(120, window.innerHeight - top - margin);
    setPopupStyle({
      position: "fixed",
      top,
      left,
      width,
      maxHeight,
    });
  }, []);

  useLayoutEffect(() => {
    if (!open) return;
    updatePopupPosition();
    const recalc = () => updatePopupPosition();
    window.addEventListener("resize", recalc);
    window.addEventListener("scroll", recalc, true);
    return () => {
      window.removeEventListener("resize", recalc);
      window.removeEventListener("scroll", recalc, true);
    };
  }, [open, updatePopupPosition]);

  function handleOverflowActionClick(action: RowActionItem) {
    setOpen(false);
    action.onClick();
  }

  return (
    <>
      {inlineActions.map((action) => (
        <button
          key={action.key}
          className={action.className ?? "btn cursor-pointer"}
          type="button"
          disabled={action.disabled}
          onClick={action.onClick}
        >
          {action.label}
        </button>
      ))}
      {overflowActions.length > 0 ? (
        <button
          ref={triggerRef}
          className="btn cursor-pointer row-action-overflow-trigger"
          type="button"
          aria-label="更多操作"
          onClick={() => setOpen(true)}
        >
          {moreLabel}
        </button>
      ) : null}

      {open
        ? createPortal(
          <>
            <div className="row-action-overflow-overlay" onClick={() => setOpen(false)} />
            <section className="row-action-overflow-modal" style={popupStyle} onClick={(event) => event.stopPropagation()}>
              <header className="row-action-overflow-header">
                <h4>{title}</h4>
                <button className="btn ghost cursor-pointer" type="button" onClick={() => setOpen(false)}>
                  关闭
                </button>
              </header>
              <div className="row-action-overflow-body">
                {overflowActions.map((action) => (
                  <button
                    key={action.key}
                    className={`${action.className ?? "btn cursor-pointer"} row-action-overflow-item`}
                    type="button"
                    disabled={action.disabled}
                    onClick={() => handleOverflowActionClick(action)}
                  >
                    {action.label}
                  </button>
                ))}
              </div>
            </section>
          </>,
          document.body,
        )
        : null}
    </>
  );
}

export function ListRowActions(props: RowActionOverflowProps) {
  return <RowActionOverflow maxInline={3} {...props} />;
}
