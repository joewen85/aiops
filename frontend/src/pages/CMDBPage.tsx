import { FormEvent, useEffect, useMemo, useState } from "react";

import {
  createCMDBRelation,
  createCMDBResource,
  createCMDBSyncJob,
  deleteCMDBResource,
  getCMDBChangeImpact,
  getCMDBImpact,
  getCMDBRegionFailover,
  getCMDBSyncJob,
  getCMDBTopology,
  listCMDBRelations,
  listCMDBResources,
  retryCMDBSyncJob,
} from "@/api/cmdb";
import { PermissionButton } from "@/components/PermissionButton";
import { formatResourceBaseSpec, formatResourceExpiry } from "@/pages/cmdbResourceDisplay";
import type { CmdbRelationItem, CmdbResourceItem, CmdbSource, CmdbSyncJobDetail } from "@/types/cmdb";
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
  const [resourceForm, setResourceForm] = useState<ResourceFormState>(defaultResourceForm);
  const [showResourceForm, setShowResourceForm] = useState(false);

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
  const [showRelationForm, setShowRelationForm] = useState(false);

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

  const resourceTotalPages = useMemo(() => totalPages(resourceTotal, resourcePageSize), [resourceTotal, resourcePageSize]);
  const relationTotalPages = useMemo(() => totalPages(relationTotal, relationPageSize), [relationTotal, relationPageSize]);
  const ciIdOptions = useMemo(() => resources.map((item) => item.ciId).filter(Boolean), [resources]);
  const selectedSyncSources = useMemo(
    () => cmdbSourceOptions.filter((source) => syncSources[source]),
    [syncSources],
  );

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
      setShowResourceForm(false);
      showToast("CI 资源创建成功");
      await loadResourcePage(resourcePage, resourcePageSize, resourceQuery);
    } catch {
      showToast("CI 资源创建失败");
    } finally {
      setResourceSubmitting(false);
    }
  }

  async function handleDeleteResource(resourceId: number) {
    if (!window.confirm("确认删除该资源？")) return;
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
      setShowRelationForm(false);
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
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => setShowResourceForm((value) => !value)}
              >
                {showResourceForm ? "收起创建" : "展开创建"}
              </button>
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

          {showResourceForm && (
            <form className="form-grid" onSubmit={handleCreateResource}>
              <div className="rbac-actions">
                <input
                  value={resourceForm.ciId}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, ciId: event.target.value }))}
                  placeholder="ciId（可留空自动生成）"
                />
                <input
                  required
                  value={resourceForm.type}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, type: event.target.value }))}
                  placeholder="类型（如 Service/K8sCluster）"
                />
                <input
                  required
                  value={resourceForm.name}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="资源名称"
                />
              </div>
              <div className="rbac-actions">
                <input
                  value={resourceForm.cloud}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, cloud: event.target.value }))}
                  placeholder="cloud"
                />
                <input
                  value={resourceForm.region}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, region: event.target.value }))}
                  placeholder="region"
                />
                <input
                  value={resourceForm.env}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, env: event.target.value }))}
                  placeholder="env"
                />
                <input
                  value={resourceForm.owner}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, owner: event.target.value }))}
                  placeholder="owner"
                />
              </div>
              <div className="rbac-actions">
                <input
                  value={resourceForm.lifecycle}
                  onChange={(event) => setResourceForm((prev) => ({ ...prev, lifecycle: event.target.value }))}
                  placeholder="lifecycle"
                />
                <select
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
              </div>
            </form>
          )}

          <form className="rbac-actions" onSubmit={handleResourceFilterSubmit}>
            <input
              value={resourceFilters.keyword}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词（name/ciId）"
            />
            <input
              value={resourceFilters.type}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, type: event.target.value }))}
              placeholder="type"
            />
            <input
              value={resourceFilters.cloud}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, cloud: event.target.value }))}
              placeholder="cloud"
            />
            <input
              value={resourceFilters.region}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, region: event.target.value }))}
              placeholder="region"
            />
            <input
              value={resourceFilters.env}
              onChange={(event) => setResourceFilters((prev) => ({ ...prev, env: event.target.value }))}
              placeholder="env"
            />
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
          </form>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  <th>CIID</th>
                  <th>类型</th>
                  <th>名称</th>
                  <th>基础配置</th>
                  <th>云/地域</th>
                  <th>环境</th>
                  <th>Owner</th>
                  <th>来源</th>
                  <th>过期时间</th>
                  <th>最近发现</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {resourceLoading
                  ? (
                    <tr><td colSpan={11}>加载中...</td></tr>
                  )
                  : resources.length === 0
                    ? (
                      <tr><td colSpan={11}>暂无数据</td></tr>
                    )
                    : resources.map((item) => (
                      <tr key={item.id}>
                        <td>{item.ciId || "-"}</td>
                        <td>{item.type || "-"}</td>
                        <td>{item.name || "-"}</td>
                        <td>{formatResourceBaseSpec(item)}</td>
                        <td>{item.cloud || "-"}/{item.region || "-"}</td>
                        <td>{item.env || "-"}</td>
                        <td>{item.owner || "-"}</td>
                        <td>{item.source || "-"}</td>
                        <td>{formatResourceExpiry(item)}</td>
                        <td>{formatDateTime(item.lastSeenAt)}</td>
                        <td>
                          <div className="rbac-row-actions">
                            <PermissionButton
                              permissionKey="button.cmdb.resource.delete"
                              className="btn cursor-pointer"
                              type="button"
                              disabled={resourceDeletingId === item.id}
                              onClick={() => void handleDeleteResource(item.id)}
                            >
                              {resourceDeletingId === item.id ? "删除中..." : "删除"}
                            </PermissionButton>
                          </div>
                        </td>
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

        <article className="card rbac-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>关系图谱视图</h3>
              <p className="muted">固化 10 类关系，支持按节点和关系类型过滤。</p>
            </div>
            <div className="rbac-actions">
              <button
                className="btn cursor-pointer"
                type="button"
                onClick={() => setShowRelationForm((value) => !value)}
              >
                {showRelationForm ? "收起创建" : "展开创建"}
              </button>
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

          {showRelationForm && (
            <form className="form-grid" onSubmit={handleCreateRelation}>
              <div className="rbac-actions">
                <input
                  list="cmdb-ciid-options"
                  value={relationForm.fromCiId}
                  onChange={(event) => setRelationForm((prev) => ({ ...prev, fromCiId: event.target.value }))}
                  placeholder="fromCiId"
                  required
                />
                <input
                  list="cmdb-ciid-options"
                  value={relationForm.toCiId}
                  onChange={(event) => setRelationForm((prev) => ({ ...prev, toCiId: event.target.value }))}
                  placeholder="toCiId"
                  required
                />
                <select
                  value={relationForm.relationType}
                  onChange={(event) => setRelationForm((prev) => ({ ...prev, relationType: event.target.value }))}
                >
                  {relationTypeOptions.map((type) => (
                    <option key={type} value={type}>{type}</option>
                  ))}
                </select>
              </div>
              <div className="rbac-actions">
                <select
                  value={relationForm.direction}
                  onChange={(event) => setRelationForm((prev) => ({ ...prev, direction: event.target.value }))}
                >
                  <option value="outbound">outbound</option>
                  <option value="inbound">inbound</option>
                  <option value="bidirectional">bidirectional</option>
                </select>
                <select
                  value={relationForm.criticality}
                  onChange={(event) => setRelationForm((prev) => ({ ...prev, criticality: event.target.value }))}
                >
                  <option value="P0">P0</option>
                  <option value="P1">P1</option>
                  <option value="P2">P2</option>
                  <option value="P3">P3</option>
                </select>
                <input
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
              </div>
            </form>
          )}

          <form className="rbac-actions" onSubmit={handleRelationFilterSubmit}>
            <input
              list="cmdb-ciid-options"
              value={relationFilters.fromCiId}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, fromCiId: event.target.value }))}
              placeholder="fromCiId"
            />
            <input
              list="cmdb-ciid-options"
              value={relationFilters.toCiId}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, toCiId: event.target.value }))}
              placeholder="toCiId"
            />
            <select
              value={relationFilters.relationType}
              onChange={(event) => setRelationFilters((prev) => ({ ...prev, relationType: event.target.value }))}
            >
              <option value="">全部关系类型</option>
              {relationTypeOptions.map((type) => (
                <option key={type} value={type}>{type}</option>
              ))}
            </select>
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
          </form>

          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  <th>From</th>
                  <th>To</th>
                  <th>关系</th>
                  <th>方向</th>
                  <th>等级</th>
                  <th>置信度</th>
                  <th>更新时间</th>
                </tr>
              </thead>
              <tbody>
                {relationLoading
                  ? <tr><td colSpan={7}>加载中...</td></tr>
                  : relations.length === 0
                    ? <tr><td colSpan={7}>暂无数据</td></tr>
                    : relations.map((item) => (
                      <tr key={item.id}>
                        <td>{item.fromCiId}</td>
                        <td>{item.toCiId}</td>
                        <td>{item.relationType}</td>
                        <td>{item.direction || "-"}</td>
                        <td>{item.criticality || "-"}</td>
                        <td>{typeof item.confidence === "number" ? item.confidence.toFixed(2) : "-"}</td>
                        <td>{formatDateTime(item.relationUpdatedAt || item.updatedAt)}</td>
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
            <div className="rbac-actions">
              <input
                value={topologyApplication}
                onChange={(event) => setTopologyApplication(event.target.value)}
                placeholder="application（拓扑）"
              />
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
            <div className="rbac-actions">
              <input
                value={impactCiId}
                onChange={(event) => setImpactCiId(event.target.value)}
                placeholder="ciId（故障影响）"
              />
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
            <div className="rbac-actions">
              <input
                value={failoverRegion}
                onChange={(event) => setFailoverRegion(event.target.value)}
                placeholder="region（地域容灾）"
              />
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
            <div className="rbac-actions">
              <input
                value={changeReleaseId}
                onChange={(event) => setChangeReleaseId(event.target.value)}
                placeholder="releaseId（变更影响）"
              />
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

          <div className="rbac-table-wrapper rbac-module-scroll">
            <pre className="cmdb-analysis">
              <strong>{analysisTitle}</strong>
              {"\n"}
              {analysisResult || "暂无结果，请先执行查询。"}
            </pre>
          </div>
        </article>
      </div>

      <datalist id="cmdb-ciid-options">
        {ciIdOptions.map((ciId) => (
          <option key={ciId} value={ciId} />
        ))}
      </datalist>
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
