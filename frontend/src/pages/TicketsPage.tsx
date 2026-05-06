import { FormEvent, useEffect, useMemo, useState } from "react";

import type { PageData } from "@/api/types";
import {
  approveTicket,
  cancelTicket,
  createTicket,
  createTicketComment,
  createTicketLink,
  deleteTicket,
  dryRunTicketOperation,
  executeTicketOperation,
  getTicket,
  getTicketAIOpsProtocol,
  listTicketTemplates,
  listTickets,
  rejectTicket,
  reopenTicket,
  submitTicket,
  transitionTicket,
  updateTicket,
} from "@/api/tickets";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { PermissionButton } from "@/components/PermissionButton";
import type { RowActionItem } from "@/components/RowActionOverflow";
import { RowActionOverflow } from "@/components/RowActionOverflow";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type { TicketAIOpsProtocol, TicketItem, TicketOperationPayload, TicketOperationResult, TicketPriority, TicketStatus, TicketSummary, TicketTemplateItem, TicketType } from "@/types/tickets";
import { loadPersistedListSettings, sanitizeVisibleColumnKeys, savePersistedListSettings } from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const ticketSettingsKey = "tickets.table.settings";
const confirmText = "确认删除资源";

const ticketTypes: TicketType[] = ["event", "change", "release", "resource_request", "permission_request", "incident", "service_request"];
const ticketStatuses: TicketStatus[] = ["draft", "submitted", "assigned", "processing", "pending_approval", "approved", "rejected", "resolved", "closed", "cancelled"];
const priorities: TicketPriority[] = ["P0", "P1", "P2", "P3", "P4"];

const ticketColumns: TableSettingsColumn[] = [
  { key: "ticketNo", label: "工单号" },
  { key: "title", label: "标题" },
  { key: "type", label: "类型" },
  { key: "status", label: "状态" },
  { key: "priority", label: "优先级" },
  { key: "env", label: "环境" },
  { key: "assigneeId", label: "处理人" },
  { key: "slaDueAt", label: "SLA" },
  { key: "createdAt", label: "创建时间" },
  { key: "actions", label: "操作", required: true },
];
const defaultVisibleColumns = ["ticketNo", "title", "type", "status", "priority", "env", "assigneeId", "slaDueAt", "actions"];

const templateExamples: Record<TicketType, { title: string; description: string; metadata: Record<string, unknown>; tags: Record<string, unknown> }> = {
  event: { title: "业务告警处理", description: "请描述事件现象、影响范围和期望处理结果。", metadata: { source: "monitor", impact: "single-service" }, tags: { category: "event" } },
  change: { title: "生产变更申请", description: "请描述变更目标、窗口期、回滚方案和验证步骤。", metadata: { window: "22:00-23:00", rollback: "restore previous config" }, tags: { category: "change" } },
  release: { title: "应用发布申请", description: "请描述发布版本、服务、环境、验证和回滚方案。", metadata: { version: "v1.0.0", strategy: "rolling" }, tags: { category: "release" } },
  resource_request: { title: "云资源申请", description: "请描述云厂商、地域、规格、数量、使用期限和预算。", metadata: { provider: "tencent", region: "ap-guangzhou", resourceType: "cvm" }, tags: { category: "resource" } },
  permission_request: { title: "权限申请", description: "请描述账号、权限范围、有效期和审批依据。", metadata: { system: "aiops", scope: "read-only", expireDays: 7 }, tags: { category: "permission" } },
  incident: { title: "故障处理", description: "请描述故障时间线、影响、当前状态和恢复动作。", metadata: { severity: "P1", customerImpact: true }, tags: { category: "incident" } },
  service_request: { title: "服务请求", description: "请描述服务诉求、期望完成时间和业务背景。", metadata: { service: "ops", requesterTeam: "business" }, tags: { category: "service" } },
};

interface TicketFilter {
  keyword: string;
  type: string;
  status: string;
  priority: string;
  env: string;
}

interface TicketForm {
  title: string;
  description: string;
  type: TicketType;
  priority: TicketPriority;
  severity: TicketPriority;
  env: string;
  requesterId: string;
  assigneeId: string;
  departmentId: string;
  slaDueAt: string;
  dueAt: string;
  tagsJSON: string;
  metadataJSON: string;
}

