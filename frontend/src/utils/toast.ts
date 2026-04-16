export function showToast(message: string): void {
  window.dispatchEvent(new CustomEvent("app:toast", { detail: message }));
}
