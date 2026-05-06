export type MessageChannel = "broadcast" | "user" | "role" | "department";

export interface InAppMessageItem {
  id: number;
  traceId: string;
  channel: MessageChannel | string;
  target: string;
  title?: string;
  content: string;
  module?: string;
  source?: string;
  event?: string;
  severity?: string;
  resourceType?: string;
  resourceId?: string;
  data?: Record<string, unknown>;
  read: boolean;
  readAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MessageCreatePayload {
  channel: MessageChannel;
  target?: string;
  title?: string;
  content: string;
  module?: string;
  source?: string;
  event?: string;
  severity?: string;
  resourceType?: string;
  resourceId?: string;
  data?: Record<string, unknown>;
}
