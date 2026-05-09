import type { RoleItem } from "@/types/rbac";

export interface UserItem {
  id: number;
  username: string;
  displayName?: string;
  email?: string;
  isActive: boolean;
  roleIds?: number[];
  roles?: RoleItem[];
  departmentIds?: number[];
  departments?: DepartmentListItem[];
  createdAt?: string;
  updatedAt?: string;
}

export interface UserRoleBindingDetail {
  user: UserItem;
  roleIds: number[];
  roles: RoleItem[];
}

export interface UserDepartmentBindingDetail {
  user: UserItem;
  departmentIds: number[];
  departments: DepartmentListItem[];
}

export interface DepartmentUserBindingDetail {
  department: DepartmentListItem;
  userIds: number[];
  users: UserItem[];
}

export interface DepartmentListItem {
  id: number;
  name: string;
  parentId?: number | null;
  createdAt?: string;
  updatedAt?: string;
}

export interface DepartmentTreeNode {
  id: number;
  name: string;
  parentId?: number | null;
  children: DepartmentTreeNode[];
}
