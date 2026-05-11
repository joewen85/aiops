import { FormEvent, useEffect, useMemo, useState } from "react";

import type { PageData } from "@/api/types";
import {
  checkKubernetesCluster,
  createKubernetesCluster,
  createKubernetesResource,
  deleteKubernetesCluster,
  deleteKubernetesResource,
  getKubernetesAIOpsProtocol,
  listKubernetesNodes,
  getKubernetesResourceManifest,
  listKubernetesClusters,
  listKubernetesOperations,
  listKubernetesResources,
  registerKubernetesNode,
  registerKubernetesNodeTask,
  runKubernetesAction,
  syncKubernetesCluster,
  updateKubernetesCluster,
  updateKubernetesResource,
} from "@/api/kubernetes";
import { listCloudAssets } from "@/api/cloud";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { Pagination } from "@/components/Pagination";
import { PermissionButton } from "@/components/PermissionButton";
import { ListRowActions } from "@/components/RowActionOverflow";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type {
  KubernetesAIOpsProtocol,
  KubernetesActionPayload,
  KubernetesClusterItem,
  KubernetesManifestPayload,
  KubernetesNodeRegisterPayload,
  KubernetesNodeRegisterTaskPayload,
  KubernetesOperationItem,
  KubernetesResourceItem,
} from "@/types/kubernetes";
import type { CloudAssetItem } from "@/types/cloud";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const kubernetesConfirmDeleteText = "确认删除资源";
const kubernetesConfirmSubmitText = "确认提交资源";
const clusterSettingsKey = "kubernetes.clusters.table.settings";
const resourceSettingsKey = "kubernetes.resources.table.settings";
const operationSettingsKey = "kubernetes.operations.table.settings";
const nodePageSizeOptions = [10, 20, 50];
const resourceKinds = ["Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob", "Node", "Namespace", "Service", "Ingress", "ConfigMap", "Secret", "PVC", "PV"];
const resourceKindFilterOptions = ["", ...resourceKinds];
const resourceManifestTemplateKeys = ["deployment", "statefulSet", "daemonSet", "pod", "job", "cronJob", "service", "ingress", "configMap", "secret", "pvc", "namespace", "storageClass", "hpa", "customResource"] as const;

const clusterColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "name", label: "集群名称" },
  { key: "apiServer", label: "API Server" },
  { key: "env", label: "环境" },
  { key: "region", label: "区域" },
  { key: "owner", label: "负责人" },
  { key: "status", label: "状态" },
  { key: "version", label: "版本" },
  { key: "lastCheckedAt", label: "校验时间" },
  { key: "lastSyncedAt", label: "同步时间" },
  { key: "actions", label: "操作", required: true },
];
const resourceColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "clusterId", label: "集群" },
  { key: "namespace", label: "命名空间" },
  { key: "kind", label: "类型" },
  { key: "name", label: "名称" },
  { key: "status", label: "状态" },
  { key: "specSummary", label: "配置摘要" },
  { key: "lastSyncedAt", label: "同步时间" },
  { key: "actions", label: "操作", required: true },
];
const nodeTableColumns = ["name", "status", "specSummary", "lastSyncedAt", "actions"] as const;
const operationColumns: TableSettingsColumn[] = [
  { key: "traceId", label: "TraceID" },
  { key: "clusterId", label: "集群" },
  { key: "namespace", label: "命名空间" },
  { key: "kind", label: "类型" },
  { key: "name", label: "资源" },
  { key: "action", label: "动作" },
  { key: "status", label: "状态" },
  { key: "dryRun", label: "DryRun" },
  { key: "riskLevel", label: "风险" },
  { key: "createdAt", label: "创建时间" },
  { key: "actions", label: "操作", required: true },
];
const defaultClusterColumns = ["id", "name", "apiServer", "env", "owner", "status", "version", "actions"];
const defaultResourceColumns = ["clusterId", "namespace", "kind", "name", "status", "specSummary", "actions"];
const defaultOperationColumns = ["traceId", "clusterId", "kind", "name", "action", "status", "dryRun", "riskLevel", "actions"];

type DrawerState = "closed" | "cluster-create" | "cluster-edit" | "resource-create" | "resource-edit" | "node-register";
type SettingsTarget = "closed" | "clusters" | "resources" | "operations";
type ResourceManifestTemplateKey = (typeof resourceManifestTemplateKeys)[number] | "current";

interface ClusterFilter {
  keyword: string;
  env: string;
  status: string;
}

interface ResourceFilter {
  kind: string;
  namespace: string;
  status: string;
  keyword: string;
}

interface ClusterForm {
  name: string;
  apiServer: string;
  credentialType: string;
  kubeConfig: string;
  token: string;
  env: string;
  region: string;
  owner: string;
  labelsJSON: string;
  metadataJSON: string;
}

interface ResourceManifestForm {
  templateKey: ResourceManifestTemplateKey;
  namespace: string;
  manifestJSON: string;
}

interface PendingKubernetesAction {
  title: string;
  description: string;
  payload: KubernetesActionPayload;
  actionKey: string;
}

interface PendingManifestSubmit {
  mode: "create" | "update";
  resourceId?: number;
  payload: KubernetesManifestPayload;
}

interface NodeFilter {
  keyword: string;
  status: string;
}

interface NodeRegisterForm {
  sourceType: "manual" | "cloudAsset";
  executionMode: "direct" | "task";
  executeNow: boolean;
  cloudAssetId: string;
  hostname: string;
  internalIp: string;
  rolesCSV: string;
  cpu: string;
  memory: string;
  pods: string;
  kubeletVersion: string;
  joinCommand: string;
  sshUser: string;
  sshPassword: string;
  sshPort: string;
  dryRunTask: boolean;
  labelsJSON: string;
  metadataJSON: string;
}

function defaultClusterFilter(): ClusterFilter {
  return { keyword: "", env: "", status: "" };
}

function defaultResourceFilter(): ResourceFilter {
  return { kind: "", namespace: "", status: "", keyword: "" };
}

function defaultNodeFilter(): NodeFilter {
  return { keyword: "", status: "" };
}

function defaultClusterForm(): ClusterForm {
  return {
    name: "",
    apiServer: "mock://dev",
    credentialType: "kubeconfig",
    kubeConfig: "apiVersion: v1\nclusters: []\ncontexts: []\ncurrent-context: \"\"\nkind: Config\npreferences: {}\nusers: []\n",
    token: "",
    env: "dev",
    region: "default",
    owner: "",
    labelsJSON: JSON.stringify({ platform: "kubernetes", managedBy: "aiops" }, null, 2),
    metadataJSON: JSON.stringify({ version: "v1.29.3", aiopsEnabled: true }, null, 2),
  };
}

function defaultNodeRegisterForm(): NodeRegisterForm {
  return {
    sourceType: "manual",
    executionMode: "task",
    executeNow: true,
    cloudAssetId: "",
    hostname: "",
    internalIp: "",
    rolesCSV: "worker",
    cpu: "4",
    memory: "16Gi",
    pods: "110",
    kubeletVersion: "",
    joinCommand: "",
    sshUser: "root",
    sshPassword: "",
    sshPort: "22",
    dryRunTask: false,
    labelsJSON: JSON.stringify({ managedBy: "aiops" }, null, 2),
    metadataJSON: JSON.stringify({ source: "manual-register" }, null, 2),
  };
}

