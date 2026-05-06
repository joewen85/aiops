export type TicketType =
  | "event"
  | "change"
  | "release"
  | "resource_request"
  | "permission_request"
  | "incident"
  | "service_request";

export type TicketStatus =
  | "draft"
  | "submitted"
  | "assigned"
  | "processing"
  | "pending_approval"
  | "approved"
  | "rejected"
  | "resolved"
  | "closed"
  | "cancelled";

export type TicketPriority = "P0" | "P1" | "P2" | "P3" | "P4";

export interface TicketItem {
  id: number;
  ticketNo: string;
  title: string;
  description?: string;
  type: TicketType | string;
  status: TicketStatus | string;
  priority: TicketPriority | string;
  severity?: TicketPriority | string;
  requesterId?: number;
  assigneeId?: number;
  departmentId?: number;
  env?: string;
  slaDueAt?: string;
  dueAt?: string;
  resolvedAt?: string;
  closedAt?: string;
  tags?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface TicketFlowItem {
  id: number;
  ticketId: number;
  fromStatus?: string;
  toStatus?: string;
  action: string;
  operatorId?: number;
  comment?: string;
  createdAt?: string;
}

export interface TicketApprovalItem {
  id: number;
  ticketId: number;
  nodeKey?: string;
  approverId?: number;
  approvalType?: string;
  status: string;
  comment?: string;
  approvedAt?: string;
  rejectedAt?: string;
  createdAt?: string;
}

export interface TicketCommentItem {
  id: number;
  ticketId: number;
  userId?: number;
  content: string;
  internal?: boolean;
  attachments?: Record<string, unknown>;
  createdAt?: string;
}

export interface TicketLinkItem {
  id: number;
  ticketId: number;
  linkType: string;
  linkId: string;
  linkName?: string;
  linkModule?: string;
  relation?: string;
  metadata?: Record<string, unknown>;
  createdAt?: string;
}

export interface TicketAttachmentItem {
  id: number;
  ticketId: number;
  fileName: string;
  fileSize?: number;
  contentType?: string;
  storageKey: string;
  uploaderId?: number;
  checksum?: string;
  createdAt?: string;
}

export interface TicketOperationItem {
  id: number;
  traceId: string;
  ticketId: number;
  module: string;
  action: string;
  dryRun: boolean;
  status: string;
  riskLevel?: string;
  request?: Record<string, unknown>;
  result?: Record<string, unknown>;
  errorMessage?: string;
  createdAt?: string;
}

export interface TicketTemplateItem {
  id: number;
  type: TicketType | string;
  name: string;
  description?: string;
  formSchema?: Record<string, unknown>;
  defaultPriority?: string;
  defaultFlow?: Record<string, unknown>;
  enabled?: boolean;
}

export interface TicketSummary {
  ticket: TicketItem;
  flows: TicketFlowItem[];
  approvals: TicketApprovalItem[];
  comments: TicketCommentItem[];
  links: TicketLinkItem[];
  attachments: TicketAttachmentItem[];
  operations: TicketOperationItem[];
}

export interface TicketAIOpsProtocol {
  protocolVersion: string;
  endpoints: Record<string, string>;
  types: string[];
  statuses: string[];
  priorities: string[];
  actions: Array<Record<string, unknown>>;
  requestSchema?: Record<string, unknown>;
  safety?: Record<string, unknown>;
}

export interface TicketOperationPayload {
  module: string;
  action: string;
  params?: Record<string, unknown>;
  confirmationText?: string;
}

export interface TicketOperationResult {
  protocolVersion: string;
  traceId: string;
  operation: TicketOperationItem;
  dryRun?: Record<string, unknown>;
}
