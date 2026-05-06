import { FormEvent, useEffect, useMemo, useState } from "react";

import type { PageData } from "@/api/types";
import {
  checkMiddlewareInstance,
  collectMiddlewareMetrics,
  createMiddlewareInstance,
  deleteMiddlewareInstance,
  getMiddlewareAIOpsProtocol,
  listMiddlewareInstances,
  listMiddlewareMetrics,
  listMiddlewareOperations,
  runMiddlewareAction,
  updateMiddlewareInstance,
} from "@/api/middleware";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { PermissionButton } from "@/components/PermissionButton";
import type { RowActionItem } from "@/components/RowActionOverflow";
import { RowActionOverflow } from "@/components/RowActionOverflow";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type {
  MiddlewareAIOpsProtocol,
  MiddlewareActionPayload,
  MiddlewareInstanceItem,
  MiddlewareMetricItem,
  MiddlewareOperationItem,
  MiddlewareType,
} from "@/types/middleware";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const middlewareSettingsKey = "middleware.instances.table.settings";
const middlewareTypes: MiddlewareType[] = ["redis", "postgresql", "rabbitmq"];
const confirmText = "确认删除资源";

const instanceColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "name", label: "名称" },
  { key: "type", label: "类型" },
  { key: "endpoint", label: "Endpoint" },
  { key: "env", label: "环境" },
  { key: "owner", label: "负责人" },
  { key: "status", label: "状态" },
  { key: "version", label: "版本" },
  { key: "lastCheckedAt", label: "最近检查" },
  { key: "actions", label: "操作", required: true },
];
const defaultVisibleColumns = ["id", "name", "type", "endpoint", "env", "owner", "status", "lastCheckedAt", "actions"];

const middlewareTemplates: Record<MiddlewareType, { endpoint: string; labels: Record<string, unknown>; metadata: Record<string, unknown> }> = {
  redis: {
    endpoint: "mock://redis",
    labels: { app: "core", tier: "cache" },
    metadata: { deployMode: "standalone", db: 0, maxMemoryPolicy: "allkeys-lru" },
  },
  postgresql: {
    endpoint: "mock://postgresql",
    labels: { app: "core", tier: "database" },
    metadata: { database: "postgres", role: "primary", pool: "default" },
  },
  rabbitmq: {
    endpoint: "mock://rabbitmq",
    labels: { app: "core", tier: "message" },
    metadata: { vhost: "/", queueLimit: 1000, durable: true },
  },
};

interface InstanceFilter {
  keyword: string;
  type: string;
  env: string;
  status: string;
}

interface InstanceForm {
  name: string;
  type: MiddlewareType;
  endpoint: string;
  env: string;
  owner: string;
  authType: string;
  tlsEnable: boolean;
  username: string;
  password: string;
  token: string;
  labelsJSON: string;
  metadataJSON: string;
}

type DrawerState = "closed" | "create" | "edit" | "detail";
type ConfirmTarget =
  | { type: "delete"; item: MiddlewareInstanceItem }
  | { type: "action"; title: string; description: string; payload: MiddlewareActionPayload };

function defaultFilter(): InstanceFilter {
  return { keyword: "", type: "", env: "", status: "" };
}

function defaultForm(type: MiddlewareType = "redis"): InstanceForm {
  const template = middlewareTemplates[type];
  return {
    name: "",
    type,
    endpoint: template.endpoint,
    env: "prod",
    owner: "",
    authType: "password",
    tlsEnable: false,
    username: "",
    password: "",
    token: "",
    labelsJSON: prettyJSON(template.labels),
    metadataJSON: prettyJSON(template.metadata),
  };
}

