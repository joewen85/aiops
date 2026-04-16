export type PermissionType = "api" | "menu" | "button";

export interface RoleItem {
  id: number;
  name: string;
  description: string;
  builtIn: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface PermissionItem {
  id: number;
  name: string;
  resource: string;
  action: string;
  type: PermissionType;
  key?: string;
  deptScope?: string;
  resourceTagScope?: string;
  envScope?: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface RolePermissionDetail {
  role: RoleItem;
  permissionIds: number[];
  permissions: PermissionItem[];
}
