import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  MiddlewareAIOpsProtocol,
  MiddlewareActionPayload,
  MiddlewareActionResult,
  MiddlewareInstanceItem,
  MiddlewareMetricItem,
  MiddlewareOperationItem,
} from "@/types/middleware";

interface ListMiddlewareInstanceParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  type?: string;
  env?: string;
  status?: string;
}

interface ListMiddlewareMetricParams {
  page?: number;
  pageSize?: number;
  metricType?: string;
}

interface ListMiddlewareOperationParams {
  page?: number;
  pageSize?: number;
  instanceId?: number;
  status?: string;
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

export async function listMiddlewareInstances(params: ListMiddlewareInstanceParams = {}): Promise<PageData<MiddlewareInstanceItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    type: params.type,
    env: params.env,
    status: params.status,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<MiddlewareInstanceItem>>>(`/middleware/instances${query}`);
  return data.data;
}

export async function createMiddlewareInstance(payload: Partial<MiddlewareInstanceItem> & Record<string, unknown>): Promise<MiddlewareInstanceItem> {
  const { data } = await apiClient.post<ApiResponse<MiddlewareInstanceItem>>("/middleware/instances", payload);
  return data.data;
}

export async function updateMiddlewareInstance(instanceId: number, payload: Partial<MiddlewareInstanceItem> & Record<string, unknown>): Promise<MiddlewareInstanceItem> {
  const { data } = await apiClient.put<ApiResponse<MiddlewareInstanceItem>>(`/middleware/instances/${instanceId}`, payload);
  return data.data;
}

export async function deleteMiddlewareInstance(instanceId: number, confirmationText: string): Promise<void> {
  await apiClient.delete(`/middleware/instances/${instanceId}`, { data: { confirmationText } });
}

export async function checkMiddlewareInstance(instanceId: number): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<ApiResponse<Record<string, unknown>>>(`/middleware/instances/${instanceId}/check`);
  return data.data;
}

export async function collectMiddlewareMetrics(instanceId: number): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<ApiResponse<Record<string, unknown>>>(`/middleware/instances/${instanceId}/metrics/collect`);
  return data.data;
}

export async function listMiddlewareMetrics(instanceId: number, params: ListMiddlewareMetricParams = {}): Promise<PageData<MiddlewareMetricItem>> {
  const query = buildQuery({ page: params.page ?? 1, pageSize: params.pageSize ?? 10, metricType: params.metricType });
  const { data } = await apiClient.get<ApiResponse<PageData<MiddlewareMetricItem>>>(`/middleware/instances/${instanceId}/metrics${query}`);
  return data.data;
}

export async function getMiddlewareAIOpsProtocol(): Promise<MiddlewareAIOpsProtocol> {
  const { data } = await apiClient.get<ApiResponse<MiddlewareAIOpsProtocol>>("/middleware/aiops/protocol");
  return data.data;
}

export async function runMiddlewareAction(payload: MiddlewareActionPayload): Promise<MiddlewareActionResult> {
  const { data } = await apiClient.post<ApiResponse<MiddlewareActionResult>>("/middleware/actions", payload);
  return data.data;
}

export async function listMiddlewareOperations(params: ListMiddlewareOperationParams = {}): Promise<PageData<MiddlewareOperationItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    instanceId: params.instanceId,
    status: params.status,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<MiddlewareOperationItem>>>(`/middleware/operations${query}`);
  return data.data;
}