const resourceManifestTemplates: Record<Exclude<ResourceManifestTemplateKey, "current">, { label: string; namespace: string; manifest: (clusterName: string) => Record<string, unknown> }> = {
  deployment: {
    label: "Deployment 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "apps/v1",
      kind: "Deployment",
      metadata: { name: `${clusterName}-web`, namespace: "default", labels: { app: `${clusterName}-web`, managedBy: "aiops" } },
      spec: {
        replicas: 2,
        selector: { matchLabels: { app: `${clusterName}-web` } },
        template: { metadata: { labels: { app: `${clusterName}-web` } }, spec: { containers: [{ name: "web", image: "nginx:latest", ports: [{ containerPort: 80 }] }] } },
      },
    }),
  },
  statefulSet: {
    label: "StatefulSet 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "apps/v1",
      kind: "StatefulSet",
      metadata: { name: `${clusterName}-sts`, namespace: "default", labels: { app: `${clusterName}-sts`, managedBy: "aiops" } },
      spec: {
        serviceName: `${clusterName}-headless`,
        replicas: 1,
        selector: { matchLabels: { app: `${clusterName}-sts` } },
        template: {
          metadata: { labels: { app: `${clusterName}-sts` } },
          spec: { containers: [{ name: "app", image: "nginx:latest", ports: [{ containerPort: 80 }] }] },
        },
      },
    }),
  },
  daemonSet: {
    label: "DaemonSet 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "apps/v1",
      kind: "DaemonSet",
      metadata: { name: `${clusterName}-ds`, namespace: "default", labels: { app: `${clusterName}-ds`, managedBy: "aiops" } },
      spec: {
        selector: { matchLabels: { app: `${clusterName}-ds` } },
        template: {
          metadata: { labels: { app: `${clusterName}-ds` } },
          spec: { containers: [{ name: "agent", image: "nginx:latest", ports: [{ containerPort: 80 }] }] },
        },
      },
    }),
  },
  pod: {
    label: "Pod 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "Pod",
      metadata: { name: `${clusterName}-pod`, namespace: "default", labels: { app: `${clusterName}-pod`, managedBy: "aiops" } },
      spec: { containers: [{ name: "app", image: "nginx:latest", ports: [{ containerPort: 80 }] }] },
    }),
  },
  job: {
    label: "Job 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "batch/v1",
      kind: "Job",
      metadata: { name: `${clusterName}-job`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: {
        backoffLimit: 2,
        template: {
          metadata: { labels: { app: `${clusterName}-job` } },
          spec: {
            restartPolicy: "Never",
            containers: [{ name: "job", image: "busybox:latest", command: ["sh", "-c", "echo hello aiops"] }],
          },
        },
      },
    }),
  },
  cronJob: {
    label: "CronJob 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "batch/v1",
      kind: "CronJob",
      metadata: { name: `${clusterName}-cron`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: {
        schedule: "*/5 * * * *",
        successfulJobsHistoryLimit: 3,
        failedJobsHistoryLimit: 1,
        jobTemplate: {
          spec: {
            template: {
              metadata: { labels: { app: `${clusterName}-cron` } },
              spec: {
                restartPolicy: "OnFailure",
                containers: [{ name: "cron", image: "busybox:latest", command: ["sh", "-c", "date; echo aiops cron"] }],
              },
            },
          },
        },
      },
    }),
  },
  service: {
    label: "Service 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "Service",
      metadata: { name: `${clusterName}-web`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: { type: "ClusterIP", selector: { app: `${clusterName}-web` }, ports: [{ name: "http", port: 80, targetPort: 80 }] },
    }),
  },
  ingress: {
    label: "Ingress 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "networking.k8s.io/v1",
      kind: "Ingress",
      metadata: { name: `${clusterName}-web`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: { rules: [{ host: "web.example.local", http: { paths: [{ path: "/", pathType: "Prefix", backend: { service: { name: `${clusterName}-web`, port: { number: 80 } } } }] } }] },
    }),
  },
  configMap: {
    label: "ConfigMap 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: { name: `${clusterName}-config`, namespace: "default", labels: { managedBy: "aiops" } },
      data: { APP_ENV: "dev", LOG_LEVEL: "info" },
    }),
  },
  secret: {
    label: "Secret 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "Secret",
      metadata: { name: `${clusterName}-secret`, namespace: "default", labels: { managedBy: "aiops" } },
      type: "Opaque",
      stringData: { username: "demo", password: "change-me" },
    }),
  },
  pvc: {
    label: "PVC 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "PersistentVolumeClaim",
      metadata: { name: `${clusterName}-data`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: { accessModes: ["ReadWriteOnce"], resources: { requests: { storage: "20Gi" } } },
    }),
  },
  namespace: {
    label: "Namespace 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "v1",
      kind: "Namespace",
      metadata: { name: `${clusterName}-ns`, labels: { managedBy: "aiops" } },
    }),
  },
  storageClass: {
    label: "StorageClass 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "storage.k8s.io/v1",
      kind: "StorageClass",
      metadata: { name: `${clusterName}-standard`, labels: { managedBy: "aiops" } },
      provisioner: "kubernetes.io/no-provisioner",
      volumeBindingMode: "WaitForFirstConsumer",
    }),
  },
  hpa: {
    label: "HPA 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "autoscaling/v2",
      kind: "HorizontalPodAutoscaler",
      metadata: { name: `${clusterName}-hpa`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: {
        scaleTargetRef: { apiVersion: "apps/v1", kind: "Deployment", name: `${clusterName}-web` },
        minReplicas: 1,
        maxReplicas: 5,
        metrics: [{ type: "Resource", resource: { name: "cpu", target: { type: "Utilization", averageUtilization: 70 } } }],
      },
    }),
  },
  customResource: {
    label: "CustomResource 模板",
    namespace: "default",
    manifest: (clusterName) => ({
      apiVersion: "platform.aiops.local/v1",
      kind: "AppInstance",
      metadata: { name: `${clusterName}-app`, namespace: "default", labels: { managedBy: "aiops" } },
      spec: { replicas: 1, image: "nginx:latest" },
    }),
  },
};

function defaultResourceManifestForm(clusterName = "sample", templateKey: ResourceManifestTemplateKey = "customResource") {
  const actualKey = templateKey === "current" ? "customResource" : templateKey;
  const template = resourceManifestTemplates[actualKey];
  return {
    templateKey,
    namespace: template.namespace,
    manifestJSON: JSON.stringify(template.manifest(clusterName), null, 2),
  };
}

