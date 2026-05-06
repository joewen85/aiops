import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type { InAppMessageItem, MessageChannel, MessageCreatePayload } from "@/types/messages";

interface ListMessagesParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  channel?: MessageChannel | "";
  read?: "true" | "false" | "";
  module?: string;
  severity?: string;
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

export async function listMessages(params: ListMessagesParams = {}): Promise<PageData<InAppMessageItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    channel: params.channel,
    read: params.read,
    module: params.module,
    severity: params.severity,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<InAppMessageItem>>>(`/messages${query}`);
  return data.data;
}

export async function createMessage(payload: MessageCreatePayload): Promise<InAppMessageItem> {
  const { data } = await apiClient.post<ApiResponse<InAppMessageItem>>("/messages", payload);
  return data.data;
}

export async function markMessageRead(messageId: number): Promise<{
  id: number;
  traceId: string;
  read: boolean;
  readAt: string;
}> {
  const { data } = await apiClient.post<ApiResponse<{
    id: number;
    traceId: string;
    read: boolean;
    readAt: string;
  }>>(`/messages/${messageId}/read`);
  return data.data;
}
