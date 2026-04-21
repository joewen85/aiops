import { FormEvent, useEffect, useMemo, useState } from "react";

import {
  createCMDBRelation,
  createCMDBResource,
  createCMDBSyncJob,
  deleteCMDBResource,
  executeCMDBResourceAction,
  getCMDBChangeImpact,
  getCMDBImpact,
  getCMDBRegionFailover,
  getCMDBResource,
  getCMDBSyncJob,
  getCMDBTopology,
  listCMDBRelations,
  listCMDBResources,
  retryCMDBSyncJob,
} from "@/api/cmdb";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { PermissionButton } from "@/components/PermissionButton";
import { RowActionOverflow } from "@/components/RowActionOverflow";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import { formatResourceBaseSpec, formatResourceExpiry } from "@/pages/cmdbResourceDisplay";
import type { CmdbRelationItem, CmdbResourceItem, CmdbSource, CmdbSyncJobDetail } from "@/types/cmdb";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const cmdbSourceOptions: CmdbSource[] = ["IaC", "CloudAPI", "K8s", "APM", "Manual"];
const relationTypeOptions = [
  "deployed_on",
  "runs_in",
  "connects_to",
  "publishes_to",
  "consumes_from",
  "fronted_by",
  "resolves_via",
  "stores_in",
  "owned_by",
  "provisioned_by",
] as const;
const CMDB_RESOURCE_LIST_SETTINGS_KEY = "cmdb.resources.table.settings";
const CMDB_RELATION_LIST_SETTINGS_KEY = "cmdb.relations.table.settings";
const defaultResourceVisibleColumnKeys = [
  "ciId",
  "type",
  "name",
  "baseSpec",
  "cloudRegion",
  "env",
  "owner",
  "source",
  "expiresAt",
  "lastSeenAt",
  "actions",
];
const defaultRelationVisibleColumnKeys = [
  "fromCiId",
  "toCiId",
  "relationType",
  "direction",
  "criticality",
  "confidence",
  "updatedAt",
];
const cmdbResourceTableColumns: TableSettingsColumn[] = [
  { key: "ciId", label: "CIID" },
  { key: "type", label: "类型" },
  { key: "name", label: "名称" },
  { key: "baseSpec", label: "基础配置" },
  { key: "cloudRegion", label: "云/地域" },
  { key: "env", label: "环境" },
  { key: "owner", label: "Owner" },
  { key: "source", label: "来源" },
  { key: "expiresAt", label: "过期时间" },
  { key: "lastSeenAt", label: "最近发现" },
  { key: "actions", label: "操作", required: true },
];
const cmdbRelationTableColumns: TableSettingsColumn[] = [
  { key: "fromCiId", label: "From" },
  { key: "toCiId", label: "To" },
  { key: "relationType", label: "关系" },
  { key: "direction", label: "方向" },
  { key: "criticality", label: "等级" },
  { key: "confidence", label: "置信度" },
  { key: "updatedAt", label: "更新时间", required: true },
];

interface ResourceFilterState {
  keyword: string;
  type: string;
  cloud: string;
  region: string;
  env: string;
}

interface RelationFilterState {
  fromCiId: string;
  toCiId: string;
  relationType: string;
}

interface ResourceFormState {
  ciId: string;
  type: string;
  name: string;
  cloud: string;
  region: string;
  env: string;
  owner: string;
  lifecycle: string;
  source: CmdbSource;
}

interface RelationFormState {
  fromCiId: string;
  toCiId: string;
  relationType: string;
  direction: string;
  criticality: string;
  confidence: string;
}

type CmdbDrawerState = "closed" | "resource-create" | "relation-create" | "resource-detail";
type TableSettingsTarget = "closed" | "resources" | "relations";

function defaultResourceFilters(): ResourceFilterState {
  return {
    keyword: "",
    type: "",
    cloud: "",
    region: "",
    env: "",
  };
}

function defaultRelationFilters(): RelationFilterState {
  return {
    fromCiId: "",
    toCiId: "",
    relationType: "",
  };
}

function defaultResourceForm(): ResourceFormState {
  return {
    ciId: "",
    type: "Service",
    name: "",
    cloud: "",
    region: "",
    env: "prod",
    owner: "",
    lifecycle: "active",
    source: "Manual",
  };
}

function defaultRelationForm(): RelationFormState {
  return {
    fromCiId: "",
    toCiId: "",
    relationType: "connects_to",
    direction: "outbound",
    criticality: "P2",
    confidence: "1",
  };
}

