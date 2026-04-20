import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  CmdbRelationItem,
  CmdbResourceItem,
  CmdbSource,
  CmdbSyncJob,
  CmdbSyncJobDetail,
} from "@/types/cmdb";

interface ListResourceParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  type?: string;
  cloud?: string;
  region?: string;
  env?: string;
  owner?: string;
}

interface ListRelationParams {
  page?: number;
  pageSize?: number;
  fromCiId?: string;
  toCiId?: string;
  relationType?: string;
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

export async function listCMDBResources(params: ListResourceParams = {}): Promise<PageData<CmdbResourceItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    type: params.type,
    cloud: params.cloud,
    region: params.region,
    env: params.env,
    owner: params.owner,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<CmdbResourceItem>>>(`/cmdb/resources${query}`);
  return data.data;
}

export async function createCMDBResource(payload: Partial<CmdbResourceItem>): Promise<{
  action: string;
  resource: CmdbResourceItem;
}> {
  const { data } = await apiClient.post<ApiResponse<{ action: string; resource: CmdbResourceItem }>>("/cmdb/resources", payload);
  return data.data;
}

export async function deleteCMDBResource(resourceId: number): Promise<void> {
  await apiClient.delete(`/cmdb/resources/${resourceId}`);
}

export async function listCMDBRelations(params: ListRelationParams = {}): Promise<PageData<CmdbRelationItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    fromCiId: params.fromCiId,
    toCiId: params.toCiId,
    relationType: params.relationType,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<CmdbRelationItem>>>(`/cmdb/relations${query}`);
  return data.data;
}

export async function createCMDBRelation(payload: {
  fromCiId: string;
  toCiId: string;
  relationType: string;
  direction?: string;
  criticality?: string;
  confidence?: number;
  evidence?: Record<string, unknown>;
}): Promise<{ action: string; relation: CmdbRelationItem }> {
  const { data } = await apiClient.post<ApiResponse<{ action: string; relation: CmdbRelationItem }>>("/cmdb/relations", payload);
  return data.data;
}

export async function createCMDBSyncJob(payload: {
  sources?: CmdbSource[];
  fullScan?: boolean;
}): Promise<CmdbSyncJob> {
  const { data } = await apiClient.post<ApiResponse<CmdbSyncJob>>("/cmdb/sync/jobs", payload);
  return data.data;
}

export async function getCMDBSyncJob(jobId: number): Promise<CmdbSyncJobDetail> {
  const { data } = await apiClient.get<ApiResponse<CmdbSyncJobDetail>>(`/cmdb/sync/jobs/${jobId}`);
  return data.data;
}

export async function retryCMDBSyncJob(jobId: number): Promise<CmdbSyncJob> {
  const { data } = await apiClient.post<ApiResponse<CmdbSyncJob>>(`/cmdb/sync/jobs/${jobId}/retry`);
  return data.data;
}

export async function getCMDBTopology(application: string, depth = 2): Promise<Record<string, unknown>> {
  const query = buildQuery({ depth });
  const { data } = await apiClient.get<ApiResponse<Record<string, unknown>>>(`/cmdb/topology/${encodeURIComponent(application)}${query}`);
  return data.data;
}

export async function getCMDBImpact(ciId: string, depth = 4): Promise<Record<string, unknown>> {
  const query = buildQuery({ depth });
  const { data } = await apiClient.get<ApiResponse<Record<string, unknown>>>(`/cmdb/impact/${encodeURIComponent(ciId)}${query}`);
  return data.data;
}

export async function getCMDBRegionFailover(region: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.get<ApiResponse<Record<string, unknown>>>(`/cmdb/regions/${encodeURIComponent(region)}/failover`);
  return data.data;
}

export async function getCMDBChangeImpact(releaseId: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.get<ApiResponse<Record<string, unknown>>>(`/cmdb/change-impact/${encodeURIComponent(releaseId)}`);
  return data.data;
}

