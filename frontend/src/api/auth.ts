import { apiClient } from "@/api/client";
import type { ApiResponse } from "@/api/types";
import type { PermissionBundle } from "@/types/permission";

export async function fetchMyPermissions(): Promise<PermissionBundle> {
  const { data } = await apiClient.get<ApiResponse<PermissionBundle>>("/auth/me/permissions?compact=1");
  return {
    ...data.data,
    permissions: data.data.permissions ?? [],
  };
}
