import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  DockerAIOpsProtocol,
  DockerActionPayload,
  DockerActionResult,
  DockerComposeStackItem,
  DockerHostItem,
  DockerOperationItem,
  DockerResourceItem,
} from "@/types/docker";

interface ListDockerHostParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  env?: string;
  status?: string;
  tls?: string;
}

interface ListDockerResourceParams {
  page?: number;
  pageSize?: number;
  type?: string;
  keyword?: string;
}

interface ListComposeStackParams {
  page?: number;
  pageSize?: number;
  hostId?: number;
  status?: string;
  keyword?: string;
}

interface ListDockerOperationParams {
  page?: number;
  pageSize?: number;
  hostId?: number;
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

export async function listDockerHosts(params: ListDockerHostParams = {}): Promise<PageData<DockerHostItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    env: params.env,
    status: params.status,
    tls: params.tls,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<DockerHostItem>>>(`/docker/hosts${query}`);
  return data.data;
}

export async function createDockerHost(payload: Partial<DockerHostItem>): Promise<DockerHostItem> {
  const { data } = await apiClient.post<ApiResponse<DockerHostItem>>("/docker/hosts", payload);
  return data.data;
}

export async function updateDockerHost(hostId: number, payload: Partial<DockerHostItem>): Promise<DockerHostItem> {
  const { data } = await apiClient.put<ApiResponse<DockerHostItem>>(`/docker/hosts/${hostId}`, payload);
  return data.data;
}

export async function deleteDockerHost(hostId: number): Promise<void> {
  await apiClient.delete(`/docker/hosts/${hostId}`);
}

export async function checkDockerHost(hostId: number): Promise<{ id: number; status: string; version?: string }> {
  const { data } = await apiClient.post<ApiResponse<{ id: number; status: string; version?: string }>>(`/docker/hosts/${hostId}/check`);
  return data.data;
}

export async function listDockerHostResources(hostId: number, params: ListDockerResourceParams): Promise<PageData<DockerResourceItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    type: params.type ?? "container",
    keyword: params.keyword,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<DockerResourceItem>>>(`/docker/hosts/${hostId}/resources${query}`);
  return data.data;
}

export async function listComposeStacks(params: ListComposeStackParams = {}): Promise<PageData<DockerComposeStackItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    hostId: params.hostId,
    status: params.status,
    keyword: params.keyword,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<DockerComposeStackItem>>>(`/docker/compose/stacks${query}`);
  return data.data;
}

export async function createComposeStack(payload: Partial<DockerComposeStackItem>): Promise<DockerComposeStackItem> {
  const { data } = await apiClient.post<ApiResponse<DockerComposeStackItem>>("/docker/compose/stacks", payload);
  return data.data;
}

export async function updateComposeStack(stackId: number, payload: Partial<DockerComposeStackItem>): Promise<DockerComposeStackItem> {
  const { data } = await apiClient.put<ApiResponse<DockerComposeStackItem>>(`/docker/compose/stacks/${stackId}`, payload);
  return data.data;
}

export async function deleteComposeStack(stackId: number): Promise<void> {
  await apiClient.delete(`/docker/compose/stacks/${stackId}`);
}

export async function getDockerAIOpsProtocol(): Promise<DockerAIOpsProtocol> {
  const { data } = await apiClient.get<ApiResponse<DockerAIOpsProtocol>>("/docker/aiops/protocol");
  return data.data;
}

export async function runDockerAction(payload: DockerActionPayload): Promise<DockerActionResult> {
  const { data } = await apiClient.post<ApiResponse<DockerActionResult>>("/docker/actions", payload);
  return data.data;
}

export async function listDockerOperations(params: ListDockerOperationParams = {}): Promise<PageData<DockerOperationItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    hostId: params.hostId,
    status: params.status,
    action: params.action,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<DockerOperationItem>>>(`/docker/operations${query}`);
  return data.data;
}