export function KubernetesPage() {
  const [clusters, setClusters] = useState<PageData<KubernetesClusterItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [clusterPage, setClusterPage] = useState(1);
  const [clusterPageSize, setClusterPageSize] = useState(defaultPageSize);
  const [clusterFilter, setClusterFilter] = useState(defaultClusterFilter);
  const [clusterQuery, setClusterQuery] = useState(defaultClusterFilter);
  const [clusterLoading, setClusterLoading] = useState(false);
  const [clusterSubmitting, setClusterSubmitting] = useState(false);
  const [checkingClusterId, setCheckingClusterId] = useState<number | null>(null);
  const [syncingClusterId, setSyncingClusterId] = useState<number | null>(null);
  const [selectedClusterId, setSelectedClusterId] = useState<number | null>(null);
  const [clusterEditId, setClusterEditId] = useState<number | null>(null);
  const [deleteClusterTarget, setDeleteClusterTarget] = useState<KubernetesClusterItem | null>(null);
  const [clusterForm, setClusterForm] = useState(defaultClusterForm);

  const [resources, setResources] = useState<PageData<KubernetesResourceItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [resourcePage, setResourcePage] = useState(1);
  const [resourcePageSize, setResourcePageSize] = useState(defaultPageSize);
  const [resourceFilter, setResourceFilter] = useState(defaultResourceFilter);
  const [resourceQuery, setResourceQuery] = useState(defaultResourceFilter);
  const [resourceLoading, setResourceLoading] = useState(false);
  const [resourceSubmitting, setResourceSubmitting] = useState(false);
  const [resourceEditTarget, setResourceEditTarget] = useState<KubernetesResourceItem | null>(null);
  const [resourceForm, setResourceForm] = useState<ResourceManifestForm>(() => defaultResourceManifestForm());
  const [deleteResourceTarget, setDeleteResourceTarget] = useState<KubernetesResourceItem | null>(null);
  const [namespaceOptions, setNamespaceOptions] = useState<string[]>(["default"]);
  const [namespaceLoading, setNamespaceLoading] = useState(false);
  const [namespacePanelOpen, setNamespacePanelOpen] = useState(false);

  const [nodes, setNodes] = useState<PageData<KubernetesResourceItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [nodePage, setNodePage] = useState(1);
  const [nodePageSize, setNodePageSize] = useState(defaultPageSize);
  const [nodeFilter, setNodeFilter] = useState(defaultNodeFilter);
  const [nodeQuery, setNodeQuery] = useState(defaultNodeFilter);
  const [nodeLoading, setNodeLoading] = useState(false);
  const [nodeRegisterSubmitting, setNodeRegisterSubmitting] = useState(false);
  const [nodeRegisterForm, setNodeRegisterForm] = useState(defaultNodeRegisterForm);
  const [cloudServerAssets, setCloudServerAssets] = useState<CloudAssetItem[]>([]);
  const [cloudServerLoading, setCloudServerLoading] = useState(false);

  const [operations, setOperations] = useState<PageData<KubernetesOperationItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [operationPage, setOperationPage] = useState(1);
  const [operationPageSize, setOperationPageSize] = useState(defaultPageSize);
  const [operationLoading, setOperationLoading] = useState(false);

  const [protocol, setProtocol] = useState<KubernetesAIOpsProtocol | null>(null);
  const [aiopsResult, setAIOpsResult] = useState<Record<string, unknown> | null>(null);
  const [runningActionKey, setRunningActionKey] = useState<string | null>(null);
  const [confirmActionTarget, setConfirmActionTarget] = useState<PendingKubernetesAction | null>(null);
  const [confirmManifestTarget, setConfirmManifestTarget] = useState<PendingManifestSubmit | null>(null);

  const [drawer, setDrawer] = useState<DrawerState>("closed");
  const [settingsTarget, setSettingsTarget] = useState<SettingsTarget>("closed");
  const [visibleClusterColumns, setVisibleClusterColumns] = useState(() => sanitizeVisibleColumnKeys(loadPersistedListSettings(clusterSettingsKey)?.visibleColumnKeys ?? defaultClusterColumns, clusterColumns));
  const [visibleResourceColumns, setVisibleResourceColumns] = useState(() => sanitizeVisibleColumnKeys(loadPersistedListSettings(resourceSettingsKey)?.visibleColumnKeys ?? defaultResourceColumns, resourceColumns));
  const [visibleOperationColumns, setVisibleOperationColumns] = useState(() => sanitizeVisibleColumnKeys(loadPersistedListSettings(operationSettingsKey)?.visibleColumnKeys ?? defaultOperationColumns, operationColumns));

  const selectedCluster = useMemo(() => clusters.list.find((item) => item.id === selectedClusterId) ?? null, [clusters.list, selectedClusterId]);
  const clusterTotalPages = useMemo(() => totalPages(clusters.total, clusterPageSize), [clusters.total, clusterPageSize]);
  const resourceTotalPages = useMemo(() => totalPages(resources.total, resourcePageSize), [resources.total, resourcePageSize]);
  const nodeTotalPages = useMemo(() => totalPages(nodes.total, nodePageSize), [nodes.total, nodePageSize]);
  const operationTotalPages = useMemo(() => totalPages(operations.total, operationPageSize), [operations.total, operationPageSize]);
  const visibleClusterColumnDefs = useMemo(() => columnDefs(visibleClusterColumns, clusterColumns), [visibleClusterColumns]);
  const visibleResourceColumnDefs = useMemo(() => columnDefs(visibleResourceColumns, resourceColumns), [visibleResourceColumns]);
  const visibleOperationColumnDefs = useMemo(() => columnDefs(visibleOperationColumns, operationColumns), [visibleOperationColumns]);
  const manifestJSONError = useMemo(() => validateManifestJSON(resourceForm.manifestJSON), [resourceForm.manifestJSON]);
  const filteredNamespaceOptions = useMemo(() => {
    const keyword = resourceForm.namespace.trim().toLowerCase();
    const options = Array.from(new Set(namespaceOptions.map((name) => name.trim()).filter(Boolean)));
    if (!keyword) return options;
    return options.filter((name) => name.toLowerCase().includes(keyword));
  }, [namespaceOptions, resourceForm.namespace]);

  useEffect(() => { void loadClusters(); }, [clusterPage, clusterPageSize, clusterQuery]);
  useEffect(() => { if (selectedClusterId) void loadResources(); }, [selectedClusterId, resourcePage, resourcePageSize, resourceQuery]);
  useEffect(() => { if (selectedClusterId) void loadNodes(); }, [selectedClusterId, nodePage, nodePageSize, nodeQuery]);
  useEffect(() => { void loadProtocolAndOperations(); }, [selectedClusterId, operationPage, operationPageSize]);
  useEffect(() => { if (drawer === "node-register") void loadCloudServerOptions(); }, [drawer]);
  useEffect(() => {
    if (!selectedClusterId || drawer !== "resource-create") return;
    const timer = window.setTimeout(() => {
      void loadNamespaceOptions(resourceForm.namespace.trim());
    }, 220);
    return () => window.clearTimeout(timer);
  }, [selectedClusterId, drawer, resourceForm.namespace]);
  useEffect(() => savePersistedListSettings(clusterSettingsKey, { visibleColumnKeys: visibleClusterColumns }), [visibleClusterColumns]);
  useEffect(() => savePersistedListSettings(resourceSettingsKey, { visibleColumnKeys: visibleResourceColumns }), [visibleResourceColumns]);
  useEffect(() => savePersistedListSettings(operationSettingsKey, { visibleColumnKeys: visibleOperationColumns }), [visibleOperationColumns]);

  async function loadClusters() {
    setClusterLoading(true);
    try {
      const result = await listKubernetesClusters({ page: clusterPage, pageSize: clusterPageSize, keyword: clusterQuery.keyword || undefined, env: clusterQuery.env || undefined, status: clusterQuery.status || undefined });
      setClusters(result);
      if (!selectedClusterId && result.list.length > 0) setSelectedClusterId(result.list[0].id);
    } catch {
      showToast("Kubernetes 集群加载失败");
    } finally {
      setClusterLoading(false);
    }
  }

  async function loadResources() {
    if (!selectedClusterId) return;
    setResourceLoading(true);
    try {
      const result = await listKubernetesResources({ page: resourcePage, pageSize: resourcePageSize, clusterId: selectedClusterId, kind: resourceQuery.kind || undefined, namespace: resourceQuery.namespace || undefined, status: resourceQuery.status || undefined, keyword: resourceQuery.keyword || undefined });
      setResources(result);
    } catch {
      showToast("Kubernetes 资源加载失败，请先同步集群资源");
    } finally {
      setResourceLoading(false);
    }
  }

  async function loadNodes() {
    if (!selectedClusterId) return;
    setNodeLoading(true);
    try {
      const result = await listKubernetesNodes({
        page: nodePage,
        pageSize: nodePageSize,
        clusterId: selectedClusterId,
        status: nodeQuery.status || undefined,
        keyword: nodeQuery.keyword || undefined,
      });
      setNodes(result);
    } catch {
      showToast("Kubernetes 节点加载失败");
    } finally {
      setNodeLoading(false);
    }
  }

  async function loadCloudServerOptions() {
    setCloudServerLoading(true);
    try {
      const result = await listCloudAssets({ page: 1, pageSize: 200, type: "CloudServer" });
      setCloudServerAssets(result.list);
    } catch {
      setCloudServerAssets([]);
    } finally {
      setCloudServerLoading(false);
    }
  }

  async function loadProtocolAndOperations() {
    setOperationLoading(true);
    try {
      const [protocolData, operationPageData] = await Promise.all([
        getKubernetesAIOpsProtocol(),
        listKubernetesOperations({ page: operationPage, pageSize: operationPageSize, clusterId: selectedClusterId ?? undefined }),
      ]);
      setProtocol(protocolData);
      setOperations(operationPageData);
    } catch {
      showToast("Kubernetes AIOps 协议或操作记录加载失败");
    } finally {
      setOperationLoading(false);
    }
  }

  function openClusterCreateDrawer() {
    setClusterEditId(null);
    setClusterForm(defaultClusterForm());
    setDrawer("cluster-create");
  }

  function openResourceCreateDrawer() {
    if (!selectedCluster) {
      showToast("请先选择 Kubernetes 集群");
      return;
    }
    setResourceEditTarget(null);
    setResourceForm(defaultResourceManifestForm(selectedCluster.name));
    setNamespaceOptions(["default"]);
    setNamespacePanelOpen(false);
    setDrawer("resource-create");
  }

  function openNodeRegisterDrawer() {
    if (!selectedClusterId) {
      showToast("请先选择 Kubernetes 集群");
      return;
    }
    setNodeRegisterForm(defaultNodeRegisterForm());
    setDrawer("node-register");
  }

  function applyCloudAssetToNodeForm(assetIdText: string) {
    const assetId = Number(assetIdText);
    if (!Number.isFinite(assetId) || assetId <= 0) {
      setNodeRegisterForm((prev) => ({ ...prev, cloudAssetId: assetIdText }));
      return;
    }
    const asset = cloudServerAssets.find((item) => item.id === assetId);
    if (!asset) {
      setNodeRegisterForm((prev) => ({ ...prev, cloudAssetId: assetIdText }));
      return;
    }
    const metadata = asset.metadata ?? {};
    const pickText = (...keys: string[]) => {
      for (const key of keys) {
        const value = metadata[key];
        if (value === undefined || value === null) continue;
        const text = String(value).trim();
        if (text) return text;
      }
      return "";
    };
    setNodeRegisterForm((prev) => ({
      ...prev,
      sourceType: "cloudAsset",
      cloudAssetId: assetIdText,
      hostname: prev.hostname.trim() || asset.name || asset.resourceId,
      internalIp: prev.internalIp.trim() || pickText("internalIp", "privateIp", "privateIP", "ip"),
      sshUser: prev.sshUser.trim() || pickText("sshUser", "username", "user", "loginUser") || "root",
      sshPort: prev.sshPort.trim() || pickText("sshPort", "port") || "22",
    }));
  }

  async function openResourceEditDrawer(item: KubernetesResourceItem) {
    setResourceEditTarget(item);
    setResourceForm({
      templateKey: "current",
      namespace: item.namespace ?? "",
      manifestJSON: JSON.stringify(resourceManifestTemplate(item), null, 2),
    });
    setNamespaceOptions(["default"]);
    setNamespacePanelOpen(false);
    setDrawer("resource-edit");
    try {
      const result = await getKubernetesResourceManifest(item.id);
      setResourceForm({
        templateKey: "current",
        namespace: item.namespace ?? "",
        manifestJSON: JSON.stringify(result.manifest ?? resourceManifestTemplate(item), null, 2),
      });
    } catch {
      showToast("读取资源 Manifest 失败，已使用快照模板");
    }
  }

  function openClusterEditDrawer(item: KubernetesClusterItem) {
    setClusterEditId(item.id);
    setClusterForm({
      name: item.name,
      apiServer: item.apiServer,
      credentialType: item.credentialType ?? "kubeconfig",
      kubeConfig: "",
      token: "",
      env: item.env ?? "prod",
      region: item.region ?? "",
      owner: item.owner ?? "",
      labelsJSON: JSON.stringify(item.labels ?? {}, null, 2),
      metadataJSON: JSON.stringify(item.metadata ?? {}, null, 2),
    });
    setDrawer("cluster-edit");
  }

  function closeDrawer() {
    setDrawer("closed");
    setClusterEditId(null);
    setResourceEditTarget(null);
    setClusterForm(defaultClusterForm());
    setResourceForm(defaultResourceManifestForm());
    setNodeRegisterForm(defaultNodeRegisterForm());
    setNamespaceOptions(["default"]);
    setNamespacePanelOpen(false);
  }

  async function loadNamespaceOptions(keyword: string) {
    if (!selectedClusterId) return;
    setNamespaceLoading(true);
    try {
      const pageData = await listKubernetesResources({
        page: 1,
        pageSize: 100,
        clusterId: selectedClusterId,
        kind: "Namespace",
        keyword: keyword || undefined,
      });
      const namespaceSet = new Set<string>();
      namespaceSet.add("default");
      pageData.list.forEach((item) => {
        const name = typeof item.name === "string" ? item.name.trim() : "";
        if (name) namespaceSet.add(name);
      });
      setNamespaceOptions(Array.from(namespaceSet));
    } catch {
      setNamespaceOptions(["default"]);
    } finally {
      setNamespaceLoading(false);
    }
  }

  async function submitCluster(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setClusterSubmitting(true);
    try {
      const payload = {
        name: clusterForm.name.trim(),
        apiServer: clusterForm.apiServer.trim(),
        credentialType: clusterForm.credentialType,
        kubeConfig: clusterForm.kubeConfig.trim() || undefined,
        token: clusterForm.token.trim() || undefined,
        env: clusterForm.env.trim(),
        region: clusterForm.region.trim(),
        owner: clusterForm.owner.trim(),
        labels: parseJSONObject(clusterForm.labelsJSON, "labels"),
        metadata: parseJSONObject(clusterForm.metadataJSON, "metadata"),
      };
      if (clusterEditId) await updateKubernetesCluster(clusterEditId, payload);
      else await createKubernetesCluster(payload);
      closeDrawer();
      await loadClusters();
      showToast("Kubernetes 集群保存成功");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "Kubernetes 集群保存失败");
    } finally {
      setClusterSubmitting(false);
    }
  }

  function applyResourceManifestTemplate(templateKey: ResourceManifestTemplateKey) {
    if (templateKey === "current") return;
    setResourceForm(defaultResourceManifestForm(selectedCluster?.name ?? "sample", templateKey));
    setNamespaceOptions(["default"]);
    setNamespacePanelOpen(false);
  }

  function buildResourceManifestSubmit(dryRun: boolean): PendingManifestSubmit | null {
    if (!selectedClusterId && drawer === "resource-create") {
      showToast("请先选择 Kubernetes 集群");
      return null;
    }
    let manifest: Record<string, unknown>;
    try {
      manifest = parseJSONObject(resourceForm.manifestJSON, "manifest");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "manifest 必须是 JSON 对象");
      return null;
    }
    const payload: KubernetesManifestPayload = {
      clusterId: drawer === "resource-create" ? selectedClusterId ?? undefined : undefined,
      namespace: resourceForm.namespace.trim() || undefined,
      manifest,
      dryRun,
    };
    return { mode: drawer === "resource-edit" ? "update" : "create", resourceId: resourceEditTarget?.id, payload };
  }

  async function requestResourceManifestSubmit(dryRun: boolean) {
    if (manifestJSONError) {
      showToast(manifestJSONError);
      return;
    }
    const target = buildResourceManifestSubmit(dryRun);
    if (!target) return;
    if (!dryRun) {
      setConfirmManifestTarget(target);
      return;
    }
    await executeResourceManifest(target.mode, target.resourceId, target.payload);
  }

  async function submitResourceManifest(event: FormEvent<HTMLFormElement>, dryRun: boolean) {
    event.preventDefault();
    await requestResourceManifestSubmit(dryRun);
  }

  async function executeResourceManifest(mode: "create" | "update", resourceId: number | undefined, payload: KubernetesManifestPayload) {
    setResourceSubmitting(true);
    try {
      const result = mode === "update" && resourceId ? await updateKubernetesResource(resourceId, payload) : await createKubernetesResource(payload);
      setAIOpsResult((result.dryRun ?? result.operation.result ?? {}) as Record<string, unknown>);
      if (payload.dryRun === false) {
        closeDrawer();
        await Promise.all([loadResources(), loadNodes(), loadProtocolAndOperations()]);
      }
      showToast(payload.dryRun === false ? "Kubernetes 资源提交成功" : "资源 dry-run 已生成");
    } catch {
      showToast("Kubernetes 资源提交失败");
    } finally {
      setResourceSubmitting(false);
    }
  }

  async function confirmResourceManifestSubmit() {
    if (!confirmManifestTarget) return;
    const target = confirmManifestTarget;
    setConfirmManifestTarget(null);
    await executeResourceManifest(target.mode, target.resourceId, { ...target.payload, dryRun: false, confirmationText: kubernetesConfirmSubmitText });
  }

  async function handleCheckCluster(clusterId: number) {
    setCheckingClusterId(clusterId);
    try {
      await checkKubernetesCluster(clusterId);
      await loadClusters();
      showToast("Kubernetes 集群校验成功");
    } catch {
      showToast("Kubernetes 集群校验失败");
    } finally {
      setCheckingClusterId(null);
    }
  }

  async function handleSyncCluster(clusterId: number) {
    setSyncingClusterId(clusterId);
    try {
      const result = await syncKubernetesCluster(clusterId);
      await Promise.all([loadClusters(), loadResources(), loadNodes(), loadProtocolAndOperations()]);
      const warningText = result.warnings?.length ? `，部分资源同步失败：${result.warnings.length} 项` : "";
      showToast(`Kubernetes 资源同步完成：${result.count} 个${warningText}`);
    } catch {
      showToast("Kubernetes 资源同步失败");
    } finally {
      setSyncingClusterId(null);
    }
  }

  async function handleDeleteCluster() {
    if (!deleteClusterTarget) return;
    try {
      await deleteKubernetesCluster(deleteClusterTarget.id);
      if (selectedClusterId === deleteClusterTarget.id) setSelectedClusterId(null);
      setDeleteClusterTarget(null);
      await loadClusters();
      showToast("Kubernetes 集群删除成功");
    } catch {
      showToast("Kubernetes 集群删除失败");
    }
  }

  async function handleDeleteResource() {
    if (!deleteResourceTarget) return;
    try {
      const result = await deleteKubernetesResource(deleteResourceTarget.id);
      setAIOpsResult((result.operation.result ?? {}) as Record<string, unknown>);
      setDeleteResourceTarget(null);
      await Promise.all([loadResources(), loadNodes(), loadProtocolAndOperations()]);
      showToast("Kubernetes 资源删除成功");
    } catch {
      showToast("Kubernetes 资源删除失败");
    }
  }

  async function submitNodeRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedClusterId) {
      showToast("请先选择 Kubernetes 集群");
      return;
    }
    const isMockCluster = (selectedCluster?.apiServer ?? "").toLowerCase().startsWith("mock://");
    const hostname = nodeRegisterForm.hostname.trim();
    const internalIp = nodeRegisterForm.internalIp.trim();
    if (!hostname && !internalIp) {
      showToast("请至少填写主机名或内网 IP");
      return;
    }
    if (nodeRegisterForm.sourceType === "cloudAsset" && !nodeRegisterForm.cloudAssetId.trim()) {
      showToast("请选择 CloudServer 资产");
      return;
    }
    setNodeRegisterSubmitting(true);
    try {
      const basePayload: KubernetesNodeRegisterPayload = {
        clusterId: selectedClusterId,
        hostname,
        internalIp: internalIp || undefined,
        roles: nodeRegisterForm.rolesCSV.split(",").map((item) => item.trim()).filter(Boolean),
        cpu: nodeRegisterForm.cpu.trim() || undefined,
        memory: nodeRegisterForm.memory.trim() || undefined,
        pods: nodeRegisterForm.pods.trim() || undefined,
        kubeletVersion: nodeRegisterForm.kubeletVersion.trim() || undefined,
        labels: parseJSONObject(nodeRegisterForm.labelsJSON, "labels"),
        metadata: parseJSONObject(nodeRegisterForm.metadataJSON, "metadata"),
      };
      if (nodeRegisterForm.executionMode === "task") {
        const cloudAssetId = Number(nodeRegisterForm.cloudAssetId);
        const joinCommand = nodeRegisterForm.joinCommand.trim();
        if (nodeRegisterForm.executeNow && !joinCommand && !isMockCluster) {
          throw new Error("非 mock 集群必须填写 kubeadm join 命令");
        }
        const sshPort = (() => {
          const port = Number(nodeRegisterForm.sshPort.trim());
          if (!nodeRegisterForm.sshPort.trim()) return undefined;
          if (!Number.isInteger(port) || port <= 0 || port > 65535) {
            throw new Error("SSH 端口必须在 1-65535 之间");
          }
          return port;
        })();
        const payload: KubernetesNodeRegisterTaskPayload = {
          ...basePayload,
          cloudAssetId: Number.isFinite(cloudAssetId) && cloudAssetId > 0 ? cloudAssetId : undefined,
          joinCommand: joinCommand || undefined,
          sshUser: nodeRegisterForm.sshUser.trim() || undefined,
          sshPassword: nodeRegisterForm.sshPassword.trim() || undefined,
          sshPort,
          dryRun: nodeRegisterForm.dryRunTask,
          executeNow: nodeRegisterForm.executeNow,
        };
        await registerKubernetesNodeTask(payload);
      } else {
        await registerKubernetesNode(basePayload);
      }
      closeDrawer();
      await Promise.all([loadNodes(), loadResources(), loadProtocolAndOperations()]);
      if (nodeRegisterForm.executionMode === "task") {
        showToast(nodeRegisterForm.executeNow ? "Kubernetes 节点注册任务提交成功" : "Kubernetes 节点注册任务已创建，可在任务中心执行");
      } else {
        showToast("Kubernetes 节点注册成功");
      }
    } catch (error) {
      showToast(error instanceof Error ? error.message : "Kubernetes 节点注册失败");
    } finally {
      setNodeRegisterSubmitting(false);
    }
  }

  async function executeKubernetesAction(payload: KubernetesActionPayload, actionKey: string) {
    setRunningActionKey(actionKey);
    try {
      const result = await runKubernetesAction(payload);
      setAIOpsResult((result.dryRun ?? result.operation.result ?? {}) as Record<string, unknown>);
      await loadProtocolAndOperations();
      if (!payload.dryRun) await Promise.all([loadResources(), loadNodes()]);
      showToast(payload.dryRun === false ? "Kubernetes 动作执行成功" : "dry-run 已生成");
    } catch {
      showToast("Kubernetes 动作执行失败");
    } finally {
      setRunningActionKey(null);
    }
  }

  function requestKubernetesAction(resource: KubernetesResourceItem, action: string, dryRun: boolean) {
    const actionKey = `${resource.id}-${action}-${dryRun ? "dry" : "run"}`;
    const params: Record<string, unknown> = { source: "kubernetes-page" };
    if (action === "scale") {
      const current = typeof resource.metadata?.replicas === "number" ? String(resource.metadata.replicas) : "";
      const replicasText = window.prompt("请输入目标副本数（非负整数）", current);
      if (replicasText === null) return;
      const replicas = Number(replicasText.trim());
      if (!Number.isInteger(replicas) || replicas < 0) {
        showToast("副本数必须是非负整数");
        return;
      }
      params.replicas = replicas;
    }
    const payload: KubernetesActionPayload = { clusterId: resource.clusterId, namespace: resource.namespace, kind: resource.kind, name: resource.name, action, dryRun, params };
    if (dryRun || !kubernetesActionNeedsConfirm(payload)) {
      void executeKubernetesAction(payload, actionKey);
      return;
    }
    setConfirmActionTarget({
      title: action === "delete" ? "删除 Kubernetes 资源" : "执行 Kubernetes 高危操作",
      description: `确认对 ${resource.namespace || "cluster"}/${resource.kind}/${resource.name} 执行 ${action}？请先确认影响面和回滚方式。`,
      payload,
      actionKey,
    });
  }

  async function confirmKubernetesAction() {
    if (!confirmActionTarget) return;
    const target = confirmActionTarget;
    setConfirmActionTarget(null);
    await executeKubernetesAction({ ...target.payload, confirmationText: kubernetesConfirmDeleteText }, target.actionKey);
  }

  function handleClusterFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setClusterPage(1);
    setClusterQuery({ ...clusterFilter });
  }

  function handleResourceFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setResourcePage(1);
    setResourceQuery({ ...resourceFilter });
  }

  function handleNodeFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNodePage(1);
    setNodeQuery({ ...nodeFilter });
  }

  function renderActionsHeader(label: string, onClick: () => void) {
    return <div className="table-actions-header"><span>{label}</span><button className="table-settings-trigger cursor-pointer" type="button" onClick={onClick} aria-label={`${label}列表字段设置`}>⚙️</button></div>;
  }

  function renderTableHeader(column: TableSettingsColumn, target: Exclude<SettingsTarget, "closed">) {
    if (column.key !== "actions") return column.label;
    return renderActionsHeader(column.label, () => setSettingsTarget(target));
  }

  function renderClusterCell(item: KubernetesClusterItem, key: string) {
    switch (key) {
      case "id": return item.id;
      case "name": return item.name;
      case "apiServer": return <code>{item.apiServer}</code>;
      case "env": return item.env || "-";
      case "region": return item.region || "-";
      case "owner": return item.owner || "-";
      case "status": return <span className={`docker-status docker-status-${item.status ?? "unknown"}`}>{item.status || "unknown"}</span>;
      case "version": return item.version || "-";
      case "lastCheckedAt": return formatDateTime(item.lastCheckedAt);
      case "lastSyncedAt": return formatDateTime(item.lastSyncedAt);
      case "actions": return (
        <div className="rbac-row-actions">
          <ListRowActions
            title="Kubernetes 集群更多操作"
            actions={[
              { key: "select", label: "查看资源", permissionKey: "button.kubernetes.resource.view", onClick: () => { setSelectedClusterId(item.id); setResourcePage(1); } },
              { key: "check", label: checkingClusterId === item.id ? "校验中..." : "校验", permissionKey: "button.kubernetes.cluster.check", disabled: checkingClusterId === item.id, onClick: () => void handleCheckCluster(item.id) },
              { key: "sync", label: syncingClusterId === item.id ? "同步中..." : "同步", permissionKey: "button.kubernetes.cluster.sync", disabled: syncingClusterId === item.id, onClick: () => { setSelectedClusterId(item.id); void handleSyncCluster(item.id); } },
              { key: "edit", label: "编辑", permissionKey: "button.kubernetes.cluster.update", onClick: () => openClusterEditDrawer(item) },
              { key: "delete", label: "删除", permissionKey: "button.kubernetes.cluster.delete", onClick: () => setDeleteClusterTarget(item) },
            ]}
          />
        </div>
      );
      default: return "-";
    }
  }

  function renderResourceCell(item: KubernetesResourceItem, key: string) {
    switch (key) {
      case "id": return item.id;
      case "clusterId": return item.clusterId;
      case "namespace": return item.namespace || "-";
      case "kind": return item.kind;
      case "name": return item.name;
      case "status": return item.status || "-";
      case "specSummary": return item.specSummary || "-";
      case "lastSyncedAt": return formatDateTime(item.lastSyncedAt);
      case "actions": return (
        <div className="rbac-row-actions">
          <ListRowActions
            title="Kubernetes 资源更多操作"
            actions={[
              { key: "edit-manifest", label: "编辑", permissionKey: "button.kubernetes.resource.update", onClick: () => void openResourceEditDrawer(item) },
              { key: "delete-resource", label: "删除", permissionKey: "button.kubernetes.resource.delete", onClick: () => setDeleteResourceTarget(item) },
              ...resourceActions(item).filter((action) => action !== "delete").flatMap((action) => [
                { key: `${action}-dry`, label: `${action} dry-run`, permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-${action}-dry`, onClick: () => requestKubernetesAction(item, action, true) },
                { key: `${action}-run`, label: action, permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-${action}-run`, onClick: () => requestKubernetesAction(item, action, false) },
              ]),
            ]}
          />
        </div>
      );
      default: return "-";
    }
  }

  function renderOperationCell(item: KubernetesOperationItem, key: string) {
    switch (key) {
      case "traceId": return <code>{shorten(item.traceId)}</code>;
      case "clusterId": return item.clusterId;
      case "namespace": return item.namespace || "-";
      case "kind": return item.kind;
      case "name": return item.name;
      case "action": return item.action;
      case "status": return item.status;
      case "dryRun": return item.dryRun ? "是" : "否";
      case "riskLevel": return item.riskLevel || "-";
      case "createdAt": return formatDateTime(item.createdAt);
      case "actions": return "-";
      default: return "-";
    }
  }

  function renderNodeCell(item: KubernetesResourceItem, key: (typeof nodeTableColumns)[number]) {
    switch (key) {
      case "name": return item.name;
      case "status": return item.status || "-";
      case "specSummary": return item.specSummary || "-";
      case "lastSyncedAt": return formatDateTime(item.lastSyncedAt);
      case "actions":
        return (
          <div className="rbac-row-actions">
            <ListRowActions
              title="节点更多操作"
              actions={[
                { key: "node-cordon-dry", label: "cordon dry-run", permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-cordon-dry`, onClick: () => requestKubernetesAction(item, "cordon", true) },
                { key: "node-cordon", label: "cordon", permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-cordon-run`, onClick: () => requestKubernetesAction(item, "cordon", false) },
                { key: "node-uncordon-dry", label: "uncordon dry-run", permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-uncordon-dry`, onClick: () => requestKubernetesAction(item, "uncordon", true) },
                { key: "node-uncordon", label: "uncordon", permissionKey: "button.kubernetes.resource.action", disabled: runningActionKey === `${item.id}-uncordon-run`, onClick: () => requestKubernetesAction(item, "uncordon", false) },
              ]}
            />
          </div>
        );
      default:
        return "-";
    }
  }

  const drawerVisible = drawer !== "closed";
  const isResourceDrawer = drawer.startsWith("resource");
  const isNodeRegisterDrawer = drawer === "node-register";
  const resourceDrawerMode = drawer === "resource-edit" ? "edit" : "create";
  const resourceDrawerTitle = resourceDrawerMode === "edit" ? "编辑 Kubernetes 资源" : "创建 Kubernetes 资源";
  const resourceDrawerHint = resourceDrawerMode === "edit"
    ? `正在编辑 ${resourceEditTarget?.namespace || "cluster"}/${resourceEditTarget?.kind ?? "-"}/${resourceEditTarget?.name ?? "-"}，真实提交会更新集群中的现有对象。`
    : `将在 ${selectedCluster?.name ?? "当前集群"} 中创建新资源，建议先执行 dry-run 验证影响范围。`;

  return (
    <section className="page">
      <h2>Kubernetes 管理</h2>
      <div className="rbac-module-grid docker-module-grid">
        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>K8s 集群</h3>
              <p className="muted">纳管多 Kubernetes 集群，统一校验、同步资源快照和执行安全动作。</p>
            </div>
            <PermissionButton permissionKey="button.kubernetes.cluster.create" className="btn primary cursor-pointer" type="button" onClick={openClusterCreateDrawer}>创建集群</PermissionButton>
          </header>
          <form className="cloud-filter-bar" onSubmit={handleClusterFilterSubmit}>
            <input className="cloud-filter-control cloud-filter-keyword" value={clusterFilter.keyword} onChange={(event) => setClusterFilter((prev) => ({ ...prev, keyword: event.target.value }))} placeholder="关键词：名称/API Server/负责人" />
            <input className="cloud-filter-control" value={clusterFilter.env} onChange={(event) => setClusterFilter((prev) => ({ ...prev, env: event.target.value }))} placeholder="环境：prod" />
            <select className="cloud-filter-control" value={clusterFilter.status} onChange={(event) => setClusterFilter((prev) => ({ ...prev, status: event.target.value }))}>
              <option value="">状态：全部</option><option value="connected">connected</option><option value="error">error</option><option value="unknown">unknown</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={clusterLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={() => { const next = defaultClusterFilter(); setClusterFilter(next); setClusterQuery(next); }}>重置</button>
            </div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table"><thead><tr>{visibleClusterColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "clusters")}</th>)}</tr></thead><tbody>
              {clusterLoading ? <tr><td colSpan={visibleClusterColumns.length}>加载中...</td></tr> : clusters.list.length === 0 ? <tr><td colSpan={visibleClusterColumns.length}>暂无数据</td></tr> : clusters.list.map((item) => <tr key={item.id} className={selectedClusterId === item.id ? "docker-selected-row" : ""}>{visibleClusterColumnDefs.map((column) => <td key={column.key}>{renderClusterCell(item, column.key)}</td>)}</tr>)}
            </tbody></table>
          </div>
          <Pagination total={clusters.total} page={clusterPage} totalPages={clusterTotalPages} pageSize={clusterPageSize} pageSizeOptions={pageSizeOptions} onPageChange={setClusterPage} onPageSizeChange={(next) => { setClusterPage(1); setClusterPageSize(next); }} />
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header"><div><h3>资源快照</h3><p className="muted">{selectedCluster ? `当前集群：${selectedCluster.name}` : "请选择集群后同步并查看资源。"}</p></div><PermissionButton permissionKey="button.kubernetes.resource.create" className="btn primary cursor-pointer" type="button" disabled={!selectedClusterId} onClick={openResourceCreateDrawer}>创建资源</PermissionButton></header>
          <form className="cloud-filter-bar" onSubmit={handleResourceFilterSubmit}>
            <select className="cloud-filter-control" value={resourceFilter.kind} onChange={(event) => setResourceFilter((prev) => ({ ...prev, kind: event.target.value }))}>{resourceKindFilterOptions.map((kind) => <option key={kind || "all"} value={kind}>{kind || "类型：全部"}</option>)}</select>
            <input className="cloud-filter-control" value={resourceFilter.namespace} onChange={(event) => setResourceFilter((prev) => ({ ...prev, namespace: event.target.value }))} placeholder="命名空间" />
            <input className="cloud-filter-control" value={resourceFilter.status} onChange={(event) => setResourceFilter((prev) => ({ ...prev, status: event.target.value }))} placeholder="状态" />
            <input className="cloud-filter-control cloud-filter-keyword" value={resourceFilter.keyword} onChange={(event) => setResourceFilter((prev) => ({ ...prev, keyword: event.target.value }))} placeholder="关键词：名称/UID/配置摘要" />
            <div className="cloud-filter-actions"><button className="btn cursor-pointer" type="submit" disabled={!selectedClusterId || resourceLoading}>查询</button><button className="btn cursor-pointer" type="button" onClick={() => { const next = defaultResourceFilter(); setResourceFilter(next); setResourceQuery(next); }}>重置</button></div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll"><table className="rbac-table"><thead><tr>{visibleResourceColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "resources")}</th>)}</tr></thead><tbody>
            {resourceLoading ? <tr><td colSpan={visibleResourceColumns.length}>加载中...</td></tr> : resources.list.length === 0 ? <tr><td colSpan={visibleResourceColumns.length}>暂无资源</td></tr> : resources.list.map((item) => <tr key={item.id}>{visibleResourceColumnDefs.map((column) => <td key={column.key}>{renderResourceCell(item, column.key)}</td>)}</tr>)}
          </tbody></table></div>
          <Pagination total={resources.total} page={resourcePage} totalPages={resourceTotalPages} pageSize={resourcePageSize} pageSizeOptions={pageSizeOptions} onPageChange={setResourcePage} onPageSizeChange={(next) => { setResourcePage(1); setResourcePageSize(next); }} />
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header"><div><h3>节点管理</h3><p className="muted">{selectedCluster ? `当前集群：${selectedCluster.name}，可执行 cordon/uncordon 并注册主机。` : "请选择集群后查看节点。"}</p></div><PermissionButton permissionKey="button.kubernetes.node.register" className="btn primary cursor-pointer" type="button" disabled={!selectedClusterId} onClick={openNodeRegisterDrawer}>注册主机</PermissionButton></header>
          <form className="cloud-filter-bar" onSubmit={handleNodeFilterSubmit}>
            <input className="cloud-filter-control cloud-filter-keyword" value={nodeFilter.keyword} onChange={(event) => setNodeFilter((prev) => ({ ...prev, keyword: event.target.value }))} placeholder="关键词：节点名称/UID/配置摘要" />
            <input className="cloud-filter-control" value={nodeFilter.status} onChange={(event) => setNodeFilter((prev) => ({ ...prev, status: event.target.value }))} placeholder="状态：Ready/NotReady" />
            <div className="cloud-filter-actions"><button className="btn cursor-pointer" type="submit" disabled={!selectedClusterId || nodeLoading}>查询</button><button className="btn cursor-pointer" type="button" onClick={() => { const next = defaultNodeFilter(); setNodeFilter(next); setNodeQuery(next); }}>重置</button></div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll"><table className="rbac-table"><thead><tr>{nodeTableColumns.map((column) => <th key={column}>{column === "name" ? "节点" : column === "status" ? "状态" : column === "specSummary" ? "配置摘要" : column === "lastSyncedAt" ? "同步时间" : "操作"}</th>)}</tr></thead><tbody>
            {nodeLoading ? <tr><td colSpan={nodeTableColumns.length}>加载中...</td></tr> : nodes.list.length === 0 ? <tr><td colSpan={nodeTableColumns.length}>暂无节点</td></tr> : nodes.list.map((item) => <tr key={item.id}>{nodeTableColumns.map((column) => <td key={column}>{renderNodeCell(item, column)}</td>)}</tr>)}
          </tbody></table></div>
          <Pagination total={nodes.total} page={nodePage} totalPages={nodeTotalPages} pageSize={nodePageSize} pageSizeOptions={nodePageSizeOptions} onPageChange={setNodePage} onPageSizeChange={(next) => { setNodePage(1); setNodePageSize(next); }} />
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header"><div><h3>操作记录</h3><p className="muted">所有 dry-run 与真实动作写入审计记录，供 AIOps 读取分析。</p></div></header>
          <div className="rbac-table-wrapper rbac-module-scroll"><table className="rbac-table"><thead><tr>{visibleOperationColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "operations")}</th>)}</tr></thead><tbody>
            {operationLoading ? <tr><td colSpan={visibleOperationColumns.length}>加载中...</td></tr> : operations.list.length === 0 ? <tr><td colSpan={visibleOperationColumns.length}>暂无操作记录</td></tr> : operations.list.map((item) => <tr key={item.id}>{visibleOperationColumnDefs.map((column) => <td key={column.key}>{renderOperationCell(item, column.key)}</td>)}</tr>)}
          </tbody></table></div>
          <Pagination total={operations.total} page={operationPage} totalPages={operationTotalPages} pageSize={operationPageSize} pageSizeOptions={pageSizeOptions} onPageChange={setOperationPage} onPageSizeChange={(next) => { setOperationPage(1); setOperationPageSize(next); }} />
        </article>

        <article className="card rbac-module-card cloud-module-card docker-aiops-card">
          <header className="rbac-module-header"><div><h3>AIOps 操作协议</h3><p className="muted">自然语言后续可复用统一 <code>{"{clusterId, namespace, kind, name, action, dryRun, params}"}</code> 协议。</p></div></header>
          <div className="docker-aiops-protocol"><div><strong>协议版本</strong><code>{protocol?.protocolVersion ?? "-"}</code></div><div><strong>Action Endpoint</strong><code>{protocol?.actionEndpoint ?? "-"}</code></div><div><strong>支持资源</strong><span>{protocol?.resources?.map((item) => `${item.kind}:${item.actions.join("/")}`).join("，") ?? "-"}</span></div></div>
          {aiopsResult ? <pre className="docker-aiops-result">{JSON.stringify(aiopsResult, null, 2)}</pre> : <p className="muted">点击资源行中的 dry-run 可查看影响范围、风险等级、审批要求和回滚提示。</p>}
        </article>
      </div>

      {drawerVisible && <div className="rbac-drawer-mask"><aside className={`rbac-drawer docker-drawer-wide ${isResourceDrawer ? `k8s-resource-drawer k8s-resource-drawer-${resourceDrawerMode}` : ""}`}><header className="rbac-drawer-header"><h3>{isResourceDrawer ? resourceDrawerTitle : isNodeRegisterDrawer ? "注册 Kubernetes 节点" : clusterEditId ? "编辑 Kubernetes 集群" : "创建 Kubernetes 集群"}</h3><button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>关闭</button></header>
        {drawer.startsWith("cluster") ? <form className="rbac-drawer-body middleware-form" onSubmit={submitCluster}>
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>集群基础信息</h4><p className="muted">生产环境 API Server 必须使用 HTTPS；开发环境可用 mock://dev 做功能联调。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field middleware-form-field-wide"><span>集群名称</span><input required value={clusterForm.name} placeholder="prod-k8s-guangzhou" onChange={(event) => setClusterForm((prev) => ({ ...prev, name: event.target.value }))} /></label>
            <label className="middleware-form-field middleware-form-field-wide"><span>API Server</span><input required value={clusterForm.apiServer} placeholder="https://10.0.0.1:6443 或 mock://dev" onChange={(event) => setClusterForm((prev) => ({ ...prev, apiServer: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>环境</span><input value={clusterForm.env} placeholder="prod / staging / dev" onChange={(event) => setClusterForm((prev) => ({ ...prev, env: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>区域</span><input value={clusterForm.region} placeholder="ap-guangzhou" onChange={(event) => setClusterForm((prev) => ({ ...prev, region: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>负责人</span><input value={clusterForm.owner} placeholder="SRE / 平台组" onChange={(event) => setClusterForm((prev) => ({ ...prev, owner: event.target.value }))} /></label>
          </div></section>
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>连接凭据</h4><p className="muted">凭据只提交到后端加密保存，列表不会回显明文。编辑时留空表示不更新凭据。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field"><span>凭据类型</span><select value={clusterForm.credentialType} onChange={(event) => setClusterForm((prev) => ({ ...prev, credentialType: event.target.value }))}><option value="kubeconfig">kubeconfig</option><option value="token">token</option></select></label>
            {clusterForm.credentialType === "kubeconfig" ? <label className="middleware-form-field middleware-form-field-wide"><span>KubeConfig</span><textarea className="docker-compose-editor docker-compose-editor-large" required={!clusterEditId} value={clusterForm.kubeConfig} onChange={(event) => setClusterForm((prev) => ({ ...prev, kubeConfig: event.target.value }))} /></label> : <label className="middleware-form-field middleware-form-field-wide"><span>ServiceAccount Token</span><textarea className="middleware-json-editor" required={!clusterEditId} value={clusterForm.token} onChange={(event) => setClusterForm((prev) => ({ ...prev, token: event.target.value }))} /></label>}
          </div></section>
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>标签与扩展元数据</h4><p className="muted">用于 AIOps 上下文、容量分析和集群策略编排。</p></div><div className="middleware-form-grid"><label className="middleware-form-field middleware-form-field-wide"><span>Labels JSON</span><textarea className="middleware-json-editor" value={clusterForm.labelsJSON} onChange={(event) => setClusterForm((prev) => ({ ...prev, labelsJSON: event.target.value }))} /></label><label className="middleware-form-field middleware-form-field-wide"><span>Metadata JSON</span><textarea className="middleware-json-editor" value={clusterForm.metadataJSON} onChange={(event) => setClusterForm((prev) => ({ ...prev, metadataJSON: event.target.value }))} /></label></div></section>
          <div className="middleware-form-actions"><button className="btn primary cursor-pointer" type="submit" disabled={clusterSubmitting}>{clusterSubmitting ? "保存中..." : "保存"}</button><button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>取消</button></div>
        </form> : drawer === "node-register" ? <form className="rbac-drawer-body middleware-form" onSubmit={(event) => void submitNodeRegister(event)}>
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>注册模式</h4><p className="muted">支持手动录入主机信息，或从多云 CloudServer 资产选择主机；支持直接注册与任务编排注册。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field"><span>主机来源</span><select value={nodeRegisterForm.sourceType} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, sourceType: event.target.value as "manual" | "cloudAsset" }))}><option value="manual">手动录入</option><option value="cloudAsset">多云主机</option></select></label>
            <label className="middleware-form-field"><span>执行方式</span><select value={nodeRegisterForm.executionMode} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, executionMode: event.target.value as "direct" | "task" }))}><option value="task">任务编排（推荐）</option><option value="direct">直接注册</option></select></label>
            {nodeRegisterForm.executionMode === "task" ? <label className="middleware-form-field"><span>任务执行策略</span><select value={nodeRegisterForm.executeNow ? "now" : "later"} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, executeNow: event.target.value === "now" }))}><option value="now">立即执行</option><option value="later">仅创建任务</option></select></label> : null}
            {nodeRegisterForm.sourceType === "cloudAsset" ? <label className="middleware-form-field middleware-form-field-wide"><span>云主机资产</span><select value={nodeRegisterForm.cloudAssetId} onChange={(event) => applyCloudAssetToNodeForm(event.target.value)}><option value="">{cloudServerLoading ? "云主机加载中..." : "请选择 CloudServer 资产"}</option>{cloudServerAssets.map((asset) => <option key={asset.id} value={asset.id}>{`${asset.id} · ${asset.provider}/${asset.region} · ${asset.name}`}</option>)}</select></label> : null}
          </div></section>
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>主机基础信息</h4><p className="muted">用于将目标主机注册为 Kubernetes Node（开发环境 mock 集群将生成节点快照）。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field"><span>主机名</span><input required value={nodeRegisterForm.hostname} placeholder="worker-01" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, hostname: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>内网 IP</span><input value={nodeRegisterForm.internalIp} placeholder="10.0.0.21" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, internalIp: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>角色</span><input value={nodeRegisterForm.rolesCSV} placeholder="worker,ingress" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, rolesCSV: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>Kubelet 版本</span><input value={nodeRegisterForm.kubeletVersion} placeholder="v1.29.3" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, kubeletVersion: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>CPU</span><input value={nodeRegisterForm.cpu} placeholder="4" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, cpu: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>内存</span><input value={nodeRegisterForm.memory} placeholder="16Gi" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, memory: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>Pod 上限</span><input value={nodeRegisterForm.pods} placeholder="110" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, pods: event.target.value }))} /></label>
          </div></section>
          {nodeRegisterForm.executionMode === "task" ? <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>任务编排参数</h4><p className="muted">填写 SSH 与 kubeadm join 命令，通过任务中心执行并记录日志。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field middleware-form-field-wide"><span>kubeadm join 命令</span><textarea className="middleware-json-editor" value={nodeRegisterForm.joinCommand} placeholder="kubeadm join <control-plane>:6443 --token ... --discovery-token-ca-cert-hash sha256:..." onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, joinCommand: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>SSH 用户</span><input value={nodeRegisterForm.sshUser} placeholder="root" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, sshUser: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>SSH 端口</span><input value={nodeRegisterForm.sshPort} placeholder="22" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, sshPort: event.target.value }))} /></label>
            <label className="middleware-form-field middleware-form-field-wide"><span>SSH 密码（可选）</span><input type="password" value={nodeRegisterForm.sshPassword} placeholder="如使用密钥免密可留空" onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, sshPassword: event.target.value }))} /></label>
            <label className="middleware-form-field"><span>先 Dry-run</span><select value={nodeRegisterForm.dryRunTask ? "yes" : "no"} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, dryRunTask: event.target.value === "yes" }))}><option value="no">否</option><option value="yes">是</option></select></label>
          </div></section> : null}
          <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>标签与扩展元数据</h4><p className="muted">标签会写入 Node metadata.labels，元数据用于审计和 AIOps 上下文。</p></div><div className="middleware-form-grid">
            <label className="middleware-form-field middleware-form-field-wide"><span>Labels JSON</span><textarea className="middleware-json-editor" value={nodeRegisterForm.labelsJSON} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, labelsJSON: event.target.value }))} /></label>
            <label className="middleware-form-field middleware-form-field-wide"><span>Metadata JSON</span><textarea className="middleware-json-editor" value={nodeRegisterForm.metadataJSON} onChange={(event) => setNodeRegisterForm((prev) => ({ ...prev, metadataJSON: event.target.value }))} /></label>
          </div></section>
          <div className="middleware-form-actions"><button className="btn primary cursor-pointer" type="submit" disabled={nodeRegisterSubmitting}>{nodeRegisterSubmitting ? "提交中..." : nodeRegisterForm.executionMode === "task" ? "提交注册任务" : "注册节点"}</button><button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>取消</button></div>
        </form> : <form className="rbac-drawer-body middleware-form k8s-resource-form" onSubmit={(event) => void submitResourceManifest(event, true)}>
          <div className={`k8s-resource-mode-banner k8s-resource-mode-${resourceDrawerMode}`}><span className="k8s-resource-mode-label">{resourceDrawerMode === "edit" ? "编辑模式" : "创建模式"}</span><strong>{resourceDrawerTitle}</strong><p>{resourceDrawerHint}</p></div>
          <section className="middleware-form-section k8s-resource-section k8s-resource-config-section"><div className="middleware-form-section-title k8s-resource-section-title"><h4>资源配置</h4></div><div className={`middleware-form-grid k8s-resource-grid ${resourceDrawerMode === "edit" ? "k8s-resource-grid-edit" : ""}`}>
            {drawer === "resource-create" ? <label className="middleware-form-field k8s-template-field"><span>模板示例</span><small className="muted k8s-field-hint">选择一个基础模板后再调整 Manifest。</small><select value={resourceForm.templateKey} onChange={(event) => applyResourceManifestTemplate(event.target.value as ResourceManifestTemplateKey)}>{resourceManifestTemplateKeys.map((key) => <option key={key} value={key}>{resourceManifestTemplates[key].label}</option>)}</select></label> : null}
            <label className="middleware-form-field k8s-namespace-field"><span>命名空间</span><small className="muted k8s-field-hint">{resourceDrawerMode === "edit" ? "编辑模式下命名空间不可修改。若需变更命名空间，请新建资源后迁移。" : "支持输入关键字搜索命名空间，默认值为 default。"}</small>{resourceDrawerMode === "edit" ? <input value={resourceForm.namespace || "default"} readOnly aria-readonly="true" /> : <div className="k8s-namespace-combobox"><input value={resourceForm.namespace} placeholder="default" onFocus={() => setNamespacePanelOpen(true)} onBlur={() => window.setTimeout(() => setNamespacePanelOpen(false), 120)} onChange={(event) => { setResourceForm((prev) => ({ ...prev, namespace: event.target.value })); setNamespacePanelOpen(true); }} /><button className="k8s-namespace-toggle cursor-pointer" type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => setNamespacePanelOpen((open) => !open)} aria-label="切换命名空间候选">▾</button>{namespacePanelOpen ? <div className="k8s-namespace-dropdown">{namespaceLoading ? <div className="k8s-namespace-empty">命名空间加载中...</div> : filteredNamespaceOptions.length > 0 ? filteredNamespaceOptions.map((name) => <button key={name} className="k8s-namespace-option cursor-pointer" type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => { setResourceForm((prev) => ({ ...prev, namespace: name })); setNamespacePanelOpen(false); }}>{name}</button>) : <div className="k8s-namespace-empty">无匹配命名空间，可直接输入新值</div>}</div> : null}</div>}</label>
          </div></section>
          <section className="middleware-form-section k8s-resource-section k8s-resource-manifest-section"><div className="middleware-form-section-title"><h4>Manifest JSON</h4><p className="muted">{resourceDrawerMode === "edit" ? "编辑前建议先 dry-run，避免直接覆盖集群对象。" : "模板只提供基础结构，提交前请检查 apiVersion、kind、metadata 和 spec。"}</p></div><div className="middleware-form-grid k8s-resource-grid">
            <label className="middleware-form-field middleware-form-field-wide"><span>Manifest JSON</span><textarea className={`docker-compose-editor docker-compose-editor-large k8s-manifest-editor ${manifestJSONError ? "k8s-json-editor-invalid" : ""}`} required value={resourceForm.manifestJSON} onChange={(event) => setResourceForm((prev) => ({ ...prev, manifestJSON: event.target.value }))} />{manifestJSONError ? <small className="k8s-json-error">{manifestJSONError}</small> : <small className="muted k8s-json-ok">JSON 语法校验通过</small>}</label>
          </div></section>
          <div className="middleware-form-actions k8s-manifest-actions"><button className="btn cursor-pointer" type="submit" disabled={resourceSubmitting || Boolean(manifestJSONError)}>{resourceSubmitting ? "提交中..." : "Dry-run"}</button><button className="btn primary cursor-pointer" type="button" disabled={resourceSubmitting || Boolean(manifestJSONError)} onClick={() => void requestResourceManifestSubmit(false)}>确认提交</button><button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>取消</button></div>
        </form>}
      </aside></div>}

      <DeleteConfirmModal open={Boolean(deleteClusterTarget)} title="删除 Kubernetes 集群" description={`确认删除 Kubernetes 集群 ${deleteClusterTarget?.name ?? ""}？该操作会删除关联凭据、资源快照和事件记录。`} confirming={false} onCancel={() => setDeleteClusterTarget(null)} onConfirm={() => void handleDeleteCluster()} />
      <DeleteConfirmModal open={Boolean(deleteResourceTarget)} title="删除 Kubernetes 资源" description={`确认删除 ${deleteResourceTarget?.namespace || "cluster"}/${deleteResourceTarget?.kind ?? ""}/${deleteResourceTarget?.name ?? ""}？该操作会删除集群中的真实资源。`} confirming={false} onCancel={() => setDeleteResourceTarget(null)} onConfirm={() => void handleDeleteResource()} />
      <DeleteConfirmModal open={Boolean(confirmActionTarget)} title={confirmActionTarget?.title ?? "Kubernetes 高危操作确认"} description={confirmActionTarget?.description ?? "该 Kubernetes 操作需要二次确认。"} confirming={Boolean(runningActionKey)} onCancel={() => setConfirmActionTarget(null)} onConfirm={() => void confirmKubernetesAction()} />
      <DeleteConfirmModal open={Boolean(confirmManifestTarget)} title="Kubernetes 资源真实提交确认" description="该操作会写入 Kubernetes 集群，请确认 manifest、命名空间、影响面和回滚方式。" confirmText={kubernetesConfirmSubmitText} confirmActionLabel="确认提交" confirmingLabel="提交中..." confirming={resourceSubmitting} onCancel={() => setConfirmManifestTarget(null)} onConfirm={() => void confirmResourceManifestSubmit()} />
      <TableSettingsModal open={settingsTarget !== "closed"} title="Kubernetes 列表字段" columns={settingsTarget === "resources" ? resourceColumns : settingsTarget === "operations" ? operationColumns : clusterColumns} visibleColumnKeys={settingsTarget === "resources" ? visibleResourceColumns : settingsTarget === "operations" ? visibleOperationColumns : visibleClusterColumns} onClose={() => setSettingsTarget("closed")} onToggleColumn={(key) => { if (settingsTarget === "resources") setVisibleResourceColumns((prev) => toggleColumn(prev, key, resourceColumns)); else if (settingsTarget === "operations") setVisibleOperationColumns((prev) => toggleColumn(prev, key, operationColumns)); else setVisibleClusterColumns((prev) => toggleColumn(prev, key, clusterColumns)); }} onMoveColumn={(key, direction) => { if (settingsTarget === "resources") setVisibleResourceColumns((prev) => moveColumn(prev, key, direction)); else if (settingsTarget === "operations") setVisibleOperationColumns((prev) => moveColumn(prev, key, direction)); else setVisibleClusterColumns((prev) => moveColumn(prev, key, direction)); }} onReset={() => { if (settingsTarget === "resources") setVisibleResourceColumns(sanitizeVisibleColumnKeys(defaultResourceColumns, resourceColumns)); else if (settingsTarget === "operations") setVisibleOperationColumns(sanitizeVisibleColumnKeys(defaultOperationColumns, operationColumns)); else setVisibleClusterColumns(sanitizeVisibleColumnKeys(defaultClusterColumns, clusterColumns)); }} />
    </section>
  );
}

function columnDefs(keys: string[], columns: TableSettingsColumn[]) {
  return keys.map((key) => columns.find((column) => column.key === key)).filter((column): column is TableSettingsColumn => Boolean(column));
}

function toggleColumn(current: string[], columnKey: string, columns: TableSettingsColumn[]) {
  const column = columns.find((item) => item.key === columnKey);
  if (!column || column.required) return current;
  return sanitizeVisibleColumnKeys(current.includes(columnKey) ? current.filter((key) => key !== columnKey) : [...current, columnKey], columns);
}

function moveColumn(current: string[], columnKey: string, direction: "up" | "down") {
  const index = current.indexOf(columnKey);
  if (index < 0) return current;
  const targetIndex = direction === "up" ? index - 1 : index + 1;
  if (targetIndex < 0 || targetIndex >= current.length) return current;
  const next = [...current];
  [next[index], next[targetIndex]] = [next[targetIndex], next[index]];
  return next;
}

function parseJSONObject(raw: string, label: string): Record<string, unknown> {
  const parsed = JSON.parse(raw || "{}");
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error(`${label} 必须是 JSON 对象`);
  return parsed as Record<string, unknown>;
}

function validateManifestJSON(raw: string) {
  try {
    parseJSONObject(raw, "manifest");
    return "";
  } catch (error) {
    return error instanceof Error ? error.message : "manifest 必须是合法 JSON 对象";
  }
}

function resourceManifestTemplate(resource: KubernetesResourceItem): Record<string, unknown> {
  const metadataManifest = resource.metadata?.manifest;
  if (metadataManifest && typeof metadataManifest === "object" && !Array.isArray(metadataManifest)) {
    return metadataManifest as Record<string, unknown>;
  }
  const metadata: Record<string, unknown> = { name: resource.name };
  if (resource.namespace) metadata.namespace = resource.namespace;
  if (resource.labels && Object.keys(resource.labels).length > 0) metadata.labels = resource.labels;
  return {
    apiVersion: defaultAPIVersionForKind(resource.kind),
    kind: resource.kind,
    metadata,
  };
}

function defaultAPIVersionForKind(kind: string) {
  switch (kind) {
    case "Deployment":
    case "StatefulSet":
    case "DaemonSet": return "apps/v1";
    case "Ingress": return "networking.k8s.io/v1";
    default: return "v1";
  }
}

function resourceActions(resource: KubernetesResourceItem) {
  switch (resource.kind) {
    case "Deployment": return ["restart", "scale", "pause", "resume", "delete"];
    case "StatefulSet": return ["restart", "scale", "delete"];
    case "DaemonSet": return ["restart"];
    case "Pod": return ["delete", "evict"];
    case "Node": return ["cordon", "uncordon"];
    case "Namespace": return ["delete"];
    case "ConfigMap": return ["delete"];
    case "Secret": return ["delete"];
    case "PVC":
    case "PV": return ["delete"];
    default: return [];
  }
}

function kubernetesActionNeedsConfirm(payload: KubernetesActionPayload) {
  return payload.action === "delete" || payload.kind === "Secret";
}

function totalPages(total: number, pageSize: number) {
  return Math.max(1, Math.ceil(total / pageSize));
}

function shorten(value: string) {
  if (!value || value.length <= 18) return value || "-";
  return `${value.slice(0, 12)}...${value.slice(-6)}`;
}

function formatDateTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}
