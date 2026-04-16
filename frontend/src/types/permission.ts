import type { PermissionItem } from "@/types/rbac";

export interface PermissionBundle {
  permissions: PermissionItem[];
  menuKeys: string[];
  buttonKeys: string[];
  apiKeys: string[];
  allAccess: boolean;
}
