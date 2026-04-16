import { useEffect, useState } from "react";

type ThemeMode = "light" | "dark";
const THEME_KEY = "devops_theme";

export function useTheme() {
  const [mode, setMode] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", mode);
    localStorage.setItem(THEME_KEY, mode);
  }, [mode]);

  return {
    mode,
    toggle: () => setMode((value) => (value === "light" ? "dark" : "light")),
  };
}
