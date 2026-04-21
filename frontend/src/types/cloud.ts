export type CloudProvider = "aws" | "aliyun" | "tencent" | "huawei" | string;

export type CloudAssetType =
  | "CloudServer"
  | "MySQL"
  | "PrivateNetwork"
  | "ObjectStorage"
  | "FileStorage"
  | "ContainerService"
  | "LoadBalancer"
  | "DNS"
  | "SSLCertificate"
  | "LogService"
  | "CloudResource"
  | string;

export interface CloudAccountItem {
  id: number;
  provider: CloudProvider;
  name: string;
  accessKey: string;
  secretKey: string;
  region?: string;
  isVerified?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface CloudAssetItem {
  id: number;
  provider: CloudProvider;
  accountId: number;
  region?: string;
  type: CloudAssetType;
  resourceId: string;
  name: string;
  status?: string;
  source?: string;
  tags?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  lastSyncedAt?: string;
  expiresAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CloudSyncJob {
  id: number;
  accountId: number;
  provider: string;
  region?: string;
  status: string;
  startedAt?: string;
  finishedAt?: string;
  summary?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
}

export interface CloudSyncResult {
  id: number;
  job?: CloudSyncJob;
  providerAssetCount?: number;
  cloudAssetCount?: number;
  cmdbAssetCount?: number;
  assets?: Array<Record<string, unknown>>;
  cloudAssets?: CloudAssetItem[];
  cloudAssetItems?: CloudAssetItem[];
  cmdbResources?: Array<Record<string, unknown>>;
  cmdbAssets?: Array<Record<string, unknown>>;
  cmdbAssetItems?: Array<Record<string, unknown>>;
  syncSummary?: Record<string, unknown>;
}
