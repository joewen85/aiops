import type { PermissionBundle } from "@/types/permission";

const PERMISSION_KEY = "devops_permission_bundle";
const USER_KEY = "devops_user";
const PERMISSION_EVENT = "permission:updated";

export function getPermissionBundle(): PermissionBundle | null {
  const raw = localStorage.getItem(PERMISSION_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as PermissionBundle;
  } catch {
    return null;
  }
}

export function setPermissionBundle(bundle: PermissionBundle): void {
  localStorage.setItem(PERMISSION_KEY, JSON.stringify(bundle));
  window.dispatchEvent(new Event(PERMISSION_EVENT));
}

export function clearPermissionBundle(): void {
  localStorage.removeItem(PERMISSION_KEY);
  window.dispatchEvent(new Event(PERMISSION_EVENT));
}

export function permissionEventName(): string {
  return PERMISSION_EVENT;
}

export function hasMenuPermission(key: string, bundle?: PermissionBundle | null): boolean {
  return hasPermissionByType(key, "menu", bundle);
}

export function hasButtonPermission(key: string, bundle?: PermissionBundle | null): boolean {
  return hasPermissionByType(key, "button", bundle);
}

function hasPermissionByType(key: string, type: "menu" | "button", bundle?: PermissionBundle | null): boolean {
  if (!key) return true;
  if (isAdminUser()) return true;
  const current = bundle ?? getPermissionBundle();
  if (!current) return false;
  if (current.allAccess) return true;
  const keys = type === "menu" ? current.menuKeys : current.buttonKeys;
  return keys.includes(key);
}

export function isAdminUser(): boolean {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return false;
  try {
    const user = JSON.parse(raw) as { roles?: string[] };
    return (user.roles ?? []).some((role) => role.toLowerCase() === "admin");
  } catch {
    return false;
  }
}
