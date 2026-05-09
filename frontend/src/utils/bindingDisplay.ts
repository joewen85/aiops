export type BindingDisplayItem =
  | string
  | number
  | {
    id?: unknown;
    ID?: unknown;
    name?: unknown;
    Name?: unknown;
    displayName?: unknown;
  }
  | null
  | undefined;

export function formatBindingNameList(
  items?: BindingDisplayItem[],
  ids?: Array<number | string | null | undefined>,
  fallbackPrefix = "资源",
): string {
  const names = (items ?? [])
    .map(bindingDisplayName)
    .filter((value): value is string => Boolean(value));
  if (names.length > 0) {
    return uniqueValues(names).join("、");
  }

  const fallbackIDs = uniqueValues(
    (ids ?? [])
      .map((value) => Number(value))
      .filter((value) => Number.isInteger(value) && value > 0)
      .map((value) => `${fallbackPrefix}#${value}`),
  );
  return fallbackIDs.length > 0 ? fallbackIDs.join("、") : "-";
}

function bindingDisplayName(item: BindingDisplayItem): string | undefined {
  if (typeof item === "string") return item.trim() || undefined;
  if (typeof item === "number") return Number.isFinite(item) ? String(item) : undefined;
  if (!item || typeof item !== "object") return undefined;

  const name = firstNonEmptyString(item.name, item.Name, item.displayName);
  if (name) return name;

  const id = Number(item.id ?? item.ID);
  return Number.isInteger(id) && id > 0 ? `#${id}` : undefined;
}

function firstNonEmptyString(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value !== "string") continue;
    const text = value.trim();
    if (text) return text;
  }
  return undefined;
}

function uniqueValues(values: string[]): string[] {
  return [...new Set(values)];
}
