import { useEffect, useState } from "react";

export function ToastViewport() {
  const [message, setMessage] = useState("");

  useEffect(() => {
    const handler = (event: Event) => {
      const custom = event as CustomEvent<string>;
      setMessage(custom.detail ?? "");
      setTimeout(() => setMessage(""), 2500);
    };
    window.addEventListener("app:toast", handler);
    return () => window.removeEventListener("app:toast", handler);
  }, []);

  if (!message) return null;
  return <div className="toast" role="alert">{message}</div>;
}
