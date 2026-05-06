import { FormEvent, useEffect, useMemo, useState } from "react";

import { createMessage, listMessages, markMessageRead } from "@/api/messages";
import type { PageData } from "@/api/types";
import { PermissionButton } from "@/components/PermissionButton";
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
const defaultVisibleColumnKeys = ["status", "channel", "target", "title", "content", "traceId", "createdAt", "actions"];
const messageTableColumns: TableSettingsColumn[] = [
  { key: "status", label: "状态" },
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
}

interface MessageFormState {
  channel: MessageChannel;
  target: string;
  title: string;
  content: string;
  dataJSON: string;
}

function defaultFilter(): MessageFilterState {
  return { keyword: "", channel: "", read: "" };
}

function defaultForm(): MessageFormState {
  return {
    channel: "broadcast",
    target: "",
    title: "",
    content: "",
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

  function renderCell(message: InAppMessageItem, key: string) {
    switch (key) {
      case "status":
        return <span className={message.read ? "message-status-read" : "message-status-unread"}>{message.read ? "已读" : "未读"}</span>;
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
            <PermissionButton
              permissionKey="button.messages.message.mark_read"
              className="btn ghost"
              disabled={message.read}
              onClick={() => void handleMarkRead(message.id)}
            >
              标记已读
            </PermissionButton>
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
                        <button className="table-settings-trigger cursor-pointer" type="button" onClick={() => setSettingsOpen(true)}>
                          列
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

        <div className="rbac-pagination">
          <div className="rbac-pagination-group">
            <button className="btn ghost" disabled={page <= 1} onClick={() => setPage((prev) => Math.max(1, prev - 1))}>上一页</button>
            <span className="rbac-pagination-text">{page} / {totalPages}</span>
            <button className="btn ghost" disabled={page >= totalPages} onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}>下一页</button>
          </div>
          <div className="rbac-pagination-group">
            <span className="muted">每页</span>
            <select
              className="rbac-pagination-select"
              value={pageSize}
              onChange={(event) => {
                setPageSize(Number(event.target.value));
                setPage(1);
              }}
            >
              {pageSizeOptions.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
          </div>
        </div>
      </article>

      {drawerOpen && (
        <div className="rbac-drawer-mask">
          <aside className="rbac-drawer">
            <header className="rbac-drawer-header">
              <h3>创建消息</h3>
              <button className="btn ghost" onClick={() => setDrawerOpen(false)}>关闭</button>
            </header>
            <form className="rbac-drawer-body form-grid" onSubmit={submitMessage}>
              <label>
                频道
                <select
                  value={form.channel}
                  onChange={(event) => setForm((prev) => ({ ...prev, channel: event.target.value as MessageChannel }))}
                >
                  {channelOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                </select>
              </label>
              {form.channel !== "broadcast" && (
                <label>
                  目标
                  <input
                    value={form.target}
                    onChange={(event) => setForm((prev) => ({ ...prev, target: event.target.value }))}
                    placeholder={targetPlaceholder(form.channel)}
                    required
                  />
                </label>
              )}
              <label>
                标题
                <input value={form.title} onChange={(event) => setForm((prev) => ({ ...prev, title: event.target.value }))} />
              </label>
              <label>
                内容
                <textarea required value={form.content} onChange={(event) => setForm((prev) => ({ ...prev, content: event.target.value }))} />
              </label>
              <label>
                扩展数据 JSON
                <textarea value={form.dataJSON} onChange={(event) => setForm((prev) => ({ ...prev, dataJSON: event.target.value }))} />
              </label>
              <div className="rbac-row-actions">
                <button className="btn primary" type="submit">发送</button>
                <button className="btn ghost" type="button" onClick={() => setDrawerOpen(false)}>取消</button>
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
