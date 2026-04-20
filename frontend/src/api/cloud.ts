import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type { CloudAccountItem, CloudAssetItem, CloudSyncResult } from "@/types/cloud";

interface ListCloudAccountParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  provider?: string;
  region?: string;
  verified?: string;
}

interface ListCloudAssetParams {
  page?: number;
  pageSize?: number;
  provider?: string;
  accountId?: number;
  region?: string;
  type?: string;
  status?: string;
  source?: string;
  keyword?: string;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined) return;
    const text = String(value).trim();
    if (!text) return;
    searchParams.set(key, text);
  });
  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

export async function listCloudAccounts(params: ListCloudAccountParams = {}): Promise<PageData<CloudAccountItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    provider: params.provider,
    region: params.region,
    verified: params.verified,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<CloudAccountItem>>>(`/cloud/accounts${query}`);
  return data.data;
}

export async function createCloudAccount(payload: Partial<CloudAccountItem>): Promise<CloudAccountItem> {
  const { data } = await apiClient.post<ApiResponse<CloudAccountItem>>("/cloud/accounts", payload);
  return data.data;
}

export async function updateCloudAccount(accountId: number, payload: Partial<CloudAccountItem>): Promise<CloudAccountItem> {
  const { data } = await apiClient.put<ApiResponse<CloudAccountItem>>(`/cloud/accounts/${accountId}`, payload);
  return data.data;
}

export async function deleteCloudAccount(accountId: number): Promise<void> {
  await apiClient.delete(`/cloud/accounts/${accountId}`);
}

export async function verifyCloudAccount(accountId: number): Promise<{ id: number; verified: boolean }> {
  const { data } = await apiClient.post<ApiResponse<{ id: number; verified: boolean }>>(`/cloud/accounts/${accountId}/verify`);
  return data.data;
}

export async function syncCloudAccount(accountId: number): Promise<CloudSyncResult> {
  const { data } = await apiClient.post<ApiResponse<CloudSyncResult>>(`/cloud/accounts/${accountId}/sync`);
  return data.data;
}

export async function listCloudAssets(params: ListCloudAssetParams = {}): Promise<PageData<CloudAssetItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    provider: params.provider,
    accountId: params.accountId,
    region: params.region,
    type: params.type,
    status: params.status,
    source: params.source,
    keyword: params.keyword,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<CloudAssetItem>>>(`/cloud/assets${query}`);
  return data.data;
}

export async function listCloudAccountAssets(accountId: number, params: Pick<ListCloudAssetParams, "page" | "pageSize" | "region" | "type"> = {}): Promise<PageData<CloudAssetItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    region: params.region,
    type: params.type,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<CloudAssetItem>>>(`/cloud/accounts/${accountId}/assets${query}`);
  return data.data;
}

export async function createCloudAsset(payload: Partial<CloudAssetItem>): Promise<{ action: string; asset: CloudAssetItem }> {
  const { data } = await apiClient.post<ApiResponse<{ action: string; asset: CloudAssetItem }>>("/cloud/assets", payload);
  return data.data;
}

export async function updateCloudAsset(assetId: number, payload: Partial<CloudAssetItem>): Promise<CloudAssetItem> {
  const { data } = await apiClient.put<ApiResponse<CloudAssetItem>>(`/cloud/assets/${assetId}`, payload);
  return data.data;
}

export async function deleteCloudAsset(assetId: number): Promise<void> {
  await apiClient.delete(`/cloud/assets/${assetId}`);
}