export function CMDBPage() {
  const [resources, setResources] = useState<CmdbResourceItem[]>([]);
  const [resourceTotal, setResourceTotal] = useState(0);
  const [resourcePage, setResourcePage] = useState(1);
  const [resourcePageSize, setResourcePageSize] = useState(defaultPageSize);
  const [resourceJumpPageInput, setResourceJumpPageInput] = useState("1");
  const [resourceFilters, setResourceFilters] = useState<ResourceFilterState>(defaultResourceFilters);
  const [resourceQuery, setResourceQuery] = useState<ResourceFilterState>(defaultResourceFilters);
  const [resourceLoading, setResourceLoading] = useState(false);
  const [resourceSubmitting, setResourceSubmitting] = useState(false);
  const [resourceDeletingId, setResourceDeletingId] = useState<number | null>(null);
  const [deleteResourceTarget, setDeleteResourceTarget] = useState<CmdbResourceItem | null>(null);
  const [resourceDetailLoading, setResourceDetailLoading] = useState(false);
  const [resourceDetail, setResourceDetail] = useState<CmdbResourceItem | null>(null);
  const [resourceActionLoadingKey, setResourceActionLoadingKey] = useState<string | null>(null);
  const [resourceForm, setResourceForm] = useState<ResourceFormState>(defaultResourceForm);

  const [relations, setRelations] = useState<CmdbRelationItem[]>([]);
  const [relationTotal, setRelationTotal] = useState(0);
  const [relationPage, setRelationPage] = useState(1);
  const [relationPageSize, setRelationPageSize] = useState(defaultPageSize);
  const [relationJumpPageInput, setRelationJumpPageInput] = useState("1");
  const [relationFilters, setRelationFilters] = useState<RelationFilterState>(defaultRelationFilters);
  const [relationQuery, setRelationQuery] = useState<RelationFilterState>(defaultRelationFilters);
  const [relationLoading, setRelationLoading] = useState(false);
  const [relationSubmitting, setRelationSubmitting] = useState(false);
  const [relationForm, setRelationForm] = useState<RelationFormState>(defaultRelationForm);
  const [drawer, setDrawer] = useState<CmdbDrawerState>("closed");
  const [tableSettingsTarget, setTableSettingsTarget] = useState<TableSettingsTarget>("closed");
  const [visibleResourceColumnKeys, setVisibleResourceColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(CMDB_RESOURCE_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultResourceVisibleColumnKeys, cmdbResourceTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, cmdbResourceTableColumns);
  });
  const [visibleRelationColumnKeys, setVisibleRelationColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(CMDB_RELATION_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultRelationVisibleColumnKeys, cmdbRelationTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, cmdbRelationTableColumns);
  });

  const [syncLoading, setSyncLoading] = useState(false);
  const [syncSources, setSyncSources] = useState<Record<CmdbSource, boolean>>({
    IaC: true,
    CloudAPI: true,
    K8s: true,
    APM: true,
    Manual: false,
  });
  const [syncFullScan, setSyncFullScan] = useState(false);
  const [syncJobDetail, setSyncJobDetail] = useState<CmdbSyncJobDetail | null>(null);

  const [topologyApplication, setTopologyApplication] = useState("");
  const [impactCiId, setImpactCiId] = useState("");
  const [failoverRegion, setFailoverRegion] = useState("");
  const [changeReleaseId, setChangeReleaseId] = useState("");
  const [analysisLoading, setAnalysisLoading] = useState(false);
  const [analysisTitle, setAnalysisTitle] = useState("结果预览");
  const [analysisResult, setAnalysisResult] = useState<string>("");

  useEffect(() => {
    void loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
  }, [resourcePage, resourcePageSize, resourceQuery]);

  useEffect(() => {
    void loadRelationPage(relationPage, relationPageSize, relationQuery);
  }, [relationPage, relationPageSize, relationQuery]);

  useEffect(() => {
    setResourceJumpPageInput(String(resourcePage));
  }, [resourcePage]);

  useEffect(() => {
    setRelationJumpPageInput(String(relationPage));
  }, [relationPage]);

  useEffect(() => {
    savePersistedListSettings(CMDB_RESOURCE_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleResourceColumnKeys,
    });
  }, [visibleResourceColumnKeys]);

  useEffect(() => {
    savePersistedListSettings(CMDB_RELATION_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleRelationColumnKeys,
    });
  }, [visibleRelationColumnKeys]);

  const resourceTotalPages = useMemo(() => totalPages(resourceTotal, resourcePageSize), [resourceTotal, resourcePageSize]);
  const relationTotalPages = useMemo(() => totalPages(relationTotal, relationPageSize), [relationTotal, relationPageSize]);
  const ciIdOptions = useMemo(() => resources.map((item) => item.ciId).filter(Boolean), [resources]);
  const resourceVisibleColumnSet = useMemo(() => new Set(visibleResourceColumnKeys), [visibleResourceColumnKeys]);
  const relationVisibleColumnSet = useMemo(() => new Set(visibleRelationColumnKeys), [visibleRelationColumnKeys]);
  const resourceColSpan = Math.max(1, visibleResourceColumnKeys.length);
  const relationColSpan = Math.max(1, visibleRelationColumnKeys.length);
  const selectedSyncSources = useMemo(
    () => cmdbSourceOptions.filter((source) => syncSources[source]),
    [syncSources],
  );
  const drawerVisible = drawer !== "closed";

  async function loadResourcePage(page: number, pageSize: number, filters: ResourceFilterState) {
    setResourceLoading(true);
    try {
      const data = await listCMDBResources({
        page,
        pageSize,
        keyword: filters.keyword,
        type: filters.type,
        cloud: filters.cloud,
        region: filters.region,
        env: filters.env,
      });
      const pages = totalPages(data.total, pageSize);
      if (page > pages) {
        setResourcePage(pages);
        return;
      }
      setResources(data.list);
      setResourceTotal(data.total);
    } catch {
      showToast("CMDB 资源加载失败");
    } finally {
      setResourceLoading(false);
    }
  }

  async function loadRelationPage(page: number, pageSize: number, filters: RelationFilterState) {
    setRelationLoading(true);
    try {
      const data = await listCMDBRelations({
        page,
        pageSize,
        fromCiId: filters.fromCiId,
        toCiId: filters.toCiId,
        relationType: filters.relationType,
      });
      const pages = totalPages(data.total, pageSize);
      if (page > pages) {
        setRelationPage(pages);
        return;
      }
      setRelations(data.list);
      setRelationTotal(data.total);
    } catch {
      showToast("CMDB 关系加载失败");
    } finally {
      setRelationLoading(false);
    }
  }

  function handleResourceFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setResourcePage(1);
    setResourceQuery({ ...resourceFilters });
  }

  function handleRelationFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRelationPage(1);
    setRelationQuery({ ...relationFilters });
  }

  function closeDrawer() {
    setDrawer("closed");
    setResourceForm(defaultResourceForm());
    setRelationForm(defaultRelationForm());
    setResourceDetail(null);
    setResourceDetailLoading(false);
  }

  function openResourceCreateDrawer() {
    setResourceForm(defaultResourceForm());
    setDrawer("resource-create");
  }

  function openRelationCreateDrawer() {
    setRelationForm(defaultRelationForm());
    setDrawer("relation-create");
  }

  async function openResourceDetailDrawer(resourceId: number) {
    setDrawer("resource-detail");
    setResourceDetail(null);
    setResourceDetailLoading(true);
    try {
      const detail = await getCMDBResource(resourceId);
      setResourceDetail(detail);
    } catch {
      showToast("资源详情加载失败");
      setDrawer("closed");
    } finally {
      setResourceDetailLoading(false);
    }
  }

  function toggleResourceVisibleColumn(columnKey: string) {
    setVisibleResourceColumnKeys((prev) => {
      const exists = prev.includes(columnKey);
      if (exists) return prev.filter((key) => key !== columnKey);
      return [...prev, columnKey];
    });
  }

  function toggleRelationVisibleColumn(columnKey: string) {
    setVisibleRelationColumnKeys((prev) => {
      const exists = prev.includes(columnKey);
      if (exists) return prev.filter((key) => key !== columnKey);
      return [...prev, columnKey];
    });
  }

  async function handleCreateResource(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!resourceForm.name.trim() || !resourceForm.type.trim()) {
      showToast("资源类型和名称必填");
      return;
    }
    setResourceSubmitting(true);
    try {
      await createCMDBResource({
        ciId: resourceForm.ciId.trim() || undefined,
        type: resourceForm.type.trim(),
        name: resourceForm.name.trim(),
        cloud: resourceForm.cloud.trim() || undefined,
        region: resourceForm.region.trim() || undefined,
        env: resourceForm.env.trim() || undefined,
        owner: resourceForm.owner.trim() || undefined,
        lifecycle: resourceForm.lifecycle.trim() || undefined,
        source: resourceForm.source,
      });
      setResourceForm(defaultResourceForm());
      setDrawer("closed");
      showToast("CI 资源创建成功");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
    } catch {
      showToast("CI 资源创建失败");
    } finally {
      setResourceSubmitting(false);
    }
  }

  async function handleDeleteResource(resourceId: number) {
    setResourceDeletingId(resourceId);
    try {
      await deleteCMDBResource(resourceId);
      showToast("CI 资源已删除");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
      await loadRelationPage(relationPage, relationPageSize, relationQuery);
    } catch {
      showToast("CI 资源删除失败");
    } finally {
      setResourceDeletingId(null);
    }
  }

  function requestDeleteResource(resource: CmdbResourceItem) {
    if (isResourceRunning(resource)) {
      showToast("运行中资源不可删除");
      return;
    }
    setDeleteResourceTarget(resource);
  }

  async function confirmDeleteResource() {
    if (!deleteResourceTarget) return;
    const deletingID = deleteResourceTarget.id;
    await handleDeleteResource(deletingID);
    setDeleteResourceTarget(null);
  }

  async function handleVMAction(resource: CmdbResourceItem, action: "restart" | "stop") {
    if (action === "stop" && !window.confirm("确认停止该云服务器？")) {
      return;
    }
    const actionKey = `${resource.id}:${action}`;
    setResourceActionLoadingKey(actionKey);
    try {
      await executeCMDBResourceAction(resource.id, action);
      showToast(action === "restart" ? "已发起重启请求" : "已发起停止请求");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
      if (drawer === "resource-detail" && resourceDetail?.id === resource.id) {
        const detail = await getCMDBResource(resource.id);
        setResourceDetail(detail);
      }
    } catch {
      showToast(action === "restart" ? "重启请求失败" : "停止请求失败");
    } finally {
      setResourceActionLoadingKey(null);
    }
  }

  async function handleCreateRelation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!relationForm.fromCiId.trim() || !relationForm.toCiId.trim()) {
      showToast("关系两端 CIID 必填");
      return;
    }
    const confidence = Number(relationForm.confidence);
    if (Number.isNaN(confidence) || confidence < 0 || confidence > 1) {
      showToast("confidence 需在 0 到 1 之间");
      return;
    }
    setRelationSubmitting(true);
    try {
      await createCMDBRelation({
        fromCiId: relationForm.fromCiId.trim(),
        toCiId: relationForm.toCiId.trim(),
        relationType: relationForm.relationType,
        direction: relationForm.direction,
        criticality: relationForm.criticality,
        confidence,
        evidence: { source: "Manual", note: "frontend create" },
      });
      setRelationForm(defaultRelationForm());
      setDrawer("closed");
      showToast("关系创建成功");
      await loadRelationPage(relationPage, relationPageSize, relationQuery);
    } catch {
      showToast("关系创建失败");
    } finally {
      setRelationSubmitting(false);
    }
  }

  async function handleCreateSyncJob() {
    setSyncLoading(true);
    try {
      const job = await createCMDBSyncJob({
        sources: selectedSyncSources,
        fullScan: syncFullScan,
      });
      const detail = await getCMDBSyncJob(job.id);
      setSyncJobDetail(detail);
      showToast("同步任务已执行");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
      await loadRelationPage(relationPage, relationPageSize, relationQuery);
    } catch {
      showToast("同步任务执行失败");
    } finally {
      setSyncLoading(false);
    }
  }

  async function handleRetrySyncJob() {
    if (!syncJobDetail?.job?.id) {
      showToast("暂无可重试任务");
      return;
    }
    setSyncLoading(true);
    try {
      const job = await retryCMDBSyncJob(syncJobDetail.job.id);
      const detail = await getCMDBSyncJob(job.id);
      setSyncJobDetail(detail);
      showToast("同步任务已重试");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
      await loadRelationPage(relationPage, relationPageSize, relationQuery);
    } catch {
      showToast("同步任务重试失败");
    } finally {
      setSyncLoading(false);
    }
  }

  async function runAnalysis(
    title: string,
    fn: () => Promise<Record<string, unknown>>,
  ) {
    setAnalysisLoading(true);
    try {
      const data = await fn();
      setAnalysisTitle(title);
      setAnalysisResult(JSON.stringify(data, null, 2));
    } catch {
      showToast(`${title} 查询失败`);
    } finally {
      setAnalysisLoading(false);
    }
  }

  return (
    <section className="page">
      <h2>CMDB</h2>

      <article className="card">
        <h3>统一模型 + 关系图谱 + 自动采集 + 数据治理</h3>
        <p className="muted">围绕 CI、关系、采集任务三条主线，支持影响分析和地域容灾视图。</p>
        <div className="grid cards">
          <div className="card">
            <h4>CI 资源</h4>
            <p>{resourceTotal} 条</p>
          </div>
          <div className="card">
            <h4>关系边</h4>
            <p>{relationTotal} 条</p>
          </div>
          <div className="card">
            <h4>最近同步</h4>
            <p>{syncJobDetail?.job?.status ?? "未执行"}</p>
          </div>
        </div>
      </article>

      <div className="rbac-module-grid">
        <article className="card rbac-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>CI 资源视图</h3>
              <p className="muted">支持关键字段过滤、分页和快速创建资源。</p>
            </div>
            <div className="rbac-actions">
              <PermissionButton
                permissionKey="button.cmdb.resource.create"
                className="btn cursor-pointer"
                type="button"
                onClick={openResourceCreateDrawer}
              >
                创建 CI
              </PermissionButton>
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => void loadResourcePage(resourcePage, resourcePageSize, resourceQuery)}
                disabled={resourceLoading}
              >
                刷新
              </button>
            </div>
          </header>

          <form className="cloud-filter-bar" onSubmit={handleResourceFilterSubmit}>
            <input
              className="cloud-filter-control cloud-filter-keyword"
              value={resourceFilters.keyword}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词（name/ciId）"
            />
            <input
              className="cloud-filter-control"
              value={resourceFilters.type}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, type: event.target.value }))}
              placeholder="type"
            />
            <input
              className="cloud-filter-control"
              value={resourceFilters.cloud}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, cloud: event.target.value }))}
              placeholder="cloud"
            />
            <input
              className="cloud-filter-control"
              value={resourceFilters.region}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, region: event.target.value }))}
              placeholder="region"
            />
            <input
              className="cloud-filter-control"
              value={resourceFilters.env}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, env: event.target.value }))}
              placeholder="env"
            />
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={resourceLoading}>查询</button>
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => {
                  const defaults = defaultResourceFilters();
                  setResourceFilters(defaults);
                  setResourceQuery(defaults);
                  setResourcePage(1);
                }}
              >
                重置
              </button>
            </div>
          </form>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  {resourceVisibleColumnSet.has("ciId") && <th>CIID</th>}
                  {resourceVisibleColumnSet.has("type") && <th>类型</th>}
                  {resourceVisibleColumnSet.has("name") && <th>名称</th>}
                  {resourceVisibleColumnSet.has("baseSpec") && <th>基础配置</th>}
                  {resourceVisibleColumnSet.has("cloudRegion") && <th>云/地域</th>}
                  {resourceVisibleColumnSet.has("env") && <th>环境</th>}
                  {resourceVisibleColumnSet.has("owner") && <th>Owner</th>}
                  {resourceVisibleColumnSet.has("source") && <th>来源</th>}
                  {resourceVisibleColumnSet.has("expiresAt") && <th>过期时间</th>}
                  {resourceVisibleColumnSet.has("lastSeenAt") && <th>最近发现</th>}
                  {resourceVisibleColumnSet.has("actions") && (
                    <th>
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button
                          className="table-settings-trigger cursor-pointer"
                          type="button"
                          onClick={() => setTableSettingsTarget("resources")}
                          aria-label="CI资源列表设置"
                        >
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {resourceLoading
                  ? (
                    <tr><td colSpan={resourceColSpan}>加载中...</td></tr>
                  )
                  : resources.length === 0
                    ? (
                      <tr><td colSpan={resourceColSpan}>暂无数据</td></tr>
                    )
                    : resources.map((item) => (
                      <tr key={item.id}>
                        {resourceVisibleColumnSet.has("ciId") && <td>{item.ciId || "-"}</td>}
                        {resourceVisibleColumnSet.has("type") && <td>{item.type || "-"}</td>}
                        {resourceVisibleColumnSet.has("name") && <td>{item.name || "-"}</td>}
                        {resourceVisibleColumnSet.has("baseSpec") && <td>{formatResourceBaseSpec(item)}</td>}
                        {resourceVisibleColumnSet.has("cloudRegion") && <td>{item.cloud || "-"}/{item.region || "-"}</td>}
                        {resourceVisibleColumnSet.has("env") && <td>{item.env || "-"}</td>}
                        {resourceVisibleColumnSet.has("owner") && <td>{item.owner || "-"}</td>}
                        {resourceVisibleColumnSet.has("source") && <td>{item.source || "-"}</td>}
                        {resourceVisibleColumnSet.has("expiresAt") && <td>{formatResourceExpiry(item)}</td>}
                        {resourceVisibleColumnSet.has("lastSeenAt") && <td>{formatDateTime(item.lastSeenAt)}</td>}
                        {resourceVisibleColumnSet.has("actions") && (
                          <td>
                            <div className="rbac-row-actions">
                              <RowActionOverflow
                                title="CI资源更多操作"
                                actions={[
                                  {
                                    key: `${item.id}-detail`,
                                    label: "详情",
                                    permissionKey: "button.cmdb.resource.detail",
                                    onClick: () => void openResourceDetailDrawer(item.id),
                                  },
                                  ...(isVMType(item.type)
                                    ? [
                                      {
                                        key: `${item.id}-restart`,
                                        label: resourceActionLoadingKey === `${item.id}:restart` ? "重启中..." : "重启",
                                        permissionKey: "button.cmdb.resource.restart",
                                        disabled: resourceActionLoadingKey === `${item.id}:restart`,
                                        onClick: () => void handleVMAction(item, "restart"),
                                      },
                                      {
                                        key: `${item.id}-stop`,
                                        label: resourceActionLoadingKey === `${item.id}:stop` ? "停止中..." : "停止",
                                        permissionKey: "button.cmdb.resource.stop",
                                        disabled: resourceActionLoadingKey === `${item.id}:stop`,
                                        onClick: () => void handleVMAction(item, "stop"),
                                      },
                                    ]
                                    : []),
                                  {
                                    key: `${item.id}-delete`,
                                    label: resourceDeletingId === item.id ? "删除中..." : "删除",
                                    permissionKey: "button.cmdb.resource.delete",
                                    disabled: resourceDeletingId === item.id || isResourceRunning(item),
                                    onClick: () => requestDeleteResource(item),
                                  },
                                ]}
                              />
                            </div>
                          </td>
                        )}
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>

          <footer className="rbac-pagination">
            <div className="rbac-pagination-group">
              <span>总计 {resourceTotal} 条</span>
              <select
                className="rbac-pagination-select cursor-pointer"
                value={resourcePageSize}
                onChange={(event) => {
                  setResourcePageSize(Number(event.target.value));
                  setResourcePage(1);
                }}
              >
                {pageSizeOptions.map((option) => (
                  <option key={option} value={option}>{option}/页</option>
                ))}
              </select>
            </div>
            <div className="rbac-pagination-group">
              <button
                className="btn cursor-pointer"
                type="button"
                disabled={resourcePage <= 1}
                onClick={() => setResourcePage((page) => Math.max(1, page - 1))}
              >
                上一页
              </button>
              <span className="rbac-pagination-text">{resourcePage} / {resourceTotalPages}</span>
              <button
                className="btn cursor-pointer"
                type="button"
                disabled={resourcePage >= resourceTotalPages}
                onClick={() => setResourcePage((page) => Math.min(resourceTotalPages, page + 1))}
              >
                下一页
              </button>
              <input
                className="rbac-pagination-input"
                value={resourceJumpPageInput}
                onChange={(event) => setResourceJumpPageInput(event.target.value)}
                placeholder="页码"
              />
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => {
                  const target = Number(resourceJumpPageInput);
                  if (Number.isNaN(target)) return;
                  setResourcePage(clamp(target, 1, resourceTotalPages));
                }}
              >
                跳转
              </button>
            </div>
          </footer>
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>关系图谱视图</h3>
              <p className="muted">固化 10 类关系，支持按节点和关系类型过滤。</p>
            </div>
            <div className="rbac-actions">
              <PermissionButton
                permissionKey="button.cmdb.relation.create"
                className="btn cursor-pointer"
                type="button"
                onClick={openRelationCreateDrawer}
              >
                创建关系
              </PermissionButton>
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => void loadRelationPage(relationPage, relationPageSize, relationQuery)}
                disabled={relationLoading}
              >
                刷新
              </button>
            </div>
          </header>

          <form className="cloud-filter-bar" onSubmit={handleRelationFilterSubmit}>
            <input
              className="cloud-filter-control"
              list="cmdb-ciid-options"
              value={relationFilters.fromCiId}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, fromCiId: event.target.value }))}
              placeholder="fromCiId"
            />
            <input
              className="cloud-filter-control"
              list="cmdb-ciid-options"
              value={relationFilters.toCiId}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, toCiId: event.target.value }))}
              placeholder="toCiId"
            />
            <select
              className="cloud-filter-control"
              value={relationFilters.relationType}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, relationType: event.target.value }))}
            >
              <option value="">全部关系类型</option>
              {relationTypeOptions.map((type) => (
                <option key={type} value={type}>{type}</option>
              ))}
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={relationLoading}>查询</button>
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => {
                  const defaults = defaultRelationFilters();
                  setRelationFilters(defaults);
                  setRelationQuery(defaults);
                  setRelationPage(1);
                }}
              >
                重置
              </button>
            </div>
          </form>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  {relationVisibleColumnSet.has("fromCiId") && <th>From</th>}
                  {relationVisibleColumnSet.has("toCiId") && <th>To</th>}
                  {relationVisibleColumnSet.has("relationType") && <th>关系</th>}
                  {relationVisibleColumnSet.has("direction") && <th>方向</th>}
                  {relationVisibleColumnSet.has("criticality") && <th>等级</th>}
                  {relationVisibleColumnSet.has("confidence") && <th>置信度</th>}
                  {relationVisibleColumnSet.has("updatedAt") && (
                    <th>
                      <div className="table-actions-header">
                        <span>更新时间</span>
                        <button
                          className="table-settings-trigger cursor-pointer"
                          type="button"
                          onClick={() => setTableSettingsTarget("relations")}
                          aria-label="关系列表设置"
                        >
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {relationLoading
                  ? <tr><td colSpan={relationColSpan}>加载中...</td></tr>
                  : relations.length === 0
                    ? <tr><td colSpan={relationColSpan}>暂无数据</td></tr>
                    : relations.map((item) => (
                      <tr key={item.id}>
                        {relationVisibleColumnSet.has("fromCiId") && <td>{item.fromCiId}</td>}
                        {relationVisibleColumnSet.has("toCiId") && <td>{item.toCiId}</td>}
                        {relationVisibleColumnSet.has("relationType") && <td>{item.relationType}</td>}
                        {relationVisibleColumnSet.has("direction") && <td>{item.direction || "-"}</td>}
                        {relationVisibleColumnSet.has("criticality") && <td>{item.criticality || "-"}</td>}
                        {relationVisibleColumnSet.has("confidence") && <td>{typeof item.confidence === "number" ? item.confidence.toFixed(2) : "-"}</td>}
                        {relationVisibleColumnSet.has("updatedAt") && <td>{formatDateTime(item.relationUpdatedAt || item.updatedAt)}</td>}
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>

          <footer className="rbac-pagination">
            <div className="rbac-pagination-group">
              <span>总计 {relationTotal} 条</span>
              <select
                className="rbac-pagination-select cursor-pointer"
                value={relationPageSize}
                onChange={(event) => {
                  setRelationPageSize(Number(event.target.value));
                  setRelationPage(1);
                }}
              >
                {pageSizeOptions.map((option) => (
                  <option key={option} value={option}>{option}/页</option>
                ))}
              </select>
            </div>
            <div className="rbac-pagination-group">
              <button
                className="btn cursor-pointer"
                type="button"
                disabled={relationPage <= 1}
                onClick={() => setRelationPage((page) => Math.max(1, page - 1))}
              >
                上一页
              </button>
              <span className="rbac-pagination-text">{relationPage} / {relationTotalPages}</span>
              <button
                className="btn cursor-pointer"
                type="button"
                disabled={relationPage >= relationTotalPages}
                onClick={() => setRelationPage((page) => Math.min(relationTotalPages, page + 1))}
              >
                下一页
              </button>
              <input
                className="rbac-pagination-input"
                value={relationJumpPageInput}
                onChange={(event) => setRelationJumpPageInput(event.target.value)}
                placeholder="页码"
              />
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => {
                  const target = Number(relationJumpPageInput);
                  if (Number.isNaN(target)) return;
                  setRelationPage(clamp(target, 1, relationTotalPages));
                }}
              >
                跳转
              </button>
            </div>
          </footer>
        </article>

        <article className="card rbac-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>采集任务与影响分析</h3>
              <p className="muted">执行多源采集任务，并直接查询拓扑、故障与变更影响。</p>
            </div>
          </header>

          <div className="form-grid">
            <div className="rbac-actions">
              {cmdbSourceOptions.map((source) => (
                <label key={source} className="rbac-binding-toggle">
                  <input
                    type="checkbox"
                    checked={syncSources[source]}
                    onChange={(event) => setSyncSources((prev) => ({ ...prev, [source]: event.target.checked }))}
                  />
                  {source}
                </label>
              ))}
              <label className="rbac-binding-toggle">
                <input
                  type="checkbox"
                  checked={syncFullScan}
                  onChange={(event) => setSyncFullScan(event.target.checked)}
                />
                Full Scan
              </label>
              <PermissionButton
                permissionKey="button.cmdb.sync.create"
                className="btn primary cursor-pointer"
                type="button"
                onClick={() => void handleCreateSyncJob()}
                disabled={syncLoading}
              >
                {syncLoading ? "执行中..." : "执行同步"}
              </PermissionButton>
              <PermissionButton
                permissionKey="button.cmdb.sync.retry"
                className="btn cursor-pointer"
                type="button"
                onClick={() => void handleRetrySyncJob()}
                disabled={syncLoading || !syncJobDetail?.job?.id}
              >
                重试最近任务
              </PermissionButton>
            </div>
            <div className="rbac-kv-grid">
              <div>
                <span className="muted">任务 ID</span>
                <strong>{syncJobDetail?.job?.id ?? "-"}</strong>
              </div>
              <div>
                <span className="muted">状态</span>
                <strong>{syncJobDetail?.job?.status ?? "-"}</strong>
              </div>
              <div>
                <span className="muted">请求来源</span>
                <strong>{selectedSyncSources.join(", ") || "-"}</strong>
              </div>
              <div>
                <span className="muted">任务明细</span>
                <strong>{syncJobDetail?.items?.length ?? 0} 条</strong>
              </div>
            </div>
          </div>

          <div className="form-grid">
            <div className="cloud-filter-bar">
              <input
                className="cloud-filter-control cloud-filter-keyword"
                value={topologyApplication}
                onChange={(event) => setTopologyApplication(event.target.value)}
                placeholder="application（拓扑）"
              />
              <div className="cloud-filter-actions">
                <button
                  className="btn cursor-pointer"
                  type="button"
                  disabled={analysisLoading || !topologyApplication.trim()}
                  onClick={() => void runAnalysis(
                    `业务拓扑：${topologyApplication.trim()}`,
                    () => getCMDBTopology(topologyApplication.trim()),
                  )}
                >
                  查询拓扑
                </button>
              </div>
            </div>
            <div className="cloud-filter-bar">
              <input
                className="cloud-filter-control cloud-filter-keyword"
                value={impactCiId}
                onChange={(event) => setImpactCiId(event.target.value)}
                placeholder="ciId（故障影响）"
              />
              <div className="cloud-filter-actions">
                <button
                  className="btn cursor-pointer"
                  type="button"
                  disabled={analysisLoading || !impactCiId.trim()}
                  onClick={() => void runAnalysis(
                    `故障影响：${impactCiId.trim()}`,
                    () => getCMDBImpact(impactCiId.trim()),
                  )}
                >
                  查询影响
                </button>
              </div>
            </div>
            <div className="cloud-filter-bar">
              <input
                className="cloud-filter-control cloud-filter-keyword"
                value={failoverRegion}
                onChange={(event) => setFailoverRegion(event.target.value)}
                placeholder="region（地域容灾）"
              />
              <div className="cloud-filter-actions">
                <button
                  className="btn cursor-pointer"
                  type="button"
                  disabled={analysisLoading || !failoverRegion.trim()}
                  onClick={() => void runAnalysis(
                    `地域容灾：${failoverRegion.trim()}`,
                    () => getCMDBRegionFailover(failoverRegion.trim()),
                  )}
                >
                  查询容灾
                </button>
              </div>
            </div>
            <div className="cloud-filter-bar">
              <input
                className="cloud-filter-control cloud-filter-keyword"
                value={changeReleaseId}
                onChange={(event) => setChangeReleaseId(event.target.value)}
                placeholder="releaseId（变更影响）"
              />
              <div className="cloud-filter-actions">
                <button
                  className="btn cursor-pointer"
                  type="button"
                  disabled={analysisLoading || !changeReleaseId.trim()}
                  onClick={() => void runAnalysis(
                    `变更影响：${changeReleaseId.trim()}`,
                    () => getCMDBChangeImpact(changeReleaseId.trim()),
                  )}
                >
                  查询变更
                </button>
              </div>
            </div>
          </div>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <pre className="cmdb-analysis">
              <strong>{analysisTitle}</strong>
              {"\n"}
              {analysisResult || "暂无结果，请先执行查询。"}
            </pre>
          </div>
        </article>
      </div>

      <TableSettingsModal
        open={tableSettingsTarget === "resources"}
        title="CI资源列表设置"
        columns={cmdbResourceTableColumns}
        visibleColumnKeys={visibleResourceColumnKeys}
        onToggleColumn={toggleResourceVisibleColumn}
        onReset={() => setVisibleResourceColumnKeys(sanitizeVisibleColumnKeys(defaultResourceVisibleColumnKeys, cmdbResourceTableColumns))}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <TableSettingsModal
        open={tableSettingsTarget === "relations"}
        title="关系列表设置"
        columns={cmdbRelationTableColumns}
        visibleColumnKeys={visibleRelationColumnKeys}
        onToggleColumn={toggleRelationVisibleColumn}
        onReset={() => setVisibleRelationColumnKeys(sanitizeVisibleColumnKeys(defaultRelationVisibleColumnKeys, cmdbRelationTableColumns))}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <datalist id="cmdb-ciid-options">
        {ciIdOptions.map((ciId) => (
          <option key={ciId} value={ciId} />
        ))}
      </datalist>

      {drawerVisible && (
        <div className="rbac-drawer-mask" onClick={closeDrawer}>
          <aside className="rbac-drawer" onClick={(event) => event.stopPropagation()}>
            <header className="rbac-drawer-header">
              <h3>
                {drawer === "resource-create"
                  ? "创建 CI 资源"
                  : drawer === "relation-create"
                    ? "创建关系"
                    : "资源详情"}
              </h3>
              <button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>
                关闭
              </button>
            </header>
            <div className="rbac-drawer-body">
              {drawer === "resource-create" && (
                <form className="form-grid" onSubmit={handleCreateResource}>
                  <label htmlFor="cmdb-resource-ciid">ciId（可留空自动生成）</label>
                  <input
                    id="cmdb-resource-ciid"
                    value={resourceForm.ciId}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, ciId: event.target.value }))}
                    placeholder="ciId（可留空自动生成）"
                  />
                  <label htmlFor="cmdb-resource-type">类型</label>
                  <input
                    id="cmdb-resource-type"
                    required
                    value={resourceForm.type}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, type: event.target.value }))}
                    placeholder="类型（如 Service/K8sCluster）"
                  />
                  <label htmlFor="cmdb-resource-name">资源名称</label>
                  <input
                    id="cmdb-resource-name"
                    required
                    value={resourceForm.name}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="资源名称"
                  />
                  <label htmlFor="cmdb-resource-cloud">cloud</label>
                  <input
                    id="cmdb-resource-cloud"
                    value={resourceForm.cloud}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, cloud: event.target.value }))}
                    placeholder="cloud"
                  />
                  <label htmlFor="cmdb-resource-region">region</label>
                  <input
                    id="cmdb-resource-region"
                    value={resourceForm.region}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, region: event.target.value }))}
                    placeholder="region"
                  />
                  <label htmlFor="cmdb-resource-env">env</label>
                  <input
                    id="cmdb-resource-env"
                    value={resourceForm.env}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, env: event.target.value }))}
                    placeholder="env"
                  />
                  <label htmlFor="cmdb-resource-owner">owner</label>
                  <input
                    id="cmdb-resource-owner"
                    value={resourceForm.owner}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, owner: event.target.value }))}
                    placeholder="owner"
                  />
                  <label htmlFor="cmdb-resource-lifecycle">lifecycle</label>
                  <input
                    id="cmdb-resource-lifecycle"
                    value={resourceForm.lifecycle}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, lifecycle: event.target.value }))}
                    placeholder="lifecycle"
                  />
                  <label htmlFor="cmdb-resource-source">来源</label>
                  <select
                    id="cmdb-resource-source"
                    value={resourceForm.source}
                    onChange={(event) => setResourceForm((prev) => ({ ...prev, source: event.target.value as CmdbSource }))}
                  >
                    {cmdbSourceOptions.map((source) => (
                      <option key={source} value={source}>{source}</option>
                    ))}
                  </select>
                  <PermissionButton
                    permissionKey="button.cmdb.resource.create"
                    className="btn primary cursor-pointer"
                    type="submit"
                    disabled={resourceSubmitting}
                  >
                    {resourceSubmitting ? "创建中..." : "创建 CI"}
                  </PermissionButton>
                </form>
              )}

              {drawer === "relation-create" && (
                <form className="form-grid" onSubmit={handleCreateRelation}>
                  <label htmlFor="cmdb-relation-from">fromCiId</label>
                  <input
                    id="cmdb-relation-from"
                    list="cmdb-ciid-options"
                    value={relationForm.fromCiId}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, fromCiId: event.target.value }))}
                    placeholder="fromCiId"
                    required
                  />
                  <label htmlFor="cmdb-relation-to">toCiId</label>
                  <input
                    id="cmdb-relation-to"
                    list="cmdb-ciid-options"
                    value={relationForm.toCiId}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, toCiId: event.target.value }))}
                    placeholder="toCiId"
                    required
                  />
                  <label htmlFor="cmdb-relation-type">关系类型</label>
                  <select
                    id="cmdb-relation-type"
                    value={relationForm.relationType}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, relationType: event.target.value }))}
                  >
                    {relationTypeOptions.map((type) => (
                      <option key={type} value={type}>{type}</option>
                    ))}
                  </select>
                  <label htmlFor="cmdb-relation-direction">方向</label>
                  <select
                    id="cmdb-relation-direction"
                    value={relationForm.direction}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, direction: event.target.value }))}
                  >
                    <option value="outbound">outbound</option>
                    <option value="inbound">inbound</option>
                    <option value="bidirectional">bidirectional</option>
                  </select>
                  <label htmlFor="cmdb-relation-criticality">等级</label>
                  <select
                    id="cmdb-relation-criticality"
                    value={relationForm.criticality}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, criticality: event.target.value }))}
                  >
                    <option value="P0">P0</option>
                    <option value="P1">P1</option>
                    <option value="P2">P2</option>
                    <option value="P3">P3</option>
                  </select>
                  <label htmlFor="cmdb-relation-confidence">置信度</label>
                  <input
                    id="cmdb-relation-confidence"
                    value={relationForm.confidence}
                    onChange={(event) => setRelationForm((prev) => ({ ...prev, confidence: event.target.value }))}
                    placeholder="confidence (0~1)"
                  />
                  <PermissionButton
                    permissionKey="button.cmdb.relation.create"
                    className="btn primary cursor-pointer"
                    type="submit"
                    disabled={relationSubmitting}
                  >
                    {relationSubmitting ? "创建中..." : "创建关系"}
                  </PermissionButton>
                </form>
              )}

              {drawer === "resource-detail" && (
                resourceDetailLoading
                  ? <p className="muted">资源详情加载中...</p>
                  : resourceDetail
                    ? (
                      <div className="rbac-detail-stack">
                        <div className="rbac-kv-grid">
                          <div><strong>ID</strong><span>{resourceDetail.id}</span></div>
                          <div><strong>CIID</strong><span>{resourceDetail.ciId || "-"}</span></div>
                          <div><strong>类型</strong><span>{resourceDetail.type || "-"}</span></div>
                          <div><strong>名称</strong><span>{resourceDetail.name || "-"}</span></div>
                          <div><strong>云厂商</strong><span>{resourceDetail.cloud || "-"}</span></div>
                          <div><strong>地域</strong><span>{resourceDetail.region || "-"}</span></div>
                          <div><strong>环境</strong><span>{resourceDetail.env || "-"}</span></div>
                          <div><strong>Owner</strong><span>{resourceDetail.owner || "-"}</span></div>
                          <div><strong>生命周期</strong><span>{resourceDetail.lifecycle || "-"}</span></div>
                          <div><strong>来源</strong><span>{resourceDetail.source || "-"}</span></div>
                          <div><strong>最近发现</strong><span>{formatDateTime(resourceDetail.lastSeenAt)}</span></div>
                          <div><strong>更新时间</strong><span>{formatDateTime(resourceDetail.updatedAt)}</span></div>
                        </div>
                        <div className="rbac-kv-grid">
                          <div>
                            <strong>基础配置摘要</strong>
                            <span>{formatResourceBaseSpec(resourceDetail)}</span>
                          </div>
                          <div>
                            <strong>过期时间</strong>
                            <span>{formatResourceExpiry(resourceDetail)}</span>
                          </div>
                        </div>
                        <div className="rbac-kv-grid">
                          <div>
                            <strong>Attributes</strong>
                            <pre className="cmdb-analysis">{formatPrettyJSON(resourceDetail.attributes)}</pre>
                          </div>
                        </div>
                      </div>
                    )
                    : <p className="muted">暂无详情数据</p>
              )}
            </div>
          </aside>
        </div>
      )}

      <DeleteConfirmModal
        open={deleteResourceTarget !== null}
        title="删除资源确认"
        description={`将删除资源：${deleteResourceTarget?.name || deleteResourceTarget?.ciId || "-"}`}
        confirming={deleteResourceTarget !== null && resourceDeletingId === deleteResourceTarget.id}
        onCancel={() => setDeleteResourceTarget(null)}
        onConfirm={() => void confirmDeleteResource()}
      />
    </section>
  );
}

function totalPages(total: number, pageSize: number): number {
  if (pageSize <= 0) return 1;
  return Math.max(1, Math.ceil(total / pageSize));
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function formatDateTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function isVMType(resourceType?: string): boolean {
  const value = (resourceType ?? "").trim().toLowerCase();
  return value === "vm" || value === "compute" || value === "ecs" || value === "ec2" || value === "cloudserver";
}

function isResourceRunning(resource: CmdbResourceItem): boolean {
  const status = String(resource.attributes?.status ?? "").trim().toLowerCase();
  return status === "running" || status === "运行中";
}

function formatPrettyJSON(value: unknown): string {
  if (value === undefined || value === null) return "{}";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
