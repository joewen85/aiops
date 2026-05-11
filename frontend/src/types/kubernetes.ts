export interface KubernetesClusterItem {
  id: number;
  name: string;
  apiServer: string;
  credentialType?: string;
  kubeConfig?: string;
  token?: string;
  env?: string;
  region?: string;
  owner?: string;
  status?: string;
  version?: string;
  labels?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  lastCheckedAt?: string;
  lastSyncedAt?: string;
  certificateExpiresAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export type KubernetesKind = "Deployment" | "StatefulSet" | "DaemonSet" | "Pod" | "Node" | "Namespace" | "Service" | "Ingress" | "ConfigMap" | "Secret" | "PVC" | "PV";

export interface KubernetesResourceItem {
  id: number;
  clusterId: number;
  namespace?: string;
  kind: KubernetesKind | string;
  name: string;
  uid?: string;
  status?: string;
  specSummary?: string;
  resourceVersion?: string;
  labels?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  lastSyncedAt?: string;
}

export interface KubernetesOperationItem {
  id: number;
  traceId: string;
  clusterId: number;
  namespace?: string;
  kind: KubernetesKind | string;
  name: string;
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

export interface KubernetesAIOpsProtocol {
  protocolVersion: string;
  actionEndpoint: string;
  resources: Array<{
    kind: KubernetesKind | string;
    actions: string[];
    namespaceScoped: boolean;
  }>;
  requestSchema?: Record<string, unknown>;
  safety?: Record<string, unknown>;
}

export interface KubernetesActionPayload {
  clusterId: number;
  namespace?: string;
  kind: KubernetesKind | string;
  name: string;
  action: string;
  dryRun?: boolean;
  confirmationText?: string;
  params?: Record<string, unknown>;
}

export interface KubernetesActionResult {
  protocolVersion: string;
  traceId: string;
  operation: KubernetesOperationItem;
  dryRun?: Record<string, unknown>;
}

export interface KubernetesManifestPayload {
  clusterId?: number;
  namespace?: string;
  manifest: Record<string, unknown>;
  dryRun?: boolean;
  confirmationText?: string;
}

export interface KubernetesNodeRegisterPayload {
  clusterId: number;
  hostname: string;
  internalIp?: string;
  roles?: string[];
  cpu?: string;
  memory?: string;
  pods?: string;
  kubeletVersion?: string;
  labels?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export interface KubernetesNodeRegisterTaskPayload extends KubernetesNodeRegisterPayload {
  cloudAssetId?: number;
  joinCommand?: string;
  sshUser?: string;
  sshPassword?: string;
  sshPort?: number;
  dryRun?: boolean;
  executeNow?: boolean;
}
