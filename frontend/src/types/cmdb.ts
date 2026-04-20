export type CmdbSource = "IaC" | "CloudAPI" | "K8s" | "APM" | "Manual";

export interface CmdbResourceItem {
  id: number;
  ciId: string;
  type: string;
  name: string;
  categoryId?: number;
  cloud?: string;
  region?: string;
  env?: string;
  owner?: string;
  lifecycle?: string;
  source?: CmdbSource | string;
  lastSeenAt?: string;
  attributes?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface CmdbRelationItem {
  id: number;
  fromCiId: string;
  toCiId: string;
  relationType: string;
  direction?: "outbound" | "inbound" | "bidirectional" | string;
  criticality?: "P0" | "P1" | "P2" | "P3" | string;
  confidence?: number;
  evidence?: Record<string, unknown>;
  relationUpdatedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CmdbSyncJob {
  id: number;
  status: "pending" | "running" | "success" | "failed" | string;
  requestedSources?: string[] | string;
  fullScan?: boolean;
  startedAt?: string;
  finishedAt?: string;
  summary?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface CmdbSyncJobItem {
  id: number;
  jobId: number;
  ciId: string;
  source: string;
  action: string;
  status: string;
  message?: string;
  qualityScore?: number;
  data?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface CmdbSyncJobDetail {
  job: CmdbSyncJob;
  items: CmdbSyncJobItem[];
}

