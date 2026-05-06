import { FormEvent, useEffect, useMemo, useState } from "react";

import type { PageData } from "@/api/types";
import {
  checkDockerHost,
  createComposeStack,
  createDockerHost,
  deleteComposeStack,
  deleteDockerHost,
  getDockerAIOpsProtocol,
  listComposeStacks,
  listDockerHostResources,
  listDockerHosts,
  listDockerOperations,
  runDockerAction,
  updateComposeStack,
  updateDockerHost,
} from "@/api/docker";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { PermissionButton } from "@/components/PermissionButton";
import { RowActionOverflow } from "@/components/RowActionOverflow";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type {
  DockerAIOpsProtocol,
  DockerActionPayload,
  DockerComposeStackItem,
  DockerHostItem,
  DockerOperationItem,
  DockerResourceItem,
  DockerResourceType,
} from "@/types/docker";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const dockerHostSettingsKey = "docker.hosts.table.settings";
const dockerResourceSettingsKey = "docker.resources.table.settings";
const dockerStackSettingsKey = "docker.stacks.table.settings";
const resourceTypes: DockerResourceType[] = ["container", "image", "network", "volume"];
const hostColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "name", label: "名称" },
  { key: "endpoint", label: "Endpoint" },
  { key: "env", label: "环境" },
  { key: "owner", label: "负责人" },
  { key: "status", label: "状态" },
  { key: "version", label: "版本" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];
