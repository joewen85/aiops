import { FormEvent, useEffect, useMemo, useState } from "react";

import { createMessage, listMessages, markMessageRead } from "@/api/messages";
import type { PageData } from "@/api/types";
import { Pagination } from "@/components/Pagination";
import { PermissionButton } from "@/components/PermissionButton";
import { ListRowActions } from "@/components/RowActionOverflow";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { useWebSocket } from "@/hooks/useWebSocket";
import type { InAppMessageItem, MessageChannel } from "@/types/messages";
import {
  loadPersistedListSettings,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const messageListSettingsKey = "messages.table.settings";
const defaultVisibleColumnKeys = ["status", "module", "severity", "channel", "target", "title", "content", "traceId", "createdAt", "actions"];
const messageTableColumns: TableSettingsColumn[] = [
  { key: "status", label: "状态" },
  { key: "module", label: "模块" },
  { key: "severity", label: "级别" },
  { key: "event", label: "事件" },
  { key: "channel", label: "频道" },
  { key: "target", label: "目标" },
  { key: "title", label: "标题" },
  { key: "content", label: "内容" },
  { key: "traceId", label: "traceId" },
  { key: "createdAt", label: "创建时间" },
  { key: "actions", label: "操作", required: true },
];
const channelOptions: Array<{ value: MessageChannel; label: string }> = [
  { value: "broadcast", label: "广播" },
  { value: "user", label: "用户" },
  { value: "role", label: "角色" },
  { value: "department", label: "部门" },
];

interface MessageFilterState {
  keyword: string;
  channel: "" | MessageChannel;
  read: "" | "true" | "false";
  module: string;
  severity: string;
}

interface MessageFormState {
  channel: MessageChannel;
  target: string;
  title: string;
  content: string;
  module: string;
  severity: string;
  event: string;
  resourceType: string;
  resourceId: string;
  dataJSON: string;
}

function defaultFilter(): MessageFilterState {
  return { keyword: "", channel: "", read: "", module: "", severity: "" };
}

function defaultForm(): MessageFormState {
  return {
    channel: "broadcast",
    target: "",
    title: "",
    content: "",
    module: "system",
    severity: "info",
    event: "manual.message.created",
    resourceType: "",
    resourceId: "",
    dataJSON: "{}",
  };
}

export function MessagesPage() {
  const [filter, setFilter] = useState<MessageFilterState>(() => defaultFilter());
  const [appliedFilter, setAppliedFilter] = useState<MessageFilterState>(() => defaultFilter());
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [data, setData] = useState<PageData<InAppMessageItem>>({ list: [], total: 0, page: 1, pageSize: defaultPageSize });
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [form, setForm] = useState<MessageFormState>(() => defaultForm());
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [visibleColumnKeys, setVisibleColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(messageListSettingsKey);
    const defaults = sanitizeVisibleColumnKeys(defaultVisibleColumnKeys, messageTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, messageTableColumns);
  });
  const { messages: realtimeMessages } = useWebSocket(true);

  async function loadMessages() {
    setLoading(true);
    try {
      const result = await listMessages({
        page,
        pageSize,
        keyword: appliedFilter.keyword,
        channel: appliedFilter.channel,
        read: appliedFilter.read,
        module: appliedFilter.module,
        severity: appliedFilter.severity,
      });
      setData(result);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadMessages();
  }, [page, pageSize, appliedFilter]);

  useEffect(() => {
    if (realtimeMessages.length === 0) return;
    void loadMessages();
  }, [realtimeMessages.length]);

  useEffect(() => {
    savePersistedListSettings(messageListSettingsKey, {
      visibleColumnKeys,
    });
  }, [visibleColumnKeys]);

  const totalPages = useMemo(() => Math.max(1, Math.ceil(data.total / pageSize)), [data.total, pageSize]);

  function applyFilter(event: FormEvent) {
    event.preventDefault();
    setPage(1);
    setAppliedFilter(filter);
  }

  function resetFilter() {
    const next = defaultFilter();
    setFilter(next);
    setAppliedFilter(next);
    setPage(1);
  }

  async function submitMessage(event: FormEvent) {
    event.preventDefault();
    let parsedData: Record<string, unknown> = {};
    try {
      parsedData = JSON.parse(form.dataJSON || "{}") as Record<string, unknown>;
    } catch {
      showToast("扩展数据 JSON 格式错误");
      return;
    }
    await createMessage({
      channel: form.channel,
      target: form.channel === "broadcast" ? "" : form.target,
      title: form.title,
      content: form.content,
      module: form.module,
      severity: form.severity,
      event: form.event,
      resourceType: form.resourceType,
      resourceId: form.resourceId,
      data: parsedData,
    });
    showToast("消息已发送");
    setDrawerOpen(false);
    setForm(defaultForm());
    setPage(1);
    await loadMessages();
  }

  async function handleMarkRead(messageId: number) {
    await markMessageRead(messageId);
    showToast("消息已标记为已读");
    await loadMessages();
  }

  function toggleVisibleColumn(columnKey: string) {
    const column = messageTableColumns.find((item) => item.key === columnKey);
    if (!column || column.required) return;
    setVisibleColumnKeys((current) => {
      const next = current.includes(columnKey)
        ? current.filter((key) => key !== columnKey)
        : [...current, columnKey];
      return sanitizeVisibleColumnKeys(next, messageTableColumns);
    });
  }

  function moveVisibleColumn(columnKey: string, direction: "up" | "down") {
    setVisibleColumnKeys((current) => moveColumnKey(current, columnKey, direction));
  }

  function renderCell(message: InAppMessageItem, key: string) {
    switch (key) {
      case "status":
        return <span className={message.read ? "message-status-read" : "message-status-unread"}>{message.read ? "已读" : "未读"}</span>;
      case "module":
        return message.module || "system";
      case "severity":
        return <span className={`message-severity message-severity-${message.severity || "info"}`}>{message.severity || "info"}</span>;
      case "event":
        return message.event || "-";
      case "channel":
        return channelLabel(message.channel);
      case "target":
        return message.target || "全员";
      case "title":
        return message.title || "消息";
      case "content":
        return <span className="message-content-cell">{message.content}</span>;
      case "traceId":
        return <code>{message.traceId}</code>;
      case "createdAt":
        return formatTime(message.createdAt);
      case "actions":
        return (
          <div className="rbac-row-actions">
            <ListRowActions
              title="消息更多操作"
              actions={[
                {
                  key: `${message.id}-mark-read`,
                  label: "标记已读",
                  permissionKey: "button.messages.message.mark_read",
                  className: "btn ghost cursor-pointer",
                  disabled: message.read,
                  onClick: () => void handleMarkRead(message.id),
                },
              ]}
            />
          </div>
        );
      default:
        return null;
    }
  }

  return (
    <section className="page">
      <div className="rbac-header-row">
        <h2>站内消息</h2>
        <PermissionButton
          permissionKey="button.messages.message.create"
          className="btn primary"
          onClick={() => {
            setForm(defaultForm());
            setDrawerOpen(true);
          }}
        >
          创建消息
        </PermissionButton>
      </div>

      <article className="card rbac-module-card rbac-compact-card cloud-module-card">
        <div className="rbac-module-header">
          <div>
            <h3>消息列表</h3>
            <p className="muted">共 {data.total} 条，当前页 {data.list.length} 条</p>
          </div>
        </div>
        <form className="cloud-filter-bar" onSubmit={applyFilter}>
          <input
            className="cloud-filter-control cloud-filter-keyword"
            placeholder="搜索标题、内容、traceId"
            value={filter.keyword}
            onChange={(event) => setFilter((prev) => ({ ...prev, keyword: event.target.value }))}
          />
          <select
            className="cloud-filter-control"
            value={filter.channel}
            onChange={(event) => setFilter((prev) => ({ ...prev, channel: event.target.value as MessageFilterState["channel"] }))}
          >
            <option value="">全部频道</option>
            {channelOptions.map((item) => (
              <option key={item.value} value={item.value}>{item.label}</option>
            ))}
          </select>
          <select
            className="cloud-filter-control"
            value={filter.read}
            onChange={(event) => setFilter((prev) => ({ ...prev, read: event.target.value as MessageFilterState["read"] }))}
          >
            <option value="">全部状态</option>
            <option value="false">未读</option>
            <option value="true">已读</option>
          </select>
          <select
            className="cloud-filter-control"
            value={filter.module}
            onChange={(event) => setFilter((prev) => ({ ...prev, module: event.target.value }))}
          >
            <option value="">全部模块</option>
            {["tasks", "cloud", "cmdb", "docker", "middleware", "tickets", "events", "kubernetes", "observability", "aiops", "system"].map((item) => (
              <option key={item} value={item}>{item}</option>
            ))}
          </select>
          <select
            className="cloud-filter-control"
            value={filter.severity}
            onChange={(event) => setFilter((prev) => ({ ...prev, severity: event.target.value }))}
          >
            <option value="">全部级别</option>
            {["info", "success", "warning", "error", "critical"].map((item) => (
              <option key={item} value={item}>{item}</option>
            ))}
          </select>
          <div className="cloud-filter-actions">
            <button className="btn primary" type="submit">搜索</button>
            <button className="btn ghost" type="button" onClick={resetFilter}>重置</button>
          </div>
        </form>

        <div className="rbac-table-wrapper">
          <table className="rbac-table">
            <thead>
              <tr>
                {visibleColumnKeys.map((key) => (
                  <th key={key}>
                    {key === "actions" ? (
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button
                          className="table-settings-trigger cursor-pointer"
                          type="button"
                          onClick={() => setSettingsOpen(true)}
                          aria-label="站内消息列表字段设置"
                        >
                          ⚙️
                        </button>
                      </div>
                    ) : (
                      columnLabel(key)
                    )}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.list.length === 0 && (
                <tr>
                  <td colSpan={visibleColumnKeys.length}>{loading ? "加载中..." : "暂无消息"}</td>
                </tr>
              )}
              {data.list.map((message) => (
                <tr key={message.id} className={message.read ? "" : "message-row-unread"}>
                  {visibleColumnKeys.map((key) => <td key={key}>{renderCell(message, key)}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <Pagination
          total={data.total}
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          pageSizeOptions={pageSizeOptions}
          onPageChange={setPage}
          onPageSizeChange={(nextPageSize) => {
            setPageSize(nextPageSize);
            setPage(1);
          }}
        />
      </article>

      {drawerOpen && (
        <div className="rbac-drawer-mask">
          <aside className="rbac-drawer">
            <header className="rbac-drawer-header">
              <h3>创建消息</h3>
              <button className="btn ghost" onClick={() => setDrawerOpen(false)}>关闭</button>
            </header>
            <form className="rbac-drawer-body message-create-form" onSubmit={submitMessage}>
              <section className="message-form-section">
                <div className="message-form-section-title">
                  <h4>投递范围</h4>
                  <p className="muted">选择消息发送频道；广播消息会投递给所有登录用户。</p>
                </div>
                <div className="message-form-grid">
                  <label className="message-form-field">
                    <span>频道</span>
                    <select
                      value={form.channel}
                      onChange={(event) => setForm((prev) => ({ ...prev, channel: event.target.value as MessageChannel }))}
                    >
                      {channelOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                    </select>
                  </label>
                  {form.channel !== "broadcast" && (
                    <label className="message-form-field">
                      <span>目标</span>
                      <input
                        value={form.target}
                        onChange={(event) => setForm((prev) => ({ ...prev, target: event.target.value }))}
                        placeholder={targetPlaceholder(form.channel)}
                        required
                      />
                    </label>
                  )}
                </div>
              </section>

              <section className="message-form-section">
                <div className="message-form-section-title">
                  <h4>消息内容</h4>
                  <p className="muted">标题用于列表识别，正文支持较长通知内容。</p>
                </div>
                <div className="message-form-grid">
                  <label className="message-form-field message-form-field-wide">
                    <span>标题</span>
                    <input
                      value={form.title}
                      placeholder="例如：生产发布窗口提醒"
                      onChange={(event) => setForm((prev) => ({ ...prev, title: event.target.value }))}
                    />
                  </label>
                  <label className="message-form-field message-form-field-wide">
                    <span>内容</span>
                    <textarea
                      className="message-content-editor"
                      required
                      value={form.content}
                      placeholder="请输入消息正文..."
                      onChange={(event) => setForm((prev) => ({ ...prev, content: event.target.value }))}
                    />
                  </label>
                </div>
              </section>

              <section className="message-form-section">
                <div className="message-form-section-title">
                  <h4>扩展数据</h4>
                  <p className="muted">可放入 traceId、跳转链接、业务上下文等机器可读字段。</p>
                </div>
                <div className="message-form-grid">
                  <label className="message-form-field">
                    <span>模块</span>
                    <select value={form.module} onChange={(event) => setForm((prev) => ({ ...prev, module: event.target.value }))}>
                      {["system", "tasks", "cloud", "cmdb", "docker", "middleware", "tickets", "events", "kubernetes", "observability", "aiops"].map((item) => (
                        <option key={item} value={item}>{item}</option>
                      ))}
                    </select>
                  </label>
                  <label className="message-form-field">
                    <span>级别</span>
                    <select value={form.severity} onChange={(event) => setForm((prev) => ({ ...prev, severity: event.target.value }))}>
                      {["info", "success", "warning", "error", "critical"].map((item) => (
                        <option key={item} value={item}>{item}</option>
                      ))}
                    </select>
                  </label>
                  <label className="message-form-field">
                    <span>事件</span>
                    <input value={form.event} placeholder="module.event.name" onChange={(event) => setForm((prev) => ({ ...prev, event: event.target.value }))} />
                  </label>
                  <label className="message-form-field">
                    <span>资源类型</span>
                    <input value={form.resourceType} placeholder="task / cloudAccount / dockerHost" onChange={(event) => setForm((prev) => ({ ...prev, resourceType: event.target.value }))} />
                  </label>
                  <label className="message-form-field">
                    <span>资源 ID</span>
                    <input value={form.resourceId} placeholder="可选" onChange={(event) => setForm((prev) => ({ ...prev, resourceId: event.target.value }))} />
                  </label>
                </div>
                <label className="message-form-field">
                  <span>扩展数据 JSON</span>
                  <textarea
                    className="message-json-editor"
                    value={form.dataJSON}
                    onChange={(event) => setForm((prev) => ({ ...prev, dataJSON: event.target.value }))}
                  />
                </label>
              </section>

              <div className="message-form-actions">
                <button className="btn primary cursor-pointer" type="submit">发送</button>
                <button className="btn ghost cursor-pointer" type="button" onClick={() => setDrawerOpen(false)}>取消</button>
              </div>
            </form>
          </aside>
        </div>
      )}

      <TableSettingsModal
        open={settingsOpen}
        title="消息列表字段"
        columns={messageTableColumns}
        visibleColumnKeys={visibleColumnKeys}
        onClose={() => setSettingsOpen(false)}
        onToggleColumn={toggleVisibleColumn}
        onMoveColumn={moveVisibleColumn}
        onReset={() => setVisibleColumnKeys(sanitizeVisibleColumnKeys(defaultVisibleColumnKeys, messageTableColumns))}
      />
    </section>
  );
}

function columnLabel(key: string) {
  return messageTableColumns.find((item) => item.key === key)?.label ?? key;
}

function channelLabel(channel: string) {
  return channelOptions.find((item) => item.value === channel)?.label ?? channel;
}

function targetPlaceholder(channel: MessageChannel) {
  if (channel === "user") return "用户名或用户ID";
  if (channel === "role") return "角色名称";
  return "部门ID或部门名称";
}

function formatTime(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
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
