import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  KubernetesAIOpsProtocol,
  KubernetesActionPayload,
  KubernetesActionResult,
  KubernetesClusterItem,
  KubernetesManifestPayload,
  KubernetesNodeRegisterPayload,
  KubernetesNodeRegisterTaskPayload,
  KubernetesOperationItem,
  KubernetesResourceItem,
} from "@/types/kubernetes";

interface ListKubernetesClusterParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  env?: string;
  status?: string;
}

interface ListKubernetesResourceParams {
  page?: number;
  pageSize?: number;
  clusterId?: number;
  namespace?: string;
  kind?: string;
  status?: string;
  keyword?: string;
}

interface ListKubernetesOperationParams {
  page?: number;
  pageSize?: number;
  clusterId?: number;
  status?: string;
  action?: string;
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

export async function listKubernetesClusters(params: ListKubernetesClusterParams = {}): Promise<PageData<KubernetesClusterItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    env: params.env,
    status: params.status,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<KubernetesClusterItem>>>(`/kubernetes/clusters${query}`);
  return data.data;
}

export async function createKubernetesCluster(payload: Partial<KubernetesClusterItem>): Promise<KubernetesClusterItem> {
  const { data } = await apiClient.post<ApiResponse<KubernetesClusterItem>>("/kubernetes/clusters", payload);
  return data.data;
}

export async function updateKubernetesCluster(clusterId: number, payload: Partial<KubernetesClusterItem>): Promise<KubernetesClusterItem> {
  const { data } = await apiClient.put<ApiResponse<KubernetesClusterItem>>(`/kubernetes/clusters/${clusterId}`, payload);
  return data.data;
}

export async function deleteKubernetesCluster(clusterId: number): Promise<void> {
  await apiClient.delete(`/kubernetes/clusters/${clusterId}`, { data: { confirmationText: "确认删除资源" } });
}

export async function checkKubernetesCluster(clusterId: number): Promise<{ id: number; status: string; version?: string }> {
  const { data } = await apiClient.post<ApiResponse<{ id: number; status: string; version?: string }>>(`/kubernetes/clusters/${clusterId}/check`);
  return data.data;
}

export async function syncKubernetesCluster(clusterId: number): Promise<{ clusterId: number; status: string; count: number; warnings?: string[] }> {
  const { data } = await apiClient.post<ApiResponse<{ clusterId: number; status: string; count: number; warnings?: string[] }>>(`/kubernetes/clusters/${clusterId}/sync`);
  return data.data;
}

export async function listKubernetesResources(params: ListKubernetesResourceParams = {}): Promise<PageData<KubernetesResourceItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    clusterId: params.clusterId,
    namespace: params.namespace,
    kind: params.kind,
    status: params.status,
    keyword: params.keyword,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<KubernetesResourceItem>>>(`/kubernetes/resources${query}`);
  return data.data;
}

export async function listKubernetesNodes(params: ListKubernetesResourceParams = {}): Promise<PageData<KubernetesResourceItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    clusterId: params.clusterId,
    status: params.status,
    keyword: params.keyword,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<KubernetesResourceItem>>>(`/kubernetes/nodes${query}`);
  return data.data;
}

export async function getKubernetesResourceManifest(resourceId: number): Promise<{ clusterId: number; manifest: Record<string, unknown>; source: string; resourceVersion?: string }> {
  const { data } = await apiClient.get<ApiResponse<{ clusterId: number; manifest: Record<string, unknown>; source: string; resourceVersion?: string }>>(`/kubernetes/resources/${resourceId}/manifest`);
  return data.data;
}

export async function listKubernetesOperations(params: ListKubernetesOperationParams = {}): Promise<PageData<KubernetesOperationItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    clusterId: params.clusterId,
    status: params.status,
    action: params.action,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<KubernetesOperationItem>>>(`/kubernetes/operations${query}`);
  return data.data;
}

export async function getKubernetesAIOpsProtocol(): Promise<KubernetesAIOpsProtocol> {
  const { data } = await apiClient.get<ApiResponse<KubernetesAIOpsProtocol>>("/kubernetes/aiops/protocol");
  return data.data;
}

export async function runKubernetesAction(payload: KubernetesActionPayload): Promise<KubernetesActionResult> {
  const endpoint = payload.dryRun === false ? "/kubernetes/actions" : "/kubernetes/actions/dry-run";
  const { data } = await apiClient.post<ApiResponse<KubernetesActionResult>>(endpoint, payload);
  return data.data;
}

export async function createKubernetesResource(payload: KubernetesManifestPayload): Promise<KubernetesActionResult> {
  const { data } = await apiClient.post<ApiResponse<KubernetesActionResult>>("/kubernetes/resources", payload);
  return data.data;
}

export async function updateKubernetesResource(resourceId: number, payload: KubernetesManifestPayload): Promise<KubernetesActionResult> {
  const { data } = await apiClient.put<ApiResponse<KubernetesActionResult>>(`/kubernetes/resources/${resourceId}`, payload);
  return data.data;
}

export async function deleteKubernetesResource(resourceId: number): Promise<KubernetesActionResult> {
  const { data } = await apiClient.delete<ApiResponse<KubernetesActionResult>>(`/kubernetes/resources/${resourceId}`, { data: { dryRun: false, confirmationText: "确认删除资源", manifest: {} } });
  return data.data;
}

export async function registerKubernetesNode(payload: KubernetesNodeRegisterPayload): Promise<{ traceId: string; operation: KubernetesOperationItem; node: KubernetesResourceItem }> {
  const { data } = await apiClient.post<ApiResponse<{ traceId: string; operation: KubernetesOperationItem; node: KubernetesResourceItem }>>("/kubernetes/nodes/register", payload);
  return data.data;
}

export async function registerKubernetesNodeTask(payload: KubernetesNodeRegisterTaskPayload): Promise<{ traceId: string; taskId: number; playbookId: number; operation: KubernetesOperationItem; node: KubernetesResourceItem; dryRun: boolean; executeNow: boolean }> {
  const { data } = await apiClient.post<ApiResponse<{ traceId: string; taskId: number; playbookId: number; operation: KubernetesOperationItem; node: KubernetesResourceItem; dryRun: boolean; executeNow: boolean }>>("/kubernetes/nodes/register/task", payload);
  return data.data;
}
