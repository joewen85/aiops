export interface DockerHostItem {
  id: number;
  name: string;
  endpoint: string;
  tlsEnable: boolean;
  env?: string;
  owner?: string;
  status?: string;
  version?: string;
  labels?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  lastHeartbeatAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface DockerComposeStackItem {
  id: number;
  hostId: number;
  name: string;
  status?: string;
  services?: number;
  content: string;
  lastDeployedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export type DockerResourceType = "container" | "image" | "network" | "volume" | "compose";

export interface DockerResourceItem {
  id: string;
  name: string;
  type: DockerResourceType | string;
  status?: string;
  image?: string;
  driver?: string;
  size?: number;
  raw?: Record<string, unknown>;
  aiopsActions?: string[];
}

export interface DockerOperationItem {
  id: number;
  traceId: string;
  hostId: number;
  resourceType: DockerResourceType | string;
  resourceId?: string;
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

export interface DockerAIOpsProtocol {
  protocolVersion: string;
  actionEndpoint: string;
  resources: Array<{
    type: DockerResourceType | string;
    actions: string[];
    idField: string;
    dryRunOnly?: boolean;
  }>;
  requestSchema?: Record<string, unknown>;
  safety?: Record<string, unknown>;
}

export interface DockerActionPayload {
  hostId: number;
  resourceType: DockerResourceType | string;
  resourceId?: string;
  action: string;
  dryRun: boolean;
  confirmationText?: string;
  params?: Record<string, unknown>;
}

export interface DockerActionResult {
  protocolVersion: string;
  traceId: string;
  operation: DockerOperationItem;
  dryRun?: Record<string, unknown>;
}
