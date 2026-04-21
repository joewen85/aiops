import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type { PermissionItem, RoleItem, RolePermissionDetail } from "@/types/rbac";

interface RoleListQuery {
  keyword?: string;
  builtIn?: string;
}

interface PermissionListQuery {
  keyword?: string;
  type?: string;
}

export async function listRoles(page = 1, pageSize = 10, query: RoleListQuery = {}): Promise<PageData<RoleItem>> {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  if (query.builtIn?.trim()) params.set("builtIn", query.builtIn.trim());
  const { data } = await apiClient.get<ApiResponse<PageData<RoleItem>>>(`/roles?${params.toString()}`);
  return data.data;
}

export async function createRole(payload: { name: string; description?: string }): Promise<RoleItem> {
  const { data } = await apiClient.post<ApiResponse<RoleItem>>("/roles", payload);
  return data.data;
}

export async function getRole(roleId: number): Promise<RoleItem> {
  const { data } = await apiClient.get<ApiResponse<RoleItem>>(`/roles/${roleId}`);
  return data.data;
}

export async function updateRole(roleId: number, payload: { name?: string; description?: string }): Promise<RoleItem> {
  const { data } = await apiClient.put<ApiResponse<RoleItem>>(`/roles/${roleId}`, payload);
  return data.data;
}

export async function deleteRole(roleId: number): Promise<void> {
  await apiClient.delete(`/roles/${roleId}`);
}

export async function listPermissions(page = 1, pageSize = 10, query: PermissionListQuery = {}): Promise<PageData<PermissionItem>> {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  if (query.type?.trim()) params.set("type", query.type.trim());
  const { data } = await apiClient.get<ApiResponse<PageData<PermissionItem>>>(`/permissions?${params.toString()}`);
  return data.data;
}

export async function createPermission(payload: Partial<PermissionItem>): Promise<PermissionItem> {
  const { data } = await apiClient.post<ApiResponse<PermissionItem>>("/permissions", payload);
  return data.data;
}

export async function getPermission(permissionId: number): Promise<PermissionItem> {
  const { data } = await apiClient.get<ApiResponse<PermissionItem>>(`/permissions/${permissionId}`);
  return data.data;
}

export async function updatePermission(permissionId: number, payload: Partial<PermissionItem>): Promise<PermissionItem> {
  const { data } = await apiClient.put<ApiResponse<PermissionItem>>(`/permissions/${permissionId}`, payload);
  return data.data;
}

export async function deletePermission(permissionId: number): Promise<void> {
  await apiClient.delete(`/permissions/${permissionId}`);
}

export async function getRolePermissions(roleId: number): Promise<RolePermissionDetail> {
  const { data } = await apiClient.get<ApiResponse<RolePermissionDetail>>(`/roles/${roleId}/permissions`);
  return data.data;
}

export async function bindRolePermissions(roleId: number, permissionIds: number[]): Promise<void> {
  await apiClient.post(`/roles/${roleId}/permissions`, { permissionIds });
}
