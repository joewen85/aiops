import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  DepartmentListItem,
  DepartmentTreeNode,
  DepartmentUserBindingDetail,
  UserDepartmentBindingDetail,
  UserItem,
  UserRoleBindingDetail,
} from "@/types/users";

export async function listUsers(page = 1, pageSize = 10): Promise<PageData<UserItem>> {
  const { data } = await apiClient.get<ApiResponse<PageData<UserItem>>>(`/users?page=${page}&pageSize=${pageSize}`);
  return data.data;
}

export async function getUser(userId: number): Promise<UserItem> {
  const { data } = await apiClient.get<ApiResponse<UserItem>>(`/users/${userId}`);
  return data.data;
}

export async function createUser(payload: {
  username: string;
  password: string;
  displayName?: string;
  email?: string;
  isActive?: boolean;
  roleIds?: number[];
  departmentIds?: number[];
}): Promise<UserItem> {
  const { data } = await apiClient.post<ApiResponse<UserItem>>("/users", payload);
  return data.data;
}

export async function updateUser(userId: number, payload: {
  displayName?: string;
  email?: string;
  isActive?: boolean;
}): Promise<UserItem> {
  const { data } = await apiClient.put<ApiResponse<UserItem>>(`/users/${userId}`, payload);
  return data.data;
}

export async function deleteUser(userId: number): Promise<void> {
  await apiClient.delete(`/users/${userId}`);
}

export async function toggleUserStatus(userId: number, isActive: boolean): Promise<UserItem> {
  const { data } = await apiClient.patch<ApiResponse<UserItem>>(`/users/${userId}/status`, { isActive });
  return data.data;
}

export async function resetUserPassword(userId: number, password: string): Promise<void> {
  await apiClient.post(`/users/${userId}/reset-password`, { password });
}

export async function getUserRoles(userId: number): Promise<UserRoleBindingDetail> {
  const { data } = await apiClient.get<ApiResponse<UserRoleBindingDetail>>(`/users/${userId}/roles`);
  return data.data;
}

export async function bindUserRoles(userId: number, roleIds: number[]): Promise<void> {
  await apiClient.post(`/users/${userId}/roles`, { roleIds });
}

export async function getUserDepartments(userId: number): Promise<UserDepartmentBindingDetail> {
  const { data } = await apiClient.get<ApiResponse<UserDepartmentBindingDetail>>(`/users/${userId}/departments`);
  return data.data;
}

export async function bindUserDepartments(userId: number, departmentIds: number[]): Promise<void> {
  await apiClient.post(`/users/${userId}/departments`, { departmentIds });
}

export async function listDepartments(page = 1, pageSize = 10): Promise<PageData<DepartmentListItem>> {
  const { data } = await apiClient.get<ApiResponse<PageData<DepartmentListItem>>>(`/departments?page=${page}&pageSize=${pageSize}`);
  return data.data;
}

export async function listDepartmentTree(): Promise<DepartmentTreeNode[]> {
  const { data } = await apiClient.get<ApiResponse<DepartmentTreeNode[]>>("/departments/tree");
  return data.data;
}

export async function getDepartment(departmentId: number): Promise<DepartmentListItem> {
  const { data } = await apiClient.get<ApiResponse<DepartmentListItem>>(`/departments/${departmentId}`);
  return data.data;
}

export async function createDepartment(payload: {
  name: string;
  parentId?: number;
}): Promise<DepartmentListItem> {
  const { data } = await apiClient.post<ApiResponse<DepartmentListItem>>("/departments", payload);
  return data.data;
}

export async function updateDepartment(departmentId: number, payload: {
  name?: string;
  parentId?: number | null;
}): Promise<DepartmentListItem> {
  const { data } = await apiClient.put<ApiResponse<DepartmentListItem>>(`/departments/${departmentId}`, payload);
  return data.data;
}

export async function deleteDepartment(departmentId: number): Promise<void> {
  await apiClient.delete(`/departments/${departmentId}`);
}

export async function getDepartmentUsers(departmentId: number): Promise<DepartmentUserBindingDetail> {
  const { data } = await apiClient.get<ApiResponse<DepartmentUserBindingDetail>>(`/departments/${departmentId}/users`);
  return data.data;
}

export async function bindDepartmentUsers(departmentId: number, userIds: number[]): Promise<void> {
  await apiClient.post(`/departments/${departmentId}/users`, { userIds });
}