const resourceColumns: TableSettingsColumn[] = [
  { key: "id", label: "资源ID" },
  { key: "type", label: "类型" },
  { key: "name", label: "名称" },
  { key: "status", label: "状态" },
  { key: "image", label: "镜像" },
  { key: "driver", label: "驱动" },
  { key: "actions", label: "操作", required: true },
];
const stackColumns: TableSettingsColumn[] = [
  { key: "id", label: "ID" },
  { key: "hostId", label: "主机" },
  { key: "name", label: "名称" },
  { key: "status", label: "状态" },
  { key: "services", label: "服务数" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];
const defaultHostColumns = ["id", "name", "endpoint", "env", "owner", "status", "version", "actions"];
const defaultResourceColumns = ["id", "type", "name", "status", "image", "driver", "actions"];
const defaultStackColumns = ["id", "hostId", "name", "status", "services", "actions"];

type DrawerState = "closed" | "host-create" | "host-edit" | "stack-create" | "stack-edit";
type SettingsTarget = "closed" | "hosts" | "resources" | "stacks";

interface PendingDockerAction {
  title: string;
  description: string;
  payload: DockerActionPayload;
  refreshTarget: "resources" | "stacks";
  actionKey: string;
}

interface HostFilter {
  keyword: string;
  env: string;
  status: string;
}

interface HostForm {
  name: string;
  endpoint: string;
  tlsEnable: boolean;
  env: string;
  owner: string;
  labelsJSON: string;
  metadataJSON: string;
}

interface ResourceFilter {
  type: DockerResourceType;
  keyword: string;
}

interface StackForm {
  hostId: string;
  name: string;
  status: string;
  services: string;
  content: string;
}

function defaultHostFilter(): HostFilter {
  return { keyword: "", env: "", status: "" };
}

function defaultHostForm(): HostForm {
  return {
    name: "",
    endpoint: "unix:///var/run/docker.sock",
    tlsEnable: false,
    env: "prod",
    owner: "",
    labelsJSON: "{}",
    metadataJSON: "{}",
  };
}

function defaultResourceFilter(): ResourceFilter {
  return { type: "container", keyword: "" };
}

function defaultStackForm(): StackForm {
  return {
    hostId: "",
    name: "",
    status: "draft",
    services: "0",
    content: "services:\n  app:\n    image: nginx:latest\n",
  };
}

export function DockerPage() {
  const [hosts, setHosts] = useState<PageData<DockerHostItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [hostPage, setHostPage] = useState(1);
  const [hostPageSize, setHostPageSize] = useState(defaultPageSize);
  const [hostFilter, setHostFilter] = useState(defaultHostFilter);
  const [hostQuery, setHostQuery] = useState(defaultHostFilter);
  const [hostLoading, setHostLoading] = useState(false);
  const [hostSubmitting, setHostSubmitting] = useState(false);
  const [checkingHostId, setCheckingHostId] = useState<number | null>(null);
  const [selectedHostId, setSelectedHostId] = useState<number | null>(null);
  const [hostEditId, setHostEditId] = useState<number | null>(null);
  const [deleteHostTarget, setDeleteHostTarget] = useState<DockerHostItem | null>(null);
  const [hostForm, setHostForm] = useState(defaultHostForm);

  const [resources, setResources] = useState<PageData<DockerResourceItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [resourcePage, setResourcePage] = useState(1);
  const [resourcePageSize, setResourcePageSize] = useState(defaultPageSize);
  const [resourceFilter, setResourceFilter] = useState(defaultResourceFilter);
  const [resourceQuery, setResourceQuery] = useState(defaultResourceFilter);
  const [resourceLoading, setResourceLoading] = useState(false);

  const [stacks, setStacks] = useState<PageData<DockerComposeStackItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [stackPage, setStackPage] = useState(1);
  const [stackPageSize, setStackPageSize] = useState(defaultPageSize);
  const [stackLoading, setStackLoading] = useState(false);
  const [stackSubmitting, setStackSubmitting] = useState(false);
  const [stackEditId, setStackEditId] = useState<number | null>(null);
  const [deleteStackTarget, setDeleteStackTarget] = useState<DockerComposeStackItem | null>(null);
  const [stackForm, setStackForm] = useState(defaultStackForm);

  const [operations, setOperations] = useState<DockerOperationItem[]>([]);
  const [protocol, setProtocol] = useState<DockerAIOpsProtocol | null>(null);
  const [aiopsResult, setAIOpsResult] = useState<Record<string, unknown> | null>(null);
  const [runningActionKey, setRunningActionKey] = useState<string | null>(null);
  const [confirmActionTarget, setConfirmActionTarget] = useState<PendingDockerAction | null>(null);

  const [drawer, setDrawer] = useState<DrawerState>("closed");
  const [settingsTarget, setSettingsTarget] = useState<SettingsTarget>("closed");
  const [visibleHostColumns, setVisibleHostColumns] = useState(() => {
    const persisted = loadPersistedListSettings(dockerHostSettingsKey);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaultHostColumns, hostColumns);
  });
  const [visibleResourceColumns, setVisibleResourceColumns] = useState(() => {
    const persisted = loadPersistedListSettings(dockerResourceSettingsKey);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaultResourceColumns, resourceColumns);
  });
  const [visibleStackColumns, setVisibleStackColumns] = useState(() => {
    const persisted = loadPersistedListSettings(dockerStackSettingsKey);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaultStackColumns, stackColumns);
  });

  const selectedHost = useMemo(() => hosts.list.find((item) => item.id === selectedHostId) ?? null, [hosts.list, selectedHostId]);
  const hostTotalPages = useMemo(() => totalPages(hosts.total, hostPageSize), [hosts.total, hostPageSize]);
  const resourceTotalPages = useMemo(() => totalPages(resources.total, resourcePageSize), [resources.total, resourcePageSize]);
  const stackTotalPages = useMemo(() => totalPages(stacks.total, stackPageSize), [stacks.total, stackPageSize]);
  const visibleHostColumnDefs = useMemo(
    () => visibleHostColumns
      .map((key) => hostColumns.find((column) => column.key === key))
      .filter((column): column is TableSettingsColumn => Boolean(column)),
    [visibleHostColumns],
  );
  const visibleResourceColumnDefs = useMemo(
    () => visibleResourceColumns
      .map((key) => resourceColumns.find((column) => column.key === key))
      .filter((column): column is TableSettingsColumn => Boolean(column)),
    [visibleResourceColumns],
  );
  const visibleStackColumnDefs = useMemo(
    () => visibleStackColumns
      .map((key) => stackColumns.find((column) => column.key === key))
      .filter((column): column is TableSettingsColumn => Boolean(column)),
    [visibleStackColumns],
  );

  useEffect(() => {
    void loadHosts();
  }, [hostPage, hostPageSize, hostQuery]);

  useEffect(() => {
    if (selectedHostId) void loadResources();
  }, [selectedHostId, resourcePage, resourcePageSize, resourceQuery]);

  useEffect(() => {
    void loadStacks();
  }, [selectedHostId, stackPage, stackPageSize]);

  useEffect(() => {
    void loadProtocolAndOperations();
  }, [selectedHostId]);

  useEffect(() => savePersistedListSettings(dockerHostSettingsKey, { visibleColumnKeys: visibleHostColumns }), [visibleHostColumns]);
  useEffect(() => savePersistedListSettings(dockerResourceSettingsKey, { visibleColumnKeys: visibleResourceColumns }), [visibleResourceColumns]);
  useEffect(() => savePersistedListSettings(dockerStackSettingsKey, { visibleColumnKeys: visibleStackColumns }), [visibleStackColumns]);

  async function loadHosts() {
    setHostLoading(true);
    try {
      const result = await listDockerHosts({
        page: hostPage,
        pageSize: hostPageSize,
        keyword: hostQuery.keyword || undefined,
        env: hostQuery.env || undefined,
        status: hostQuery.status || undefined,
      });
      setHosts(result);
      if (!selectedHostId && result.list.length > 0) {
        setSelectedHostId(result.list[0].id);
      }
    } catch {
      showToast("Docker 主机加载失败");
    } finally {
      setHostLoading(false);
    }
  }

  async function loadResources() {
    if (!selectedHostId) return;
    setResourceLoading(true);
    try {
      const result = await listDockerHostResources(selectedHostId, {
        page: resourcePage,
        pageSize: resourcePageSize,
        type: resourceQuery.type,
        keyword: resourceQuery.keyword || undefined,
      });
      setResources(result);
    } catch {
      setResources({ list: [], total: 0, page: resourcePage, pageSize: resourcePageSize });
      showToast("Docker 资源加载失败，请先校验主机连通性");
    } finally {
      setResourceLoading(false);
    }
  }

  async function loadStacks() {
    setStackLoading(true);
    try {
      const result = await listComposeStacks({
        page: stackPage,
        pageSize: stackPageSize,
        hostId: selectedHostId ?? undefined,
      });
      setStacks(result);
    } catch {
      showToast("Compose Stack 加载失败");
    } finally {
      setStackLoading(false);
    }
  }

  async function loadProtocolAndOperations() {
    try {
      const [protocolData, operationPage] = await Promise.all([
        getDockerAIOpsProtocol(),
        listDockerOperations({ page: 1, pageSize: 5, hostId: selectedHostId ?? undefined }),
      ]);
      setProtocol(protocolData);
      setOperations(operationPage.list);
    } catch {
      showToast("Docker AIOps 协议加载失败");
    }
  }

  function openHostCreateDrawer() {
    setHostEditId(null);
    setHostForm(defaultHostForm());
    setDrawer("host-create");
  }

  function openHostEditDrawer(item: DockerHostItem) {
    setHostEditId(item.id);
    setHostForm({
      name: item.name,
      endpoint: item.endpoint,
      tlsEnable: Boolean(item.tlsEnable),
      env: item.env ?? "prod",
      owner: item.owner ?? "",
      labelsJSON: JSON.stringify(item.labels ?? {}, null, 2),
      metadataJSON: JSON.stringify(item.metadata ?? {}, null, 2),
    });
    setDrawer("host-edit");
  }

  function openStackCreateDrawer() {
    const form = defaultStackForm();
    if (selectedHostId) form.hostId = String(selectedHostId);
    setStackEditId(null);
    setStackForm(form);
    setDrawer("stack-create");
  }

  function openStackEditDrawer(item: DockerComposeStackItem) {
    setStackEditId(item.id);
    setStackForm({
      hostId: String(item.hostId),
      name: item.name,
      status: item.status ?? "draft",
      services: String(item.services ?? 0),
      content: item.content,
    });
    setDrawer("stack-edit");
  }

  function closeDrawer() {
    setDrawer("closed");
    setHostEditId(null);
    setStackEditId(null);
    setHostForm(defaultHostForm());
    setStackForm(defaultStackForm());
  }

  async function submitHost(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setHostSubmitting(true);
    try {
      const payload = {
        name: hostForm.name.trim(),
        endpoint: hostForm.endpoint.trim(),
        tlsEnable: hostForm.tlsEnable,
        env: hostForm.env.trim(),
        owner: hostForm.owner.trim(),
        labels: parseJSONObject(hostForm.labelsJSON, "labels"),
        metadata: parseJSONObject(hostForm.metadataJSON, "metadata"),
      };
      if (hostEditId) {
        await updateDockerHost(hostEditId, payload);
      } else {
        await createDockerHost(payload);
      }
      closeDrawer();
      await loadHosts();
      showToast("Docker 主机保存成功");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "Docker 主机保存失败");
    } finally {
      setHostSubmitting(false);
    }
  }

  async function submitStack(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setStackSubmitting(true);
    try {
      const payload = {
        hostId: Number(stackForm.hostId),
        name: stackForm.name.trim(),
        status: stackForm.status.trim(),
        services: Number(stackForm.services || 0),
        content: stackForm.content,
      };
      if (stackEditId) {
        await updateComposeStack(stackEditId, payload);
      } else {
        await createComposeStack(payload);
      }
      closeDrawer();
      await loadStacks();
      showToast("Compose Stack 保存成功");
    } catch {
      showToast("Compose Stack 保存失败");
    } finally {
      setStackSubmitting(false);
    }
  }

  async function handleCheckHost(hostId: number) {
    setCheckingHostId(hostId);
    try {
      const result = await checkDockerHost(hostId);
      await loadHosts();
      showToast(`Docker 主机校验${result.status === "connected" ? "成功" : "完成"}`);
    } catch {
      await loadHosts();
      showToast("Docker 主机校验失败");
    } finally {
      setCheckingHostId(null);
    }
  }

  async function handleDeleteHost() {
    if (!deleteHostTarget) return;
    try {
      await deleteDockerHost(deleteHostTarget.id);
      if (selectedHostId === deleteHostTarget.id) setSelectedHostId(null);
      setDeleteHostTarget(null);
      await loadHosts();
      showToast("Docker 主机删除成功");
    } catch {
      showToast("Docker 主机删除失败，请确认是否仍有关联 Compose Stack");
    }
  }

  async function handleDeleteStack() {
    if (!deleteStackTarget) return;
    try {
      await deleteComposeStack(deleteStackTarget.id);
      setDeleteStackTarget(null);
      await loadStacks();
      showToast("Compose Stack 删除成功");
    } catch {
      showToast("Compose Stack 删除失败");
    }
  }

  async function executeDockerAction(payload: DockerActionPayload, actionKey: string, refreshTarget: "resources" | "stacks") {
    setRunningActionKey(actionKey);
    try {
      const result = await runDockerAction(payload);
      setAIOpsResult((result.dryRun ?? result.operation.result ?? {}) as Record<string, unknown>);
      await loadProtocolAndOperations();
      if (!payload.dryRun && refreshTarget === "resources") await loadResources();
      if (!payload.dryRun && refreshTarget === "stacks") await loadStacks();
      showToast(payload.dryRun ? "dry-run 已生成" : "Docker 动作执行成功");
    } catch {
      showToast("Docker 动作执行失败");
    } finally {
      setRunningActionKey(null);
    }
  }

  function requestDockerAction(target: PendingDockerAction) {
    if (target.payload.dryRun || !dockerActionNeedsConfirm(target.payload)) {
      void executeDockerAction(target.payload, target.actionKey, target.refreshTarget);
      return;
    }
    setConfirmActionTarget(target);
  }

  async function confirmDockerAction() {
    if (!confirmActionTarget) return;
    const target = confirmActionTarget;
    setConfirmActionTarget(null);
    await executeDockerAction(
      { ...target.payload, confirmationText: "确认删除资源" },
      target.actionKey,
      target.refreshTarget,
    );
  }

  function requestResourceAction(resource: DockerResourceItem, action: string, dryRun: boolean) {
    if (!selectedHostId) return;
    const actionKey = `${resource.id}-${action}-${dryRun ? "dry" : "run"}`;
    requestDockerAction({
      title: action === "remove" ? "删除 Docker 资源" : "执行 Docker 资源动作",
      description: action === "remove"
        ? `确认删除 ${resource.type} ${resource.name || resource.id}？该操作可能不可恢复，请先确认没有业务依赖。`
        : `确认对 ${resource.type} ${resource.name || resource.id} 执行 ${action}？`,
      actionKey,
      refreshTarget: "resources",
      payload: {
        hostId: selectedHostId,
        resourceType: resource.type,
        resourceId: resource.id,
        action,
        dryRun,
        params: { source: "docker-page" },
      },
    });
  }

  function requestComposeAction(stack: DockerComposeStackItem, action: string, dryRun: boolean) {
    const actionKey = `compose-${stack.id}-${action}-${dryRun ? "dry" : "run"}`;
    requestDockerAction({
      title: "执行 Compose 动作",
      description: `确认对 Compose Stack ${stack.name} 执行 ${action}？该操作可能影响多个容器服务，请先执行 dry-run 确认影响范围。`,
      actionKey,
      refreshTarget: "stacks",
      payload: {
        hostId: stack.hostId,
        resourceType: "compose",
        resourceId: String(stack.id),
        action,
        dryRun,
        params: { source: "docker-page" },
      },
    });
  }

  function handleHostFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setHostPage(1);
    setHostQuery({ ...hostFilter });
  }

  function handleResourceFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setResourcePage(1);
    setResourceQuery({ ...resourceFilter });
  }

  function renderActionsHeader(label: string, onClick: () => void) {
    return (
      <div className="table-actions-header">
        <span>{label}</span>
        <button
          className="table-settings-trigger cursor-pointer"
          type="button"
          onClick={onClick}
          aria-label={`${label}列表字段设置`}
        >
          ⚙️
        </button>
      </div>
    );
  }

  function renderTableHeader(column: TableSettingsColumn, target: Exclude<SettingsTarget, "closed">) {
    if (column.key !== "actions") return column.label;
    return renderActionsHeader(column.label, () => setSettingsTarget(target));
  }

  function renderHostCell(item: DockerHostItem, key: string) {
    switch (key) {
      case "id":
        return item.id;
      case "name":
        return item.name;
      case "endpoint":
        return <code>{item.endpoint}</code>;
      case "env":
        return item.env || "-";
      case "owner":
        return item.owner || "-";
      case "status":
        return <span className={`docker-status docker-status-${item.status ?? "unknown"}`}>{item.status || "unknown"}</span>;
      case "version":
        return item.version || "-";
      case "updatedAt":
        return formatDateTime(item.updatedAt);
      case "actions":
        return (
          <div className="rbac-row-actions">
            <RowActionOverflow
              title="Docker 主机更多操作"
              actions={[
                { key: "select", label: "查看资源", onClick: () => { setSelectedHostId(item.id); setResourcePage(1); } },
                { key: "check", label: checkingHostId === item.id ? "校验中..." : "校验", permissionKey: "button.docker.host.check", disabled: checkingHostId === item.id, onClick: () => void handleCheckHost(item.id) },
                { key: "stack", label: "创建Stack", permissionKey: "button.docker.compose_stack.create", onClick: () => { setSelectedHostId(item.id); setStackForm({ ...defaultStackForm(), hostId: String(item.id) }); setDrawer("stack-create"); } },
                { key: "edit", label: "编辑", permissionKey: "button.docker.host.update", onClick: () => openHostEditDrawer(item) },
                { key: "delete", label: "删除", permissionKey: "button.docker.host.delete", disabled: item.status === "connected", onClick: () => setDeleteHostTarget(item) },
              ]}
            />
          </div>
        );
      default:
        return "-";
    }
  }

  function renderResourceCell(item: DockerResourceItem, key: string) {
    switch (key) {
      case "id":
        return <code>{shorten(item.id)}</code>;
      case "type":
        return item.type;
      case "name":
        return item.name || "-";
      case "status":
        return item.status || "-";
      case "image":
        return item.image || "-";
      case "driver":
        return item.driver || "-";
      case "actions":
        return (
          <div className="rbac-row-actions">
            <RowActionOverflow
              title="Docker 资源更多操作"
              actions={(item.aiopsActions ?? []).flatMap((action) => [
                { key: `${action}-dry`, label: `${action} dry-run`, permissionKey: "button.docker.action.run", disabled: runningActionKey === `${item.id}-${action}-dry`, onClick: () => requestResourceAction(item, action, true) },
                { key: `${action}-run`, label: action === "remove" ? "删除" : action, permissionKey: "button.docker.action.run", disabled: runningActionKey === `${item.id}-${action}-run`, onClick: () => requestResourceAction(item, action, false) },
              ])}
            />
          </div>
        );
      default:
        return "-";
    }
  }

  function renderStackCell(item: DockerComposeStackItem, key: string) {
    switch (key) {
      case "id":
        return item.id;
      case "hostId":
        return item.hostId;
      case "name":
        return item.name;
      case "status":
        return item.status || "-";
      case "services":
        return item.services ?? 0;
      case "updatedAt":
        return formatDateTime(item.updatedAt);
      case "actions":
        return (
          <div className="rbac-row-actions">
            <RowActionOverflow
              title="Compose Stack 更多操作"
              actions={[
                { key: "validate-dry", label: "validate dry-run", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-validate-dry`, onClick: () => requestComposeAction(item, "validate", true) },
                { key: "validate", label: "校验", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-validate-run`, onClick: () => requestComposeAction(item, "validate", false) },
                { key: "deploy-dry", label: "deploy dry-run", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-deploy-dry`, onClick: () => requestComposeAction(item, "deploy", true) },
                { key: "deploy", label: "部署", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-deploy-run`, onClick: () => requestComposeAction(item, "deploy", false) },
                { key: "restart", label: "重启", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-restart-run`, onClick: () => requestComposeAction(item, "restart", false) },
                { key: "down", label: "下线", permissionKey: "button.docker.action.run", disabled: runningActionKey === `compose-${item.id}-down-run`, onClick: () => requestComposeAction(item, "down", false) },
                { key: "edit", label: "编辑", permissionKey: "button.docker.compose_stack.update", onClick: () => openStackEditDrawer(item) },
                { key: "delete", label: "删除", permissionKey: "button.docker.compose_stack.delete", disabled: item.status === "running" || item.status === "deploying", onClick: () => setDeleteStackTarget(item) },
              ]}
            />
          </div>
        );
      default:
        return "-";
    }
  }

  const drawerVisible = drawer !== "closed";
  const hostDrawer = drawer === "host-create" || drawer === "host-edit";
  const stackDrawer = drawer === "stack-create" || drawer === "stack-edit";

  return (
    <section className="page">
      <h2>Docker 管理</h2>
      <div className="rbac-module-grid docker-module-grid">
        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>Docker 主机</h3>
              <p className="muted">纳管 Docker API Endpoint，校验连通性并作为资源操作入口。</p>
            </div>
            <PermissionButton permissionKey="button.docker.host.create" className="btn primary cursor-pointer" type="button" onClick={openHostCreateDrawer}>
              创建主机
            </PermissionButton>
          </header>
          <form className="cloud-filter-bar" onSubmit={handleHostFilterSubmit}>
            <input className="cloud-filter-control cloud-filter-keyword" value={hostFilter.keyword} onChange={(event) => setHostFilter((prev) => ({ ...prev, keyword: event.target.value }))} placeholder="关键词：名称/Endpoint/负责人" />
            <input className="cloud-filter-control" value={hostFilter.env} onChange={(event) => setHostFilter((prev) => ({ ...prev, env: event.target.value }))} placeholder="环境：prod" />
            <select className="cloud-filter-control" value={hostFilter.status} onChange={(event) => setHostFilter((prev) => ({ ...prev, status: event.target.value }))}>
              <option value="">状态：全部</option>
              <option value="connected">connected</option>
              <option value="error">error</option>
              <option value="unknown">unknown</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={hostLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={() => { const next = defaultHostFilter(); setHostFilter(next); setHostQuery(next); }}>重置</button>
            </div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>
                  {visibleHostColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "hosts")}</th>)}
                </tr>
              </thead>
              <tbody>
                {hostLoading ? <tr><td colSpan={visibleHostColumns.length}>加载中...</td></tr> : hosts.list.length === 0 ? <tr><td colSpan={visibleHostColumns.length}>暂无数据</td></tr> : hosts.list.map((item) => (
                  <tr key={item.id} className={selectedHostId === item.id ? "docker-selected-row" : ""}>
                    {visibleHostColumnDefs.map((column) => <td key={column.key}>{renderHostCell(item, column.key)}</td>)}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {paginationFooter(hosts.total, hostPage, hostTotalPages, hostPageSize, setHostPage, setHostPageSize)}
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>Docker 资源</h3>
              <p className="muted">{selectedHost ? `当前主机：${selectedHost.name}` : "请选择主机后查看容器、镜像、网络、数据卷。"}</p>
            </div>
          </header>
          <form className="cloud-filter-bar" onSubmit={handleResourceFilterSubmit}>
            <select className="cloud-filter-control" value={resourceFilter.type} onChange={(event) => setResourceFilter((prev) => ({ ...prev, type: event.target.value as DockerResourceType }))}>
              {resourceTypes.map((type) => <option key={type} value={type}>{type}</option>)}
            </select>
            <input className="cloud-filter-control cloud-filter-keyword" value={resourceFilter.keyword} onChange={(event) => setResourceFilter((prev) => ({ ...prev, keyword: event.target.value }))} placeholder="关键词：名称/状态/镜像" />
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={!selectedHostId || resourceLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={() => { const next = defaultResourceFilter(); setResourceFilter(next); setResourceQuery(next); }}>重置</button>
            </div>
          </form>
          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>{visibleResourceColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "resources")}</th>)}</tr>
              </thead>
              <tbody>
                {resourceLoading ? <tr><td colSpan={visibleResourceColumns.length}>加载中...</td></tr> : resources.list.length === 0 ? <tr><td colSpan={visibleResourceColumns.length}>暂无资源</td></tr> : resources.list.map((item) => (
                  <tr key={`${item.type}-${item.id}`}>
                    {visibleResourceColumnDefs.map((column) => <td key={column.key}>{renderResourceCell(item, column.key)}</td>)}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {paginationFooter(resources.total, resourcePage, resourceTotalPages, resourcePageSize, setResourcePage, setResourcePageSize)}
        </article>

        <article className="card rbac-module-card cloud-module-card">
          <header className="rbac-module-header">
            <div>
              <h3>Compose Stack</h3>
              <p className="muted">保存 Compose 配置，后续部署动作统一走 AIOps dry-run/action 协议。</p>
            </div>
            <PermissionButton permissionKey="button.docker.compose_stack.create" className="btn primary cursor-pointer" type="button" onClick={openStackCreateDrawer}>
              创建Stack
            </PermissionButton>
          </header>
          <div className="rbac-table-wrapper rbac-module-scroll">
            <table className="rbac-table">
              <thead>
                <tr>{visibleStackColumnDefs.map((column) => <th key={column.key}>{renderTableHeader(column, "stacks")}</th>)}</tr>
              </thead>
              <tbody>
                {stackLoading ? <tr><td colSpan={visibleStackColumns.length}>加载中...</td></tr> : stacks.list.length === 0 ? <tr><td colSpan={visibleStackColumns.length}>暂无 Stack</td></tr> : stacks.list.map((item) => (
                  <tr key={item.id}>
                    {visibleStackColumnDefs.map((column) => <td key={column.key}>{renderStackCell(item, column.key)}</td>)}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {paginationFooter(stacks.total, stackPage, stackTotalPages, stackPageSize, setStackPage, setStackPageSize)}
        </article>

        <article className="card rbac-module-card cloud-module-card docker-aiops-card">
          <header className="rbac-module-header">
            <div>
              <h3>AIOps 操作协议</h3>
              <p className="muted">自然语言后续可复用统一 <code>{"{hostId, resourceType, resourceId, action, dryRun, params}"}</code> 协议。</p>
            </div>
          </header>
          <div className="docker-aiops-protocol">
            <div><strong>协议版本</strong><code>{protocol?.protocolVersion ?? "-"}</code></div>
            <div><strong>Action Endpoint</strong><code>{protocol?.actionEndpoint ?? "-"}</code></div>
            <div><strong>支持资源</strong><span>{protocol?.resources?.map((item) => `${item.type}:${item.actions.join("/")}`).join("，") ?? "-"}</span></div>
          </div>
          {aiopsResult ? (
            <pre className="docker-aiops-result">{JSON.stringify(aiopsResult, null, 2)}</pre>
          ) : (
            <p className="muted">点击资源行中的 dry-run 可查看影响范围、风险等级、审批要求和回滚提示。</p>
          )}
          <div className="docker-operation-list">
            {operations.map((operation) => (
              <div key={operation.id} className="docker-operation-item">
                <code>{operation.traceId}</code>
                <span>{operation.resourceType}/{operation.action}</span>
                <span>{operation.status}</span>
              </div>
            ))}
          </div>
        </article>
      </div>

      {drawerVisible && (
        <div className="rbac-drawer-mask">
          <aside className="rbac-drawer">
            <header className="rbac-drawer-header">
              <h3>{hostDrawer ? (hostEditId ? "编辑 Docker 主机" : "创建 Docker 主机") : (stackEditId ? "编辑 Compose Stack" : "创建 Compose Stack")}</h3>
              <button className="btn ghost cursor-pointer" type="button" onClick={closeDrawer}>关闭</button>
            </header>
            {hostDrawer && (
              <form className="rbac-drawer-body form-grid" onSubmit={submitHost}>
                <label>名称<input required value={hostForm.name} onChange={(event) => setHostForm((prev) => ({ ...prev, name: event.target.value }))} /></label>
                <label>Endpoint<input required value={hostForm.endpoint} onChange={(event) => setHostForm((prev) => ({ ...prev, endpoint: event.target.value }))} placeholder="unix:///var/run/docker.sock 或 tcp://10.0.0.1:2376" /></label>
                <label>环境<input value={hostForm.env} onChange={(event) => setHostForm((prev) => ({ ...prev, env: event.target.value }))} /></label>
                <label>负责人<input value={hostForm.owner} onChange={(event) => setHostForm((prev) => ({ ...prev, owner: event.target.value }))} /></label>
                <label className="docker-checkbox-label"><input type="checkbox" checked={hostForm.tlsEnable} onChange={(event) => setHostForm((prev) => ({ ...prev, tlsEnable: event.target.checked }))} />启用 TLS</label>
                <label>Labels JSON<textarea value={hostForm.labelsJSON} onChange={(event) => setHostForm((prev) => ({ ...prev, labelsJSON: event.target.value }))} /></label>
                <label>Metadata JSON<textarea value={hostForm.metadataJSON} onChange={(event) => setHostForm((prev) => ({ ...prev, metadataJSON: event.target.value }))} /></label>
                <div className="rbac-row-actions">
                  <button className="btn primary cursor-pointer" type="submit" disabled={hostSubmitting}>{hostSubmitting ? "保存中..." : "保存"}</button>
                  <button className="btn cursor-pointer" type="button" onClick={closeDrawer}>取消</button>
                </div>
              </form>
            )}
            {stackDrawer && (
              <form className="rbac-drawer-body form-grid" onSubmit={submitStack}>
                <label>主机<select required value={stackForm.hostId} onChange={(event) => setStackForm((prev) => ({ ...prev, hostId: event.target.value }))}>{hosts.list.map((host) => <option key={host.id} value={host.id}>{host.id} / {host.name}</option>)}</select></label>
                <label>名称<input required value={stackForm.name} onChange={(event) => setStackForm((prev) => ({ ...prev, name: event.target.value }))} /></label>
                <label>状态<input value={stackForm.status} onChange={(event) => setStackForm((prev) => ({ ...prev, status: event.target.value }))} /></label>
                <label>服务数<input type="number" min="0" value={stackForm.services} onChange={(event) => setStackForm((prev) => ({ ...prev, services: event.target.value }))} /></label>
                <label>Compose 内容<textarea className="docker-compose-editor" required value={stackForm.content} onChange={(event) => setStackForm((prev) => ({ ...prev, content: event.target.value }))} /></label>
                <div className="rbac-row-actions">
                  <button className="btn primary cursor-pointer" type="submit" disabled={stackSubmitting}>{stackSubmitting ? "保存中..." : "保存"}</button>
                  <button className="btn cursor-pointer" type="button" onClick={closeDrawer}>取消</button>
                </div>
              </form>
            )}
          </aside>
        </div>
      )}

      <DeleteConfirmModal
        open={Boolean(deleteHostTarget)}
        title="删除 Docker 主机"
        description={`确认删除 Docker 主机 ${deleteHostTarget?.name ?? ""}？有关联 Compose Stack 或 connected 状态时不可删除。`}
        confirming={false}
        onCancel={() => setDeleteHostTarget(null)}
        onConfirm={() => void handleDeleteHost()}
      />
      <DeleteConfirmModal
        open={Boolean(deleteStackTarget)}
        title="删除 Compose Stack"
        description={`确认删除 Compose Stack ${deleteStackTarget?.name ?? ""}？运行中或部署中 Stack 不允许删除。`}
        confirming={false}
        onCancel={() => setDeleteStackTarget(null)}
        onConfirm={() => void handleDeleteStack()}
      />
      <DeleteConfirmModal
        open={Boolean(confirmActionTarget)}
        title={confirmActionTarget?.title ?? "Docker 高危操作确认"}
        description={confirmActionTarget?.description ?? "该 Docker 操作需要二次确认。"}
        confirming={Boolean(runningActionKey)}
        onCancel={() => setConfirmActionTarget(null)}
        onConfirm={() => void confirmDockerAction()}
      />
      <TableSettingsModal
        open={settingsTarget !== "closed"}
        title="Docker 列表字段"
        columns={settingsTarget === "resources" ? resourceColumns : settingsTarget === "stacks" ? stackColumns : hostColumns}
        visibleColumnKeys={settingsTarget === "resources" ? visibleResourceColumns : settingsTarget === "stacks" ? visibleStackColumns : visibleHostColumns}
        onClose={() => setSettingsTarget("closed")}
        onToggleColumn={(key) => {
          if (settingsTarget === "resources") setVisibleResourceColumns((prev) => toggleColumn(prev, key, resourceColumns));
          else if (settingsTarget === "stacks") setVisibleStackColumns((prev) => toggleColumn(prev, key, stackColumns));
          else setVisibleHostColumns((prev) => toggleColumn(prev, key, hostColumns));
        }}
        onMoveColumn={(key, direction) => {
          if (settingsTarget === "resources") setVisibleResourceColumns((prev) => moveColumn(prev, key, direction));
          else if (settingsTarget === "stacks") setVisibleStackColumns((prev) => moveColumn(prev, key, direction));
          else setVisibleHostColumns((prev) => moveColumn(prev, key, direction));
        }}
        onReset={() => {
          if (settingsTarget === "resources") setVisibleResourceColumns(sanitizeVisibleColumnKeys(defaultResourceColumns, resourceColumns));
          else if (settingsTarget === "stacks") setVisibleStackColumns(sanitizeVisibleColumnKeys(defaultStackColumns, stackColumns));
          else setVisibleHostColumns(sanitizeVisibleColumnKeys(defaultHostColumns, hostColumns));
        }}
      />
    </section>
  );
}

function paginationFooter(
  total: number,
  page: number,
  totalPage: number,
  pageSize: number,
  setPage: (updater: (page: number) => number) => void,
  setPageSize: (value: number) => void,
) {
  return (
    <footer className="rbac-pagination">
      <div className="rbac-pagination-group">
        <span>总计 {total} 条</span>
        <select className="rbac-pagination-select cursor-pointer" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(() => 1); }}>
          {pageSizeOptions.map((option) => <option key={option} value={option}>{option}/页</option>)}
        </select>
      </div>
      <div className="rbac-pagination-group">
        <button className="btn cursor-pointer" type="button" disabled={page <= 1} onClick={() => setPage((current) => Math.max(1, current - 1))}>上一页</button>
        <span className="rbac-pagination-text">{page} / {totalPage}</span>
        <button className="btn cursor-pointer" type="button" disabled={page >= totalPage} onClick={() => setPage((current) => Math.min(totalPage, current + 1))}>下一页</button>
      </div>
    </footer>
  );
}

function toggleColumn(current: string[], columnKey: string, columns: TableSettingsColumn[]) {
  const column = columns.find((item) => item.key === columnKey);
  if (!column || column.required) return current;
  const next = current.includes(columnKey) ? current.filter((key) => key !== columnKey) : [...current, columnKey];
  return sanitizeVisibleColumnKeys(next, columns);
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
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`${label} 必须是 JSON 对象`);
  }
  return parsed as Record<string, unknown>;
}

function totalPages(total: number, pageSize: number) {
  return Math.max(1, Math.ceil(total / pageSize));
}

function dockerActionNeedsConfirm(payload: DockerActionPayload) {
  if (payload.action === "remove") return true;
  return payload.resourceType === "compose" && ["deploy", "up", "down", "restart"].includes(payload.action);
}

function shorten(value: string) {
  if (!value || value.length <= 18) return value || "-";
  return `${value.slice(0, 12)}...${value.slice(-6)}`;
}

function formatDateTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}
