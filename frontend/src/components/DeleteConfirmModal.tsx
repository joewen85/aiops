import { FormEvent, useEffect, useState } from "react";

interface DeleteConfirmModalProps {
  open: boolean;
  title?: string;
  description?: string;
  confirmText?: string;
  confirming?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function DeleteConfirmModal({
  open,
  title = "删除确认",
  description = "请输入确认文案后执行删除操作。",
  confirmText = "确认删除资源",
  confirming = false,
  onCancel,
  onConfirm,
}: DeleteConfirmModalProps) {
  const [inputValue, setInputValue] = useState("");

  useEffect(() => {
    if (!open) {
      setInputValue("");
    }
  }, [open]);

  if (!open) return null;

  const canConfirm = inputValue === confirmText && !confirming;

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canConfirm) return;
    onConfirm();
  }

  return (
    <div className="delete-confirm-mask" onClick={onCancel}>
      <section className="delete-confirm-modal" onClick={(event) => event.stopPropagation()}>
        <header className="delete-confirm-header">
          <h4>{title}</h4>
        </header>
        <form className="delete-confirm-body" onSubmit={handleSubmit}>
          <p className="muted">{description}</p>
          <p className="delete-confirm-hint">
            请输入：<code>{confirmText}</code>
          </p>
          <input
            className="delete-confirm-input"
            value={inputValue}
            autoFocus
            placeholder={confirmText}
            onChange={(event) => setInputValue(event.target.value)}
            onPaste={(event) => event.preventDefault()}
          />
          <div className="delete-confirm-actions">
            <button className="btn ghost cursor-pointer" type="button" onClick={onCancel} disabled={confirming}>
              取消
            </button>
            <button className="btn cursor-pointer" type="submit" disabled={!canConfirm}>
              {confirming ? "删除中..." : "确认删除"}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}
