import { apiClient } from "@/api/client";
import type { ApiResponse, PageData } from "@/api/types";
import type {
  TicketAIOpsProtocol,
  TicketCommentItem,
  TicketItem,
  TicketLinkItem,
  TicketOperationPayload,
  TicketOperationResult,
  TicketSummary,
  TicketTemplateItem,
} from "@/types/tickets";

interface ListTicketsParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  type?: string;
  status?: string;
  priority?: string;
  env?: string;
  assigneeId?: number;
  requesterId?: number;
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

export async function listTickets(params: ListTicketsParams = {}): Promise<PageData<TicketItem>> {
  const query = buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 10,
    keyword: params.keyword,
    type: params.type,
    status: params.status,
    priority: params.priority,
    env: params.env,
    assigneeId: params.assigneeId,
    requesterId: params.requesterId,
  });
  const { data } = await apiClient.get<ApiResponse<PageData<TicketItem>>>(`/tickets${query}`);
  return data.data;
}

export async function getTicket(ticketId: number): Promise<TicketSummary> {
  const { data } = await apiClient.get<ApiResponse<TicketSummary>>(`/tickets/${ticketId}`);
  return data.data;
}

export async function createTicket(payload: Partial<TicketItem> & Record<string, unknown>): Promise<TicketItem> {
  const { data } = await apiClient.post<ApiResponse<TicketItem>>("/tickets", payload);
  return data.data;
}

export async function updateTicket(ticketId: number, payload: Partial<TicketItem> & Record<string, unknown>): Promise<TicketItem> {
  const { data } = await apiClient.put<ApiResponse<TicketItem>>(`/tickets/${ticketId}`, payload);
  return data.data;
}

export async function deleteTicket(ticketId: number, confirmationText: string): Promise<void> {
  await apiClient.delete(`/tickets/${ticketId}`, { data: { confirmationText } });
}

export async function submitTicket(ticketId: number): Promise<TicketItem> {
  const { data } = await apiClient.post<ApiResponse<TicketItem>>(`/tickets/${ticketId}/submit`);
  return data.data;
}

export async function cancelTicket(ticketId: number): Promise<TicketItem> {
  const { data } = await apiClient.post<ApiResponse<TicketItem>>(`/tickets/${ticketId}/cancel`);
  return data.data;
}

export async function reopenTicket(ticketId: number): Promise<TicketItem> {
  const { data } = await apiClient.post<ApiResponse<TicketItem>>(`/tickets/${ticketId}/reopen`);
  return data.data;
}

export async function transitionTicket(ticketId: number, status: string, comment?: string): Promise<TicketItem> {
  const { data } = await apiClient.post<ApiResponse<TicketItem>>(`/tickets/${ticketId}/transition`, { status, comment });
  return data.data;
}

export async function approveTicket(ticketId: number, comment?: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<ApiResponse<Record<string, unknown>>>(`/tickets/${ticketId}/approve`, { comment });
  return data.data;
}

export async function rejectTicket(ticketId: number, comment?: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.post<ApiResponse<Record<string, unknown>>>(`/tickets/${ticketId}/reject`, { comment });
  return data.data;
}

export async function createTicketComment(ticketId: number, payload: { content: string; internal?: boolean; attachments?: Record<string, unknown> }): Promise<TicketCommentItem> {
  const { data } = await apiClient.post<ApiResponse<TicketCommentItem>>(`/tickets/${ticketId}/comments`, payload);
  return data.data;
}

export async function createTicketLink(ticketId: number, payload: Partial<TicketLinkItem>): Promise<TicketLinkItem> {
  const { data } = await apiClient.post<ApiResponse<TicketLinkItem>>(`/tickets/${ticketId}/links`, payload);
  return data.data;
}

export async function listTicketTemplates(): Promise<PageData<TicketTemplateItem>> {
  const { data } = await apiClient.get<ApiResponse<PageData<TicketTemplateItem>>>("/ticket-templates?page=1&pageSize=100&seed=1");
  return data.data;
}

export async function getTicketAIOpsProtocol(): Promise<TicketAIOpsProtocol> {
  const { data } = await apiClient.get<ApiResponse<TicketAIOpsProtocol>>("/tickets/aiops/protocol");
  return data.data;
}

export async function dryRunTicketOperation(ticketId: number, payload: TicketOperationPayload): Promise<TicketOperationResult> {
  const { data } = await apiClient.post<ApiResponse<TicketOperationResult>>(`/tickets/${ticketId}/operations/dry-run`, payload);
  return data.data;
}

export async function executeTicketOperation(ticketId: number, payload: TicketOperationPayload): Promise<TicketOperationResult> {
  const { data } = await apiClient.post<ApiResponse<TicketOperationResult>>(`/tickets/${ticketId}/operations/execute`, payload);
  return data.data;
}
