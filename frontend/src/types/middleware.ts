export type MiddlewareType = "redis" | "postgresql" | "rabbitmq";

export interface MiddlewareInstanceItem {
  id: number;
  name: string;
  type: MiddlewareType | string;
  endpoint: string;
  healthPath?: string;
  env?: string;
  owner?: string;
  authType?: string;
  tlsEnable: boolean;
  status?: string;
  version?: string;
  labels?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  lastCheckedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MiddlewareMetricItem {
  id: number;
  instanceId: number;
  metricType: string;
  value: number;
  unit?: string;
  data?: Record<string, unknown>;
  collectedAt: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MiddlewareOperationItem {
  id: number;
  traceId: string;
  instanceId: number;
  type: MiddlewareType | string;
  action: string;
  status: string;
  dryRun: boolean;
  riskLevel?: string;
  request?: Record<string, unknown>;
  result?: Record<string, unknown>;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface MiddlewareActionSpec {
  name: string;
  description: string;
  riskLevel: string;
  confirmationRequired?: boolean;
  params?: Record<string, unknown>;
}

export interface MiddlewareAIOpsProtocol {
  protocolVersion: string;
  actionEndpoint: string;
  supportedTypes: string[];
  resources: Array<{
    type: MiddlewareType | string;
    actions: MiddlewareActionSpec[];
  }>;
  requestSchema?: Record<string, unknown>;
  safety?: Record<string, unknown>;
}

export interface MiddlewareActionPayload {
  instanceId: number;
  type?: MiddlewareType | string;
  action: string;
  dryRun?: boolean;
  confirmationText?: string;
  params?: Record<string, unknown>;
}

export interface MiddlewareActionResult {
  protocolVersion: string;
  traceId: string;
  operation: MiddlewareOperationItem;
  dryRun?: Record<string, unknown>;
}
