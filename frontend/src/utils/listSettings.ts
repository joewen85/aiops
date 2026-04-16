interface PersistedListSettings {
  [key: string]: unknown;
}

interface ColumnDefinition {
  key: string;
  required?: boolean;
}

function canUseLocalStorage(): boolean {
  return typeof window !== "undefined" && typeof window.localStorage !== "undefined";
}

export function loadPersistedListSettings(storageKey: string): PersistedListSettings | null {
  if (!canUseLocalStorage()) return null;
  const raw = window.localStorage.getItem(storageKey);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return null;
    return parsed as PersistedListSettings;
  } catch {
    return null;
  }
}

export function savePersistedListSettings(
  storageKey: string,
  settings: PersistedListSettings,
): void {
  if (!canUseLocalStorage()) return;
  window.localStorage.setItem(storageKey, JSON.stringify(settings));
}

export function sanitizeVisibleColumnKeys(
  visibleColumnKeys: unknown,
  columns: ColumnDefinition[],
): string[] {
  const allowed = new Set(columns.map((column) => column.key));
  const required = new Set(columns.filter((column) => column.required).map((column) => column.key));

  const requested = Array.isArray(visibleColumnKeys)
    ? visibleColumnKeys.filter((item): item is string => typeof item === "string" && allowed.has(item))
    : [];

  const requestedSet = new Set(requested);
  required.forEach((key) => requestedSet.add(key));

  const normalized = columns
    .map((column) => column.key)
    .filter((key) => requestedSet.has(key));

  if (normalized.length > 0) return normalized;
  return columns.map((column) => column.key);
}

export function sanitizeStatusFilter<T extends string>(
  statusFilter: unknown,
  allowed: readonly T[],
  fallback: T,
): T {
  if (typeof statusFilter === "string" && allowed.includes(statusFilter as T)) {
    return statusFilter as T;
  }
  return fallback;
}

export function sanitizeMultiFilterValues<T extends string>(
  values: unknown,
  allowed: readonly T[],
  fallback: readonly T[],
): T[] {
  const allowedSet = new Set(allowed);
  const requested = Array.isArray(values)
    ? values.filter((item): item is T => typeof item === "string" && allowedSet.has(item as T))
    : [];
  if (requested.length > 0) {
    const requestedSet = new Set(requested);
    return allowed.filter((item) => requestedSet.has(item));
  }
  return [...fallback];
}