export function MiddlewarePage() {
  const [instances, setInstances] = useState<PageData<MiddlewareInstanceItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [filter, setFilter] = useState(defaultFilter);
  const [query, setQuery] = useState(defaultFilter);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [checkingId, setCheckingId] = useState<number | null>(null);
  const [collectingId, setCollectingId] = useState<number | null>(null);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [metrics, setMetrics] = useState<PageData<MiddlewareMetricItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [operations, setOperations] = useState<MiddlewareOperationItem[]>([]);
  const [protocol, setProtocol] = useState<MiddlewareAIOpsProtocol | null>(null);
  const [aiopsResult, setAIOpsResult] = useState<Record<string, unknown> | null>(null);
  const [drawer, setDrawer] = useState<DrawerState>("closed");
  const [editId, setEditId] = useState<number | null>(null);
  const [form, setForm] = useState(defaultForm);
  const [confirmTarget, setConfirmTarget] = useState<ConfirmTarget | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [visibleColumns, setVisibleColumns] = useState(() => {
    const persisted = loadPersistedListSettings(middlewareSettingsKey);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaultVisibleColumns, instanceColumns);
  });

  const totalPagesValue = useMemo(() => totalPages(instances.total, pageSize), [instances.total, pageSize]);
  const visibleOrderedColumns = useMemo(
    () => visibleColumns
      .map((key) => instanceColumns.find((column) => column.key === key))
      .filter((column): column is TableSettingsColumn => Boolean(column)),
    [visibleColumns],
  );
  const selectedInstance = useMemo(() => instances.list.find((item) => item.id === selectedId) ?? null, [instances.list, selectedId]);

  useEffect(() => {
    void loadInstances();
  }, [page, pageSize, query]);

  useEffect(() => {
    if (selectedId) {
      void loadMetrics(selectedId);
      void loadOperations(selectedId);
    }
  }, [selectedId]);

  useEffect(() => {
    void loadProtocol();
  }, []);

  useEffect(() => {
    savePersistedListSettings(middlewareSettingsKey, { visibleColumnKeys: visibleColumns });
  }, [visibleColumns]);

  async function loadInstances() {
    setLoading(true);
    try {
      const result = await listMiddlewareInstances({
        page,
        pageSize,
        keyword: query.keyword || undefined,
        type: query.type || undefined,
        env: query.env || undefined,
        status: query.status || undefined,
      });
      setInstances(result);
      if (!selectedId && result.list.length > 0) {
        setSelectedId(result.list[0].id);
      }
      if (selectedId && !result.list.some((item) => item.id === selectedId)) {
        setSelectedId(result.list[0]?.id ?? null);
      }
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载中间件实例失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadMetrics(instanceId: number) {
    try {
      const result = await listMiddlewareMetrics(instanceId, { page: 1, pageSize: 10 });
      setMetrics(result);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载指标失败");
    }
  }

  async function loadOperations(instanceId?: number) {
    try {
      const result = await listMiddlewareOperations({ page: 1, pageSize: 8, instanceId });
      setOperations(result.list);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载操作记录失败");
    }
  }

  async function loadProtocol() {
    try {
      const result = await getMiddlewareAIOpsProtocol();
      setProtocol(result);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载AIOps协议失败");
    }
  }

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setQuery(filter);
  }

  function resetSearch() {
    const next = defaultFilter();
    setFilter(next);
    setQuery(next);
    setPage(1);
  }

  function openCreateDrawer() {
    setEditId(null);
    setForm(defaultForm());
    setDrawer("create");
  }

  function openEditDrawer(item: MiddlewareInstanceItem) {
    setEditId(item.id);
    setForm({
      name: item.name,
      type: normalizeType(item.type),
      endpoint: item.endpoint,
      env: item.env ?? "prod",
      owner: item.owner ?? "",
      authType: item.authType ?? "password",
      tlsEnable: Boolean(item.tlsEnable),
      username: "",
      password: "",
      token: "",
      labelsJSON: prettyJSON(item.labels ?? {}),
      metadataJSON: prettyJSON(item.metadata ?? {}),
    });
    setDrawer("edit");
  }

  function openDetailDrawer(item: MiddlewareInstanceItem) {
    setSelectedId(item.id);
    setDrawer("detail");
  }

  function applyTemplate(type: MiddlewareType) {
    const template = middlewareTemplates[type];
    setForm((current) => ({
      ...current,
      type,
      endpoint: template.endpoint,
      labelsJSON: prettyJSON(template.labels),
      metadataJSON: prettyJSON(template.metadata),
    }));
  }

  async function submitInstance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    try {
      const labels = parseJSONObject(form.labelsJSON, "labels JSON");
      const metadata = parseJSONObject(form.metadataJSON, "metadata JSON");
      const payload = {
        name: form.name,
        type: form.type,
        endpoint: form.endpoint,
        env: form.env,
        owner: form.owner,
        authType: form.authType,
        tlsEnable: form.tlsEnable,
        username: form.username || undefined,
        password: form.password || undefined,
        token: form.token || undefined,
        labels,
        metadata,
      };
      if (drawer === "edit" && editId) {
        await updateMiddlewareInstance(editId, payload);
        showToast("中间件实例已更新");
      } else {
        const created = await createMiddlewareInstance(payload);
        setSelectedId(created.id);
        showToast("中间件实例已创建");
      }
      setDrawer("closed");
      await loadInstances();
    } catch (error) {
      showToast(error instanceof Error ? error.message : "保存中间件实例失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleCheck(item: MiddlewareInstanceItem) {
    setCheckingId(item.id);
    try {
      await checkMiddlewareInstance(item.id);
      showToast("健康检查已完成");
      await loadInstances();
    } catch (error) {
      showToast(error instanceof Error ? error.message : "健康检查失败");
    } finally {
      setCheckingId(null);
    }
  }

  async function handleCollect(item: MiddlewareInstanceItem) {
    setCollectingId(item.id);
    try {
      await collectMiddlewareMetrics(item.id);
      setSelectedId(item.id);
      await loadMetrics(item.id);
      showToast("指标采集已完成");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "指标采集失败");
    } finally {
      setCollectingId(null);
    }
  }

  async function runAction(payload: MiddlewareActionPayload, successMessage: string) {
    try {
      const result = await runMiddlewareAction(payload);
      setAIOpsResult(result as unknown as Record<string, unknown>);
      await loadOperations(payload.instanceId);
      showToast(successMessage);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "中间件动作执行失败");
    }
  }

  function triggerAction(item: MiddlewareInstanceItem, action: string, highRisk = false) {
    const payload: MiddlewareActionPayload = { instanceId: item.id, type: item.type, action, dryRun: false };
    if (highRisk) {
      setConfirmTarget({
        type: "action",
        title: "高危动作确认",
        description: `${item.name} 将执行 ${action}，此动作可能不可恢复。`,
        payload,
      });
      return;
    }
    void runAction(payload, "中间件动作已执行");
  }

  function triggerDryRun(item: MiddlewareInstanceItem, action: string) {
    void runAction({ instanceId: item.id, type: item.type, action, dryRun: true }, "dry-run 已生成");
  }

  async function confirmDanger() {
    if (!confirmTarget) return;
    setConfirming(true);
    try {
      if (confirmTarget.type === "delete") {
        await deleteMiddlewareInstance(confirmTarget.item.id, confirmText);
        showToast("中间件实例已删除");
        if (selectedId === confirmTarget.item.id) setSelectedId(null);
        await loadInstances();
      } else {
        await runAction({ ...confirmTarget.payload, confirmationText: confirmText }, "高危动作已提交");
      }
      setConfirmTarget(null);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "确认操作失败");
    } finally {
      setConfirming(false);
    }
  }

  function rowActions(item: MiddlewareInstanceItem) {
    const actions = [
      { key: "detail", label: "详情", onClick: () => openDetailDrawer(item) },
      { key: "edit", label: "编辑", onClick: () => openEditDrawer(item), permissionKey: "button.middleware.instance.update" },
      {
        key: "check",
        label: checkingId === item.id ? "检查中" : "检查",
        onClick: () => void handleCheck(item),
        disabled: checkingId === item.id,
        permissionKey: "button.middleware.instance.check",
      },
      {
        key: "metrics",
        label: collectingId === item.id ? "采集中" : "采集指标",
        onClick: () => void handleCollect(item),
        disabled: collectingId === item.id,
        permissionKey: "button.middleware.metrics.collect",
      },
      ...middlewareActionItems(item),
      {
        key: "delete",
        label: "删除",
        onClick: () => setConfirmTarget({ type: "delete", item }),
        disabled: String(item.status ?? "").toLowerCase() === "healthy",
        permissionKey: "button.middleware.instance.delete",
        className: "btn ghost cursor-pointer",
      },
    ];
    return actions;
  }

  function middlewareActionItems(item: MiddlewareInstanceItem) {
    if (item.type === "redis") {
      return [
        { key: "info", label: "Info", onClick: () => triggerAction(item, "info"), permissionKey: "button.middleware.action.run" },
        { key: "dbsize", label: "DBSize", onClick: () => triggerAction(item, "dbsize"), permissionKey: "button.middleware.action.run" },
        { key: "flushdb-dry", label: "Flush DryRun", onClick: () => triggerDryRun(item, "flushdb"), permissionKey: "button.middleware.action.run" },
        { key: "flushdb", label: "FlushDB", onClick: () => triggerAction(item, "flushdb", true), permissionKey: "button.middleware.action.run" },
      ];
    }
    if (item.type === "postgresql") {
      return [
        { key: "version", label: "Version", onClick: () => triggerAction(item, "version"), permissionKey: "button.middleware.action.run" },
        { key: "activity", label: "Activity", onClick: () => triggerAction(item, "activity"), permissionKey: "button.middleware.action.run" },
      ];
    }
    if (item.type === "rabbitmq") {
      return [
        { key: "overview", label: "Overview", onClick: () => triggerAction(item, "overview"), permissionKey: "button.middleware.action.run" },
      ];
    }
    return [];
  }

  function toggleColumn(key: string) {
    const column = instanceColumns.find((item) => item.key === key);
    if (column?.required) return;
    setVisibleColumns((current) => (current.includes(key) ? current.filter((item) => item !== key) : [...current, key]));
  }

  function moveColumn(key: string, direction: "up" | "down") {
    setVisibleColumns((current) => moveColumnKey(current, key, direction));
  }

  return (
    <div className="page">
      <section className="rbac-module-grid docker-module-grid">
        <div className="card rbac-module-card rbac-compact-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>中间件实例</h3>
              <p className="muted">Redis / PostgreSQL / RabbitMQ 纳管、探活、指标和 AIOps 动作协议。</p>
            </div>
            <div className="rbac-actions">
              <PermissionButton permissionKey="button.middleware.instance.create" className="btn primary cursor-pointer" type="button" onClick={openCreateDrawer}>
                创建实例
              </PermissionButton>
            </div>
          </header>

          <form className="cloud-filter-bar" onSubmit={submitSearch}>
            <input className="cloud-filter-control cloud-filter-keyword" placeholder="搜索名称 / Endpoint / 负责人" value={filter.keyword} onChange={(event) => setFilter((current) => ({ ...current, keyword: event.target.value }))} />
            <select className="cloud-filter-control" value={filter.type} onChange={(event) => setFilter((current) => ({ ...current, type: event.target.value }))}>
              <option value="">全部类型</option>
              {middlewareTypes.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
            <select className="cloud-filter-control" value={filter.env} onChange={(event) => setFilter((current) => ({ ...current, env: event.target.value }))}>
              <option value="">全部环境</option>
              <option value="prod">prod</option>
              <option value="staging">staging</option>
              <option value="dev">dev</option>
            </select>
            <select className="cloud-filter-control" value={filter.status} onChange={(event) => setFilter((current) => ({ ...current, status: event.target.value }))}>
              <option value="">全部状态</option>
              <option value="healthy">healthy</option>
              <option value="error">error</option>
              <option value="unknown">unknown</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn primary cursor-pointer" type="submit">搜索</button>
              <button className="btn ghost cursor-pointer" type="button" onClick={resetSearch}>重置</button>
            </div>
          </form>

          <div className="rbac-table-wrapper">
            <table className="rbac-table">
              <thead>
                <tr>
                  {visibleOrderedColumns.map((column) => (
                    <th key={column.key}>
                      {column.key === "actions" ? (
                        <div className="table-actions-header">
                          <span>{column.label}</span>
                          <button
                            className="table-settings-trigger cursor-pointer"
                            type="button"
                            onClick={() => setSettingsOpen(true)}
                            aria-label="中间件实例列表字段设置"
                          >
                            ⚙️
                          </button>
                        </div>
                      ) : column.label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={visibleColumns.length}>加载中...</td></tr>
                ) : instances.list.length === 0 ? (
                  <tr><td colSpan={visibleColumns.length}>暂无中间件实例</td></tr>
                ) : instances.list.map((item) => (
                  <tr key={item.id} className={selectedId === item.id ? "docker-selected-row" : undefined} onClick={() => setSelectedId(item.id)}>
                    {visibleOrderedColumns.map((column) => (
                      <td key={column.key} onClick={column.key === "actions" ? (event) => event.stopPropagation() : undefined}>
                        {renderMiddlewareCell(column.key, item, rowActions(item))}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <Pagination page={page} pageSize={pageSize} total={instances.total} totalPages={totalPagesValue} onPageChange={setPage} onPageSizeChange={(next) => { setPageSize(next); setPage(1); }} />
        </div>
      </section>

      <section className="grid cards">
        <div className="card docker-aiops-card">
          <h3>最近指标</h3>
          <p className="muted">当前实例：{selectedInstance ? selectedInstance.name : "未选择"}</p>
          <div className="docker-operation-list">
            {metrics.list.length === 0 ? <span className="muted">暂无指标</span> : metrics.list.slice(0, 6).map((item) => (
              <div className="docker-operation-item" key={item.id}>
                <strong>{item.metricType}</strong>
                <span>{item.value}{item.unit ? ` ${item.unit}` : ""}</span>
                <span className="muted">{formatTime(item.collectedAt)}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card docker-aiops-card">
          <h3>AIOps 协议</h3>
          <p className="muted">{protocol?.protocolVersion || "未加载"}</p>
          <div className="docker-aiops-protocol">
            {protocol?.resources.map((resource) => (
              <div key={resource.type}>
                <strong>{resource.type}</strong>
                <span>{resource.actions.map((action) => action.name).join(" / ")}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card docker-aiops-card">
          <h3>操作记录</h3>
          <div className="docker-operation-list">
            {operations.length === 0 ? <span className="muted">暂无操作记录</span> : operations.map((item) => (
              <div className="docker-operation-item" key={item.id}>
                <strong>{item.action}</strong>
                <StatusBadge status={item.status} />
                <span className="muted">{item.traceId}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {aiopsResult ? (
        <section className="card docker-aiops-card">
          <header className="rbac-module-header">
            <div>
              <h3>最近动作结果</h3>
              <p className="muted">dry-run 与真实执行返回均保留机器可读结构，后续可直接接 AIOpsChat。</p>
            </div>
            <button className="btn ghost cursor-pointer" type="button" onClick={() => setAIOpsResult(null)}>清空</button>
          </header>
          <pre className="docker-aiops-result">{prettyJSON(aiopsResult)}</pre>
        </section>
      ) : null}

      <InstanceDrawer drawer={drawer} form={form} selectedInstance={selectedInstance} submitting={submitting} onClose={() => setDrawer("closed")} onSubmit={submitInstance} onFormChange={setForm} onTemplate={applyTemplate} />

      <DeleteConfirmModal
        open={confirmTarget !== null}
        title={confirmTarget?.type === "action" ? confirmTarget.title : "删除中间件实例"}
        description={confirmTarget?.type === "action" ? confirmTarget.description : "删除实例会同时删除关联凭据，请确认该实例不在健康运行状态。"}
        confirming={confirming}
        onCancel={() => setConfirmTarget(null)}
        onConfirm={() => void confirmDanger()}
      />

      <TableSettingsModal
        open={settingsOpen}
        title="中间件实例字段"
        columns={instanceColumns}
        visibleColumnKeys={visibleColumns}
        onToggleColumn={toggleColumn}
        onMoveColumn={moveColumn}
        onReset={() => setVisibleColumns(defaultVisibleColumns)}
        onClose={() => setSettingsOpen(false)}
      />
    </div>
  );
}

interface InstanceDrawerProps {
  drawer: DrawerState;
  form: InstanceForm;
  selectedInstance: MiddlewareInstanceItem | null;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onFormChange: (next: InstanceForm | ((current: InstanceForm) => InstanceForm)) => void;
  onTemplate: (type: MiddlewareType) => void;
}

function InstanceDrawer({ drawer, form, selectedInstance, submitting, onClose, onSubmit, onFormChange, onTemplate }: InstanceDrawerProps) {
  if (drawer === "closed") return null;
  const detailMode = drawer === "detail";
  return (
    <div className="rbac-drawer-mask" onClick={onClose}>
      <aside className="rbac-drawer" onClick={(event) => event.stopPropagation()}>
        <header className="rbac-drawer-header">
          <h3>{drawer === "create" ? "创建中间件实例" : drawer === "edit" ? "编辑中间件实例" : "中间件详情"}</h3>
          <button className="btn ghost cursor-pointer" type="button" onClick={onClose}>关闭</button>
        </header>
        <div className="rbac-drawer-body">
          {detailMode ? (
            <pre className="docker-aiops-result">{prettyJSON(selectedInstance ?? {})}</pre>
          ) : (
            <form className="middleware-form" onSubmit={onSubmit}>
              <section className="middleware-form-section">
                <div className="middleware-form-section-title">
                  <h4>基础信息</h4>
                  <p className="muted">定义实例身份、类型和归属信息。</p>
                </div>
                <div className="middleware-form-grid">
                  <label className="middleware-form-field middleware-form-field-wide">
                    <span>名称</span>
                    <input value={form.name} required placeholder="redis-prod-cache" onChange={(event) => onFormChange((current) => ({ ...current, name: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>类型</span>
                    <select value={form.type} onChange={(event) => onTemplate(normalizeType(event.target.value))}>
                      {middlewareTypes.map((item) => <option key={item} value={item}>{item}</option>)}
                    </select>
                  </label>
                  <label className="middleware-form-field">
                    <span>环境</span>
                    <input value={form.env} placeholder="prod / staging / dev" onChange={(event) => onFormChange((current) => ({ ...current, env: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>负责人</span>
                    <input value={form.owner} placeholder="SRE / Team" onChange={(event) => onFormChange((current) => ({ ...current, owner: event.target.value }))} />
                  </label>
                </div>
              </section>

              <section className="middleware-form-section">
                <div className="middleware-form-section-title">
                  <h4>连接与认证</h4>
                  <p className="muted">Endpoint 支持真实连接，也可使用 mock:// 前缀做开发联调。</p>
                </div>
                <div className="middleware-form-grid">
                  <label className="middleware-form-field middleware-form-field-wide">
                    <span>Endpoint</span>
                    <input value={form.endpoint} required placeholder="redis://10.0.0.10:6379 或 mock://redis" onChange={(event) => onFormChange((current) => ({ ...current, endpoint: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>认证方式</span>
                    <input value={form.authType} placeholder="password / token" onChange={(event) => onFormChange((current) => ({ ...current, authType: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>用户名</span>
                    <input value={form.username} autoComplete="off" placeholder="可选" onChange={(event) => onFormChange((current) => ({ ...current, username: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>密码</span>
                    <input type="password" value={form.password} autoComplete="new-password" placeholder="编辑时留空则不更新" onChange={(event) => onFormChange((current) => ({ ...current, password: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field">
                    <span>Token</span>
                    <input type="password" value={form.token} autoComplete="new-password" placeholder="可选" onChange={(event) => onFormChange((current) => ({ ...current, token: event.target.value }))} />
                  </label>
                  <label className="middleware-tls-card cursor-pointer">
                    <input type="checkbox" checked={form.tlsEnable} onChange={(event) => onFormChange((current) => ({ ...current, tlsEnable: event.target.checked }))} />
                    <span>
                      <strong>启用 TLS</strong>
                      <small>生产环境建议启用加密连接。</small>
                    </span>
                  </label>
                </div>
              </section>

              <section className="middleware-form-section">
                <div className="middleware-form-section-title">
                  <h4>高级配置</h4>
                  <p className="muted">用于标签筛选、AIOps 协议参数和后续扩展。</p>
                </div>
                <div className="middleware-form-grid">
                  <label className="middleware-form-field middleware-form-field-wide">
                    <span>Labels JSON</span>
                    <textarea className="middleware-json-editor" value={form.labelsJSON} onChange={(event) => onFormChange((current) => ({ ...current, labelsJSON: event.target.value }))} />
                  </label>
                  <label className="middleware-form-field middleware-form-field-wide">
                    <span>Metadata JSON</span>
                    <textarea className="middleware-json-editor" value={form.metadataJSON} onChange={(event) => onFormChange((current) => ({ ...current, metadataJSON: event.target.value }))} />
                  </label>
                </div>
              </section>

              <div className="middleware-form-actions">
                <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>{submitting ? "保存中..." : "保存"}</button>
                <button className="btn ghost cursor-pointer" type="button" onClick={onClose}>取消</button>
              </div>
            </form>
          )}
        </div>
      </aside>
    </div>
  );
}

function Pagination({ page, pageSize, total, totalPages: pageTotal, onPageChange, onPageSizeChange }: { page: number; pageSize: number; total: number; totalPages: number; onPageChange: (page: number) => void; onPageSizeChange: (pageSize: number) => void }) {
  return (
    <div className="rbac-pagination">
      <div className="rbac-pagination-group"><span className="muted">共 {total} 条</span></div>
      <div className="rbac-pagination-group">
        <select className="rbac-pagination-select" value={pageSize} onChange={(event) => onPageSizeChange(Number(event.target.value))}>
          {pageSizeOptions.map((item) => <option key={item} value={item}>{item}条</option>)}
        </select>
        <button className="btn ghost cursor-pointer" type="button" disabled={page <= 1} onClick={() => onPageChange(Math.max(1, page - 1))}>上一页</button>
        <span className="rbac-pagination-text">{page} / {pageTotal}</span>
        <button className="btn ghost cursor-pointer" type="button" disabled={page >= pageTotal} onClick={() => onPageChange(Math.min(pageTotal, page + 1))}>下一页</button>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status?: string }) {
  const normalized = String(status || "unknown").toLowerCase();
  const className = normalized === "healthy" || normalized === "success" ? "docker-status docker-status-connected" : normalized === "error" || normalized === "failed" ? "docker-status docker-status-error" : "docker-status";
  return <span className={className}>{status || "unknown"}</span>;
}

function renderMiddlewareCell(columnKey: string, item: MiddlewareInstanceItem, actions: RowActionItem[]) {
  switch (columnKey) {
    case "id":
      return item.id;
    case "name":
      return item.name;
    case "type":
      return item.type;
    case "endpoint":
      return item.endpoint;
    case "env":
      return item.env || "-";
    case "owner":
      return item.owner || "-";
    case "status":
      return <StatusBadge status={item.status} />;
    case "version":
      return truncate(item.version || "-", 36);
    case "lastCheckedAt":
      return formatTime(item.lastCheckedAt);
    case "actions":
      return <div className="rbac-row-actions"><RowActionOverflow actions={actions} /></div>;
    default:
      return "-";
  }
}

function normalizeType(value: string): MiddlewareType {
  if (value === "postgres" || value === "pg") return "postgresql";
  if (value === "rabbit" || value === "amqp") return "rabbitmq";
  if (value === "postgresql" || value === "rabbitmq" || value === "redis") return value;
  return "redis";
}

function parseJSONObject(value: string, label: string): Record<string, unknown> {
  const parsed = JSON.parse(value || "{}");
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`${label} 必须是 JSON object`);
  }
  return parsed as Record<string, unknown>;
}

function prettyJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2);
}

function totalPages(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

function formatTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function truncate(value: string, limit: number): string {
  if (value.length <= limit) return value;
  return `${value.slice(0, limit)}...`;
}

function moveColumnKey(columns: string[], key: string, direction: "up" | "down"): string[] {
  const index = columns.indexOf(key);
  if (index < 0) return columns;
  const targetIndex = direction === "up" ? index - 1 : index + 1;
  if (targetIndex < 0 || targetIndex >= columns.length) return columns;
  const next = [...columns];
  [next[index], next[targetIndex]] = [next[targetIndex], next[index]];
  return next;
}