type DrawerState = "closed" | "create" | "edit" | "detail";
type ConfirmTarget = { type: "delete"; item: TicketItem } | { type: "execute"; item: TicketItem; payload: TicketOperationPayload };

function defaultFilter(): TicketFilter {
  return { keyword: "", type: "", status: "", priority: "", env: "" };
}

function defaultForm(type: TicketType = "event"): TicketForm {
  const template = templateExamples[type];
  return {
    title: template.title,
    description: template.description,
    type,
    priority: type === "incident" ? "P0" : type === "change" ? "P1" : "P3",
    severity: type === "incident" ? "P0" : "P3",
    env: "prod",
    requesterId: "",
    assigneeId: "",
    departmentId: "",
    slaDueAt: "",
    dueAt: "",
    tagsJSON: prettyJSON(template.tags),
    metadataJSON: prettyJSON(template.metadata),
  };
}

export function TicketsPage() {
  const [tickets, setTickets] = useState<PageData<TicketItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [filter, setFilter] = useState(defaultFilter);
  const [query, setQuery] = useState(defaultFilter);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [drawer, setDrawer] = useState<DrawerState>("closed");
  const [editId, setEditId] = useState<number | null>(null);
  const [form, setForm] = useState(defaultForm);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [summary, setSummary] = useState<TicketSummary | null>(null);
  const [comment, setComment] = useState("");
  const [linkForm, setLinkForm] = useState({ linkModule: "cmdb", linkType: "ci", linkId: "", linkName: "", relation: "related" });
  const [operationForm, setOperationForm] = useState({ module: "tasks", action: "dry-run" });
  const [operationResult, setOperationResult] = useState<TicketOperationResult | null>(null);
  const [protocol, setProtocol] = useState<TicketAIOpsProtocol | null>(null);
  const [templates, setTemplates] = useState<TicketTemplateItem[]>([]);
  const [confirmTarget, setConfirmTarget] = useState<ConfirmTarget | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [visibleColumns, setVisibleColumns] = useState(() => {
    const persisted = loadPersistedListSettings(ticketSettingsKey);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaultVisibleColumns, ticketColumns);
  });

  const totalPagesValue = useMemo(() => totalPages(tickets.total, pageSize), [tickets.total, pageSize]);
  const visibleOrderedColumns = useMemo(
    () => visibleColumns.map((key) => ticketColumns.find((column) => column.key === key)).filter((column): column is TableSettingsColumn => Boolean(column)),
    [visibleColumns],
  );
  const selectedTicket = useMemo(() => tickets.list.find((item) => item.id === selectedId) ?? summary?.ticket ?? null, [tickets.list, selectedId, summary]);

  useEffect(() => {
    void loadTickets();
  }, [page, pageSize, query]);

  useEffect(() => {
    savePersistedListSettings(ticketSettingsKey, { visibleColumnKeys: visibleColumns });
  }, [visibleColumns]);

  useEffect(() => {
    void loadProtocol();
    void loadTemplates();
  }, []);

  async function loadTickets() {
    setLoading(true);
    try {
      const result = await listTickets({ page, pageSize, keyword: query.keyword || undefined, type: query.type || undefined, status: query.status || undefined, priority: query.priority || undefined, env: query.env || undefined });
      setTickets(result);
      if (!selectedId && result.list.length > 0) setSelectedId(result.list[0].id);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载工单失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadSummary(ticketId: number) {
    try {
      const result = await getTicket(ticketId);
      setSummary(result);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载工单详情失败");
    }
  }

  async function loadProtocol() {
    try {
      setProtocol(await getTicketAIOpsProtocol());
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载工单AIOps协议失败");
    }
  }

  async function loadTemplates() {
    try {
      const result = await listTicketTemplates();
      setTemplates(result.list);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "加载工单模板失败");
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

  function openEditDrawer(item: TicketItem) {
    setEditId(item.id);
    setForm({
      title: item.title,
      description: item.description ?? "",
      type: normalizeTicketType(item.type),
      priority: normalizePriority(item.priority),
      severity: normalizePriority(item.severity || item.priority),
      env: item.env || "prod",
      requesterId: item.requesterId ? String(item.requesterId) : "",
      assigneeId: item.assigneeId ? String(item.assigneeId) : "",
      departmentId: item.departmentId ? String(item.departmentId) : "",
      slaDueAt: toDatetimeLocal(item.slaDueAt),
      dueAt: toDatetimeLocal(item.dueAt),
      tagsJSON: prettyJSON(item.tags ?? {}),
      metadataJSON: prettyJSON(item.metadata ?? {}),
    });
    setDrawer("edit");
  }

  function openDetailDrawer(item: TicketItem) {
    setSelectedId(item.id);
    setDrawer("detail");
    void loadSummary(item.id);
  }

  function applyTemplate(type: TicketType) {
    setForm((current) => ({ ...defaultForm(type), requesterId: current.requesterId, assigneeId: current.assigneeId, departmentId: current.departmentId }));
  }

  async function submitForm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    try {
      const payload = {
        title: form.title,
        description: form.description,
        type: form.type,
        priority: form.priority,
        severity: form.severity,
        env: form.env,
        requesterId: parseOptionalNumber(form.requesterId),
        assigneeId: parseOptionalNumber(form.assigneeId),
        departmentId: parseOptionalNumber(form.departmentId),
        slaDueAt: form.slaDueAt ? new Date(form.slaDueAt).toISOString() : undefined,
        dueAt: form.dueAt ? new Date(form.dueAt).toISOString() : undefined,
        tags: parseJSONObject(form.tagsJSON, "Tags JSON"),
        metadata: parseJSONObject(form.metadataJSON, "Metadata JSON"),
      };
      if (drawer === "edit" && editId) {
        await updateTicket(editId, payload);
        showToast("工单已更新");
      } else {
        const created = await createTicket(payload);
        setSelectedId(created.id);
        showToast("工单已创建");
      }
      setDrawer("closed");
      await loadTickets();
    } catch (error) {
      showToast(error instanceof Error ? error.message : "保存工单失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function runSimpleAction(item: TicketItem, action: string) {
    try {
      if (action === "submit") await submitTicket(item.id);
      if (action === "approve") await approveTicket(item.id, "页面审批通过");
      if (action === "reject") await rejectTicket(item.id, "页面审批驳回");
      if (action === "resolve") await transitionTicket(item.id, "resolved", "处理完成");
      if (action === "close") await transitionTicket(item.id, "closed", "关闭工单");
      if (action === "cancel") await cancelTicket(item.id);
      if (action === "reopen") await reopenTicket(item.id);
      showToast("工单操作已完成");
      await loadTickets();
      if (drawer === "detail") await loadSummary(item.id);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "工单操作失败");
    }
  }

  async function submitComment() {
    if (!selectedTicket || !comment.trim()) return;
    try {
      await createTicketComment(selectedTicket.id, { content: comment.trim() });
      setComment("");
      await loadSummary(selectedTicket.id);
      showToast("评论已添加");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "添加评论失败");
    }
  }

  async function submitLink() {
    if (!selectedTicket || !linkForm.linkId.trim()) return;
    try {
      await createTicketLink(selectedTicket.id, { ...linkForm });
      setLinkForm({ linkModule: "cmdb", linkType: "ci", linkId: "", linkName: "", relation: "related" });
      await loadSummary(selectedTicket.id);
      showToast("关联对象已添加");
    } catch (error) {
      showToast(error instanceof Error ? error.message : "添加关联失败");
    }
  }

  async function runOperation(dryRun: boolean) {
    if (!selectedTicket) return;
    const payload: TicketOperationPayload = { module: operationForm.module, action: operationForm.action, params: {} };
    try {
      if (dryRun) {
        const result = await dryRunTicketOperation(selectedTicket.id, payload);
        setOperationResult(result);
        await loadSummary(selectedTicket.id);
        showToast("dry-run 已生成");
        return;
      }
      setConfirmTarget({ type: "execute", item: selectedTicket, payload });
    } catch (error) {
      showToast(error instanceof Error ? error.message : "执行工单动作失败");
    }
  }

  async function confirmDanger() {
    if (!confirmTarget) return;
    setConfirming(true);
    try {
      if (confirmTarget.type === "delete") {
        await deleteTicket(confirmTarget.item.id, confirmText);
        showToast("工单已删除");
        if (selectedId === confirmTarget.item.id) setSelectedId(null);
        await loadTickets();
      } else {
        const result = await executeTicketOperation(confirmTarget.item.id, { ...confirmTarget.payload, confirmationText: confirmText });
        setOperationResult(result);
        await loadSummary(confirmTarget.item.id);
        showToast("工单执行动作已提交");
      }
      setConfirmTarget(null);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "确认操作失败");
    } finally {
      setConfirming(false);
    }
  }

  function rowActions(item: TicketItem): RowActionItem[] {
    return [
      { key: "detail", label: "详情", onClick: () => openDetailDrawer(item) },
      { key: "edit", label: "编辑", onClick: () => openEditDrawer(item), disabled: !isEditable(item.status), permissionKey: "button.tickets.ticket.update" },
      { key: "submit", label: "提交", onClick: () => void runSimpleAction(item, "submit"), disabled: !["draft", "rejected"].includes(String(item.status)), permissionKey: "button.tickets.ticket.transition" },
      { key: "approve", label: "审批", onClick: () => void runSimpleAction(item, "approve"), disabled: !canApprove(item.status), permissionKey: "button.tickets.ticket.approve" },
      { key: "reject", label: "驳回", onClick: () => void runSimpleAction(item, "reject"), disabled: !canApprove(item.status), permissionKey: "button.tickets.ticket.approve" },
      { key: "resolve", label: "解决", onClick: () => void runSimpleAction(item, "resolve"), disabled: !["processing", "approved"].includes(String(item.status)), permissionKey: "button.tickets.ticket.transition" },
      { key: "close", label: "关闭", onClick: () => void runSimpleAction(item, "close"), disabled: String(item.status) !== "resolved", permissionKey: "button.tickets.ticket.transition" },
      { key: "cancel", label: "取消", onClick: () => void runSimpleAction(item, "cancel"), disabled: ["closed", "cancelled"].includes(String(item.status)), permissionKey: "button.tickets.ticket.transition" },
      { key: "reopen", label: "重开", onClick: () => void runSimpleAction(item, "reopen"), disabled: !["resolved", "closed", "rejected"].includes(String(item.status)), permissionKey: "button.tickets.ticket.transition" },
      { key: "delete", label: "删除", onClick: () => setConfirmTarget({ type: "delete", item }), disabled: !isDeletable(item.status), permissionKey: "button.tickets.ticket.delete", className: "btn ghost cursor-pointer" },
    ];
  }

  function toggleColumn(key: string) {
    const column = ticketColumns.find((item) => item.key === key);
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
              <h3>工单管理</h3>
              <p className="muted">工单创建、审批、流转、关联资源、时间线审计和 AIOps dry-run 协议。</p>
            </div>
            <div className="rbac-actions">
              <PermissionButton permissionKey="button.tickets.ticket.create" className="btn primary cursor-pointer" type="button" onClick={openCreateDrawer}>创建工单</PermissionButton>
            </div>
          </header>

          <form className="cloud-filter-bar" onSubmit={submitSearch}>
            <input className="cloud-filter-control cloud-filter-keyword" placeholder="搜索工单号 / 标题 / 描述" value={filter.keyword} onChange={(event) => setFilter((current) => ({ ...current, keyword: event.target.value }))} />
            <select className="cloud-filter-control" value={filter.type} onChange={(event) => setFilter((current) => ({ ...current, type: event.target.value }))}>
              <option value="">全部类型</option>
              {ticketTypes.map((item) => <option key={item} value={item}>{ticketTypeLabel(item)}</option>)}
            </select>
            <select className="cloud-filter-control" value={filter.status} onChange={(event) => setFilter((current) => ({ ...current, status: event.target.value }))}>
              <option value="">全部状态</option>
              {ticketStatuses.map((item) => <option key={item} value={item}>{ticketStatusLabel(item)}</option>)}
            </select>
            <select className="cloud-filter-control" value={filter.priority} onChange={(event) => setFilter((current) => ({ ...current, priority: event.target.value }))}>
              <option value="">全部优先级</option>
              {priorities.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
            <input className="cloud-filter-control" placeholder="环境" value={filter.env} onChange={(event) => setFilter((current) => ({ ...current, env: event.target.value }))} />
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
                    <th key={column.key}>{column.key === "actions" ? <div className="table-actions-header"><span>{column.label}</span><button className="table-settings-trigger cursor-pointer" type="button" onClick={() => setSettingsOpen(true)} aria-label="工单列表字段设置">⚙️</button></div> : column.label}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={visibleColumns.length}>加载中...</td></tr>
                ) : tickets.list.length === 0 ? (
                  <tr><td colSpan={visibleColumns.length}>暂无工单</td></tr>
                ) : tickets.list.map((item) => (
                  <tr key={item.id} className={selectedId === item.id ? "docker-selected-row" : undefined} onClick={() => setSelectedId(item.id)}>
                    {visibleOrderedColumns.map((column) => (
                      <td key={column.key} onClick={column.key === "actions" ? (event) => event.stopPropagation() : undefined}>{renderTicketCell(column.key, item, rowActions(item))}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <Pagination page={page} pageSize={pageSize} total={tickets.total} totalPages={totalPagesValue} onPageChange={setPage} onPageSizeChange={(next) => { setPageSize(next); setPage(1); }} />
        </div>
      </section>

      <section className="grid cards">
        <div className="card docker-aiops-card">
          <h3>AIOps 协议</h3>
          <p className="muted">{protocol?.protocolVersion || "未加载"}</p>
          <div className="docker-aiops-protocol"><div><strong>类型</strong><span>{protocol?.types?.join(" / ") || "-"}</span></div><div><strong>安全</strong><span>默认 dry-run，真实执行需审批和确认</span></div></div>
        </div>
        <div className="card docker-aiops-card">
          <h3>内置模板</h3>
          <div className="docker-operation-list">{templates.slice(0, 5).map((item) => <div className="docker-operation-item" key={`${item.type}-${item.name}`}><strong>{item.name}</strong><span>{ticketTypeLabel(item.type)}</span><span>{item.defaultPriority || "P3"}</span></div>)}</div>
        </div>
        <div className="card docker-aiops-card">
          <h3>当前选择</h3>
          <p className="muted">{selectedTicket ? `${selectedTicket.ticketNo} ${selectedTicket.title}` : "未选择工单"}</p>
          <button className="btn ghost cursor-pointer" type="button" disabled={!selectedTicket} onClick={() => selectedTicket && openDetailDrawer(selectedTicket)}>查看详情</button>
        </div>
      </section>

      {operationResult ? <section className="card docker-aiops-card"><header className="rbac-module-header"><div><h3>最近执行结果</h3><p className="muted">保留 traceId、riskLevel、rollback、safetyChecks，后续可接入 AIOpsChat。</p></div><button className="btn ghost cursor-pointer" type="button" onClick={() => setOperationResult(null)}>清空</button></header><pre className="docker-aiops-result">{prettyJSON(operationResult)}</pre></section> : null}

      <TicketDrawer drawer={drawer} form={form} summary={summary} selectedTicket={selectedTicket} submitting={submitting} comment={comment} linkForm={linkForm} operationForm={operationForm} onClose={() => setDrawer("closed")} onSubmit={submitForm} onFormChange={setForm} onTemplate={applyTemplate} onCommentChange={setComment} onSubmitComment={() => void submitComment()} onLinkFormChange={setLinkForm} onSubmitLink={() => void submitLink()} onOperationFormChange={setOperationForm} onDryRun={() => void runOperation(true)} onExecute={() => void runOperation(false)} />

      <DeleteConfirmModal open={confirmTarget !== null} title={confirmTarget?.type === "execute" ? "高危执行确认" : "删除工单"} description={confirmTarget?.type === "execute" ? "真实执行会写入操作记录并可能触发外部模块，请先确认审批和 dry-run 结果。" : "删除会清理工单流转、评论、关联对象和操作记录。"} confirming={confirming} onCancel={() => setConfirmTarget(null)} onConfirm={() => void confirmDanger()} />

      <TableSettingsModal open={settingsOpen} title="工单列表字段" columns={ticketColumns} visibleColumnKeys={visibleColumns} onToggleColumn={toggleColumn} onMoveColumn={moveColumn} onReset={() => setVisibleColumns(defaultVisibleColumns)} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}

interface TicketDrawerProps {
  drawer: DrawerState;
  form: TicketForm;
  summary: TicketSummary | null;
  selectedTicket: TicketItem | null;
  submitting: boolean;
  comment: string;
  linkForm: { linkModule: string; linkType: string; linkId: string; linkName: string; relation: string };
  operationForm: { module: string; action: string };
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onFormChange: (next: TicketForm | ((current: TicketForm) => TicketForm)) => void;
  onTemplate: (type: TicketType) => void;
  onCommentChange: (value: string) => void;
  onSubmitComment: () => void;
  onLinkFormChange: (next: { linkModule: string; linkType: string; linkId: string; linkName: string; relation: string } | ((current: { linkModule: string; linkType: string; linkId: string; linkName: string; relation: string }) => { linkModule: string; linkType: string; linkId: string; linkName: string; relation: string })) => void;
  onSubmitLink: () => void;
  onOperationFormChange: (next: { module: string; action: string } | ((current: { module: string; action: string }) => { module: string; action: string })) => void;
  onDryRun: () => void;
  onExecute: () => void;
}

function TicketDrawer(props: TicketDrawerProps) {
  const { drawer, form, summary, selectedTicket, submitting, comment, linkForm, operationForm, onClose, onSubmit, onFormChange, onTemplate, onCommentChange, onSubmitComment, onLinkFormChange, onSubmitLink, onOperationFormChange, onDryRun, onExecute } = props;
  if (drawer === "closed") return null;
  const detailMode = drawer === "detail";
  return (
    <div className="rbac-drawer-mask" onClick={onClose}>
      <aside className="rbac-drawer" onClick={(event) => event.stopPropagation()}>
        <header className="rbac-drawer-header"><h3>{drawer === "create" ? "创建工单" : drawer === "edit" ? "编辑工单" : "工单详情"}</h3><button className="btn ghost cursor-pointer" type="button" onClick={onClose}>关闭</button></header>
        <div className="rbac-drawer-body">
          {detailMode ? (
            <div className="ticket-detail-stack">
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>{selectedTicket?.title || "工单"}</h4><p className="muted">{selectedTicket?.ticketNo} / {ticketStatusLabel(selectedTicket?.status)}</p></div><pre className="docker-aiops-result">{prettyJSON(summary?.ticket ?? selectedTicket ?? {})}</pre></section>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>时间线</h4><p className="muted">状态流转、评论、关联与执行动作审计。</p></div><div className="docker-operation-list">{summary?.flows?.length ? summary.flows.map((item) => <div className="docker-operation-item" key={item.id}><strong>{item.action}</strong><span>{item.fromStatus || "-"} -&gt; {item.toStatus || "-"}</span><span className="muted">{item.comment || "-"}</span><span className="muted">{formatTime(item.createdAt)}</span></div>) : <span className="muted">暂无时间线</span>}</div></section>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>评论</h4><p className="muted">处理记录和内部协作信息。</p></div><div className="docker-operation-list">{summary?.comments?.map((item) => <div className="docker-operation-item" key={item.id}><strong>#{item.id}</strong><span>{item.content}</span><span className="muted">{formatTime(item.createdAt)}</span></div>)}</div><textarea className="middleware-json-editor" value={comment} placeholder="添加评论" onChange={(event) => onCommentChange(event.target.value)} /><div className="middleware-form-actions"><button className="btn primary cursor-pointer" type="button" onClick={onSubmitComment}>添加评论</button></div></section>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>关联对象</h4><p className="muted">可关联 CMDB、云资源、Docker、中间件、任务等对象。</p></div><div className="docker-operation-list">{summary?.links?.map((item) => <div className="docker-operation-item" key={item.id}><strong>{item.linkModule}</strong><span>{item.linkType}:{item.linkId}</span><span>{item.linkName || "-"}</span></div>)}</div><div className="middleware-form-grid"><label className="middleware-form-field"><span>模块</span><select value={linkForm.linkModule} onChange={(event) => onLinkFormChange((current) => ({ ...current, linkModule: event.target.value }))}><option value="cmdb">CMDB</option><option value="cloud">多云</option><option value="docker">Docker</option><option value="middleware">中间件</option><option value="tasks">任务</option></select></label><label className="middleware-form-field"><span>类型</span><input value={linkForm.linkType} onChange={(event) => onLinkFormChange((current) => ({ ...current, linkType: event.target.value }))} /></label><label className="middleware-form-field"><span>ID</span><input value={linkForm.linkId} onChange={(event) => onLinkFormChange((current) => ({ ...current, linkId: event.target.value }))} /></label><label className="middleware-form-field"><span>名称</span><input value={linkForm.linkName} onChange={(event) => onLinkFormChange((current) => ({ ...current, linkName: event.target.value }))} /></label></div><div className="middleware-form-actions"><button className="btn primary cursor-pointer" type="button" onClick={onSubmitLink}>添加关联</button></div></section>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>AIOps 执行动作</h4><p className="muted">默认先 dry-run，真实执行需二次确认。</p></div><div className="middleware-form-grid"><label className="middleware-form-field"><span>模块</span><select value={operationForm.module} onChange={(event) => onOperationFormChange((current) => ({ ...current, module: event.target.value }))}><option value="tasks">任务中心</option><option value="cloud">多云</option><option value="docker">Docker</option><option value="middleware">中间件</option></select></label><label className="middleware-form-field"><span>动作</span><input value={operationForm.action} onChange={(event) => onOperationFormChange((current) => ({ ...current, action: event.target.value }))} /></label></div><div className="middleware-form-actions"><button className="btn ghost cursor-pointer" type="button" onClick={onDryRun}>Dry-run</button><button className="btn primary cursor-pointer" type="button" onClick={onExecute}>真实执行</button></div></section>
            </div>
          ) : (
            <form className="middleware-form" onSubmit={onSubmit}>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>基础信息</h4><p className="muted">定义工单类型、优先级和处理目标。</p></div><div className="middleware-form-grid"><label className="middleware-form-field middleware-form-field-wide"><span>模板</span><select value={form.type} onChange={(event) => onTemplate(normalizeTicketType(event.target.value))}>{ticketTypes.map((item) => <option key={item} value={item}>{ticketTypeLabel(item)}</option>)}</select></label><label className="middleware-form-field middleware-form-field-wide"><span>标题</span><input value={form.title} required onChange={(event) => onFormChange((current) => ({ ...current, title: event.target.value }))} /></label><label className="middleware-form-field"><span>优先级</span><select value={form.priority} onChange={(event) => onFormChange((current) => ({ ...current, priority: normalizePriority(event.target.value) }))}>{priorities.map((item) => <option key={item} value={item}>{item}</option>)}</select></label><label className="middleware-form-field"><span>严重级别</span><select value={form.severity} onChange={(event) => onFormChange((current) => ({ ...current, severity: normalizePriority(event.target.value) }))}>{priorities.map((item) => <option key={item} value={item}>{item}</option>)}</select></label><label className="middleware-form-field"><span>环境</span><input value={form.env} onChange={(event) => onFormChange((current) => ({ ...current, env: event.target.value }))} /></label><label className="middleware-form-field"><span>处理人ID</span><input value={form.assigneeId} inputMode="numeric" onChange={(event) => onFormChange((current) => ({ ...current, assigneeId: event.target.value }))} /></label><label className="middleware-form-field middleware-form-field-wide"><span>描述</span><textarea value={form.description} onChange={(event) => onFormChange((current) => ({ ...current, description: event.target.value }))} /></label></div></section>
              <section className="middleware-form-section"><div className="middleware-form-section-title"><h4>影响与SLA</h4><p className="muted">记录部门、申请人、截止时间和结构化元数据。</p></div><div className="middleware-form-grid"><label className="middleware-form-field"><span>申请人ID</span><input value={form.requesterId} inputMode="numeric" onChange={(event) => onFormChange((current) => ({ ...current, requesterId: event.target.value }))} /></label><label className="middleware-form-field"><span>部门ID</span><input value={form.departmentId} inputMode="numeric" onChange={(event) => onFormChange((current) => ({ ...current, departmentId: event.target.value }))} /></label><label className="middleware-form-field"><span>SLA截止</span><input type="datetime-local" value={form.slaDueAt} onChange={(event) => onFormChange((current) => ({ ...current, slaDueAt: event.target.value }))} /></label><label className="middleware-form-field"><span>期望完成</span><input type="datetime-local" value={form.dueAt} onChange={(event) => onFormChange((current) => ({ ...current, dueAt: event.target.value }))} /></label><label className="middleware-form-field middleware-form-field-wide"><span>Tags JSON</span><textarea className="middleware-json-editor" value={form.tagsJSON} onChange={(event) => onFormChange((current) => ({ ...current, tagsJSON: event.target.value }))} /></label><label className="middleware-form-field middleware-form-field-wide"><span>Metadata JSON</span><textarea className="middleware-json-editor" value={form.metadataJSON} onChange={(event) => onFormChange((current) => ({ ...current, metadataJSON: event.target.value }))} /></label></div></section>
              <div className="middleware-form-actions"><button className="btn primary cursor-pointer" type="submit" disabled={submitting}>{submitting ? "保存中..." : "保存"}</button><button className="btn ghost cursor-pointer" type="button" onClick={onClose}>取消</button></div>
            </form>
          )}
        </div>
      </aside>
    </div>
  );
}

function renderTicketCell(columnKey: string, item: TicketItem, actions: RowActionItem[]) {
  switch (columnKey) {
    case "ticketNo": return item.ticketNo || `#${item.id}`;
    case "title": return <span className="ticket-title-cell">{item.title}</span>;
    case "type": return ticketTypeLabel(item.type);
    case "status": return <StatusBadge status={item.status} />;
    case "priority": return <PriorityBadge priority={item.priority} />;
    case "env": return item.env || "-";
    case "assigneeId": return item.assigneeId || "-";
    case "slaDueAt": return formatTime(item.slaDueAt);
    case "createdAt": return formatTime(item.createdAt);
    case "actions": return <div className="rbac-row-actions"><RowActionOverflow actions={actions} /></div>;
    default: return "-";
  }
}

function Pagination({ page, pageSize, total, totalPages: pageTotal, onPageChange, onPageSizeChange }: { page: number; pageSize: number; total: number; totalPages: number; onPageChange: (page: number) => void; onPageSizeChange: (pageSize: number) => void }) {
  return <div className="rbac-pagination"><div className="rbac-pagination-group"><span className="muted">共 {total} 条</span></div><div className="rbac-pagination-group"><select className="rbac-pagination-select" value={pageSize} onChange={(event) => onPageSizeChange(Number(event.target.value))}>{pageSizeOptions.map((item) => <option key={item} value={item}>{item}条</option>)}</select><button className="btn ghost cursor-pointer" type="button" disabled={page <= 1} onClick={() => onPageChange(Math.max(1, page - 1))}>上一页</button><span className="rbac-pagination-text">{page} / {pageTotal}</span><button className="btn ghost cursor-pointer" type="button" disabled={page >= pageTotal} onClick={() => onPageChange(Math.min(pageTotal, page + 1))}>下一页</button></div></div>;
}

function StatusBadge({ status }: { status?: string }) {
  const normalized = String(status || "unknown").toLowerCase();
  const className = ["resolved", "closed", "approved"].includes(normalized) ? "docker-status docker-status-connected" : ["rejected", "cancelled"].includes(normalized) ? "docker-status docker-status-error" : "docker-status";
  return <span className={className}>{ticketStatusLabel(status)}</span>;
}

function PriorityBadge({ priority }: { priority?: string }) {
  const normalized = String(priority || "P3").toUpperCase();
  const className = normalized === "P0" || normalized === "P1" ? "message-severity message-severity-critical" : normalized === "P2" ? "message-severity message-severity-warning" : "message-severity";
  return <span className={className}>{normalized}</span>;
}

function ticketTypeLabel(value?: string): string {
  const labels: Record<string, string> = { event: "事件", change: "变更", release: "发布", resource_request: "资源申请", permission_request: "权限申请", incident: "故障", service_request: "服务请求" };
  return labels[String(value)] || String(value || "-");
}

function ticketStatusLabel(value?: string): string {
  const labels: Record<string, string> = { draft: "草稿", submitted: "已提交", assigned: "已指派", processing: "处理中", pending_approval: "待审批", approved: "已审批", rejected: "已驳回", resolved: "已解决", closed: "已关闭", cancelled: "已取消" };
  return labels[String(value)] || String(value || "-");
}

function normalizeTicketType(value: string): TicketType {
  return ticketTypes.includes(value as TicketType) ? value as TicketType : "event";
}

function normalizePriority(value?: string): TicketPriority {
  const upper = String(value || "P3").toUpperCase();
  return priorities.includes(upper as TicketPriority) ? upper as TicketPriority : "P3";
}

function parseJSONObject(value: string, label: string): Record<string, unknown> {
  const parsed = JSON.parse(value || "{}");
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error(`${label} 必须是 JSON object`);
  return parsed as Record<string, unknown>;
}

function parseOptionalNumber(value: string): number | undefined {
  const text = value.trim();
  if (!text) return undefined;
  const parsed = Number(text);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
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

function toDatetimeLocal(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}

function isEditable(status?: string): boolean {
  return ["draft", "submitted", "assigned", "rejected"].includes(String(status));
}

function isDeletable(status?: string): boolean {
  return ["draft", "cancelled", "rejected", "closed"].includes(String(status));
}

function canApprove(status?: string): boolean {
  return ["submitted", "pending_approval", "approved"].includes(String(status));
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
