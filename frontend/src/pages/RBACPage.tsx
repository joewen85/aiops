import { FormEvent, useEffect, useMemo, useState } from "react";

import {
  bindRolePermissions,
  createPermission,
  createRole,
  deletePermission,
  deleteRole,
  getPermission,
  getRole,
  getRolePermissions,
  listPermissions,
  listRoles,
  updatePermission,
  updateRole,
} from "@/api/rbac";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { FieldFilterPopover } from "@/components/FieldFilterPopover";
import { PermissionButton } from "@/components/PermissionButton";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type { PermissionItem, PermissionType, RoleItem } from "@/types/rbac";
import {
  loadPersistedListSettings,
  sanitizeMultiFilterValues,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const typeOptions: PermissionType[] = ["api", "menu", "button"];
const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const ROLE_LIST_SETTINGS_KEY = "rbac.roles.table.settings";
const PERMISSION_LIST_SETTINGS_KEY = "rbac.permissions.table.settings";
const defaultRoleVisibleColumnKeys = ["name", "builtIn", "updatedAt", "actions"];
const defaultPermissionVisibleColumnKeys = ["name", "type", "resourceAction", "actions"];
const roleBuiltInFilterOptions = [
  { value: "true", label: "内置角色" },
  { value: "false", label: "自定义角色" },
] as const;
const permissionTypeFilterOptions = [
  { value: "api", label: "API" },
  { value: "menu", label: "菜单" },
  { value: "button", label: "按钮" },
] as const;
const roleBuiltInFilterValues = roleBuiltInFilterOptions.map((item) => item.value);
const permissionTypeFilterValues = permissionTypeFilterOptions.map((item) => item.value);
const permissionTypeOrder: Record<PermissionType, number> = {
  menu: 1,
  button: 2,
  api: 3,
};

const roleTableColumns: TableSettingsColumn[] = [
  { key: "name", label: "角色名" },
  { key: "description", label: "描述" },
  { key: "builtIn", label: "内置" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];

const permissionTableColumns: TableSettingsColumn[] = [
  { key: "name", label: "权限名" },
  { key: "type", label: "类型" },
  { key: "resourceAction", label: "资源/动作" },
  { key: "key", label: "Key" },
  { key: "actions", label: "操作", required: true },
];

type DrawerState =
  | { type: "closed" }
  | { type: "role-create" }
  | { type: "role-edit"; roleId: number; builtIn: boolean }
  | { type: "role-detail"; roleId: number }
  | { type: "permission-create" }
  | { type: "permission-edit"; permissionId: number }
  | { type: "permission-detail"; permissionId: number };

type TableSettingsTarget = "closed" | "roles" | "permissions";

interface RoleFormState {
  name: string;
  description: string;
}

interface PermissionFormState {
  name: string;
  type: PermissionType;
  key: string;
  resource: string;
  action: string;
  deptScope: string;
  resourceTagScope: string;
  envScope: string;
  description: string;
}

interface RoleSearchState {
  keyword: string;
  builtIn: string;
}

interface PermissionSearchState {
  keyword: string;
  type: string;
}

function defaultPermissionForm(): PermissionFormState {
  return {
    name: "",
    type: "api",
    key: "",
    resource: "",
    action: "",
    deptScope: "*",
    resourceTagScope: "*",
    envScope: "*",
    description: "",
  };
}

function defaultRoleSearchState(): RoleSearchState {
  return {
    keyword: "",
    builtIn: "",
  };
}

function defaultPermissionSearchState(): PermissionSearchState {
  return {
    keyword: "",
    type: "",
  };
}

function extractPermissionModule(permission: PermissionItem): string {
  const key = (permission.key ?? "").trim();
  if (key) {
    const keyParts = key.split(".").map((item) => item.trim()).filter(Boolean);
    if (keyParts.length >= 2) return keyParts[1];
    if (keyParts.length === 1) return keyParts[0];
  }

  const resource = (permission.resource ?? "").trim();
  if (resource.startsWith("/")) {
    const resourceParts = resource.split("/").map((item) => item.trim()).filter(Boolean);
    const apiIndex = resourceParts.indexOf("api");
    if (apiIndex >= 0) {
      if (resourceParts[apiIndex + 1] === "v1" && resourceParts[apiIndex + 2]) {
        return resourceParts[apiIndex + 2];
      }
      if (resourceParts[apiIndex + 1]) {
        return resourceParts[apiIndex + 1];
      }
    }
    if (resourceParts[0]) return resourceParts[0];
  }

  return permission.type || "unknown";
}

function normalizeKeyword(value: string): string {
  return value.trim().toLowerCase();
}

function containsKeyword(value: string, keyword: string): boolean {
  if (!keyword) return true;
  return value.toLowerCase().includes(keyword);
}

function formatModuleLabel(moduleKey: string): string {
  const cleaned = moduleKey.trim();
  if (!cleaned) return "未归类";
  if (cleaned.toLowerCase() === "rbac") return "RBAC";
  if (cleaned.toLowerCase() === "aiops") return "AIOps";
  return cleaned;
}

export function RBACPage() {
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [permissions, setPermissions] = useState<PermissionItem[]>([]);
  const [roleTotal, setRoleTotal] = useState(0);
  const [permissionTotal, setPermissionTotal] = useState(0);
  const [rolePage, setRolePage] = useState(1);
  const [permissionPage, setPermissionPage] = useState(1);
  const [rolePageSize, setRolePageSize] = useState(defaultPageSize);
  const [permissionPageSize, setPermissionPageSize] = useState(defaultPageSize);
  const [roleFilters, setRoleFilters] = useState<RoleSearchState>(defaultRoleSearchState);
  const [roleQuery, setRoleQuery] = useState<RoleSearchState>(defaultRoleSearchState);
  const [permissionFilters, setPermissionFilters] = useState<PermissionSearchState>(defaultPermissionSearchState);
  const [permissionQuery, setPermissionQuery] = useState<PermissionSearchState>(defaultPermissionSearchState);
  const [roleJumpPageInput, setRoleJumpPageInput] = useState("1");
  const [permissionJumpPageInput, setPermissionJumpPageInput] = useState("1");
  const [bindingPermissions, setBindingPermissions] = useState<PermissionItem[]>([]);
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<number[]>([]);
  const [bindingKeyword, setBindingKeyword] = useState("");
  const [bindingSelectedOnly, setBindingSelectedOnly] = useState(false);
  const [drawer, setDrawer] = useState<DrawerState>({ type: "closed" });
  const [roleForm, setRoleForm] = useState<RoleFormState>({ name: "", description: "" });
  const [permissionForm, setPermissionForm] = useState<PermissionFormState>(defaultPermissionForm);
  const [roleDetail, setRoleDetail] = useState<RoleItem | null>(null);
  const [permissionDetail, setPermissionDetail] = useState<PermissionItem | null>(null);
  const [roleListLoading, setRoleListLoading] = useState(false);
  const [permissionListLoading, setPermissionListLoading] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteRoleTarget, setDeleteRoleTarget] = useState<RoleItem | null>(null);
  const [deletePermissionTarget, setDeletePermissionTarget] = useState<PermissionItem | null>(null);
  const [bindingSaving, setBindingSaving] = useState(false);
  const [tableSettingsTarget, setTableSettingsTarget] = useState<TableSettingsTarget>("closed");
  const [visibleRoleColumnKeys, setVisibleRoleColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(ROLE_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultRoleVisibleColumnKeys, roleTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, roleTableColumns);
  });
  const [visiblePermissionColumnKeys, setVisiblePermissionColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(PERMISSION_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultPermissionVisibleColumnKeys, permissionTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, permissionTableColumns);
  });
  const [selectedRoleBuiltInValues, setSelectedRoleBuiltInValues] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(ROLE_LIST_SETTINGS_KEY);
    return sanitizeMultiFilterValues(
      persisted?.roleBuiltInValues,
      roleBuiltInFilterValues,
      roleBuiltInFilterValues,
    );
  });
  const [selectedPermissionTypeValues, setSelectedPermissionTypeValues] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(PERMISSION_LIST_SETTINGS_KEY);
    return sanitizeMultiFilterValues(
      persisted?.permissionTypeValues,
      permissionTypeFilterValues,
      permissionTypeFilterValues,
    );
  });

  useEffect(() => {
    void loadRolePage(rolePage, rolePageSize, roleQuery);
  }, [rolePage, rolePageSize, roleQuery]);

  useEffect(() => {
    void loadPermissionPage(permissionPage, permissionPageSize, permissionQuery);
  }, [permissionPage, permissionPageSize, permissionQuery]);

  useEffect(() => {
    setRoleJumpPageInput(String(rolePage));
  }, [rolePage]);

  useEffect(() => {
    setPermissionJumpPageInput(String(permissionPage));
  }, [permissionPage]);

  useEffect(() => {
    savePersistedListSettings(ROLE_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleRoleColumnKeys,
      roleBuiltInValues: selectedRoleBuiltInValues,
    });
  }, [selectedRoleBuiltInValues, visibleRoleColumnKeys]);

  useEffect(() => {
    savePersistedListSettings(PERMISSION_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visiblePermissionColumnKeys,
      permissionTypeValues: selectedPermissionTypeValues,
    });
  }, [selectedPermissionTypeValues, visiblePermissionColumnKeys]);

  useEffect(() => {
    if (drawer.type === "role-detail") {
      void loadRolePermissionDetail(drawer.roleId);
      return;
    }
    if (drawer.type === "permission-detail") {
      void loadPermissionDetail(drawer.permissionId);
      return;
    }
    setRoleDetail(null);
    setPermissionDetail(null);
    setBindingPermissions([]);
    setSelectedPermissionIds([]);
    setBindingKeyword("");
    setBindingSelectedOnly(false);
  }, [drawer]);

  const filteredBindingPermissions = useMemo(() => {
    const keyword = normalizeKeyword(bindingKeyword);
    return bindingPermissions.filter((permission) => {
      if (bindingSelectedOnly && !selectedPermissionIds.includes(permission.id)) {
        return false;
      }
      if (!keyword) return true;
      const moduleKey = extractPermissionModule(permission);
      const haystack = [
        permission.name,
        permission.key,
        permission.resource,
        permission.action,
        permission.description,
        permission.type,
        moduleKey,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(keyword);
    });
  }, [bindingKeyword, bindingPermissions, bindingSelectedOnly, selectedPermissionIds]);

  const groupedPermissions = useMemo(() => {
    const groupedMap = new Map<string, PermissionItem[]>();
    for (const permission of filteredBindingPermissions) {
      const moduleKey = extractPermissionModule(permission);
      const current = groupedMap.get(moduleKey) ?? [];
      current.push(permission);
      groupedMap.set(moduleKey, current);
    }
    return Array.from(groupedMap.entries())
      .map(([moduleKey, list]) => ({
        moduleKey,
        list: list.slice().sort((left, right) => {
          const typeCompare = permissionTypeOrder[left.type] - permissionTypeOrder[right.type];
          if (typeCompare !== 0) return typeCompare;
          return left.name.localeCompare(right.name, "zh-CN");
        }),
      }))
      .sort((left, right) => left.moduleKey.localeCompare(right.moduleKey, "zh-CN"));
  }, [filteredBindingPermissions]);

  async function loadRolePage(page: number, pageSize: number, query: RoleSearchState = roleQuery) {
    setRoleListLoading(true);
    try {
      const data = await listRoles(page, pageSize, query);
      const pages = totalPages(data.total, pageSize);
      if (page > pages) {
        setRolePage(pages);
        return;
      }
      setRoles(data.list);
      setRoleTotal(data.total);
    } catch {
      showToast("角色列表加载失败");
    } finally {
      setRoleListLoading(false);
    }
  }

  async function loadPermissionPage(page: number, pageSize: number, query: PermissionSearchState = permissionQuery) {
    setPermissionListLoading(true);
    try {
      const data = await listPermissions(page, pageSize, query);
      const pages = totalPages(data.total, pageSize);
      if (page > pages) {
        setPermissionPage(pages);
        return;
      }
      setPermissions(data.list);
      setPermissionTotal(data.total);
    } catch {
      showToast("权限列表加载失败");
    } finally {
      setPermissionListLoading(false);
    }
  }

  async function loadRolePermissionDetail(roleId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getRolePermissions(roleId);
      setBindingPermissions(detail.permissions ?? []);
      setSelectedPermissionIds((detail.permissionIds ?? []).map(Number));
      try {
        const role = await getRole(roleId);
        setRoleDetail(role);
      } catch {
        const fallbackRole = roles.find((item) => item.id === roleId) ?? null;
        setRoleDetail(fallbackRole);
      }
    } catch {
      showToast("角色详情加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  async function loadPermissionDetail(permissionId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getPermission(permissionId);
      setPermissionDetail(detail);
    } catch {
      showToast("权限详情加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  function togglePermission(permissionId: number) {
    setSelectedPermissionIds((prev) => (
      prev.includes(permissionId)
        ? prev.filter((item) => item !== permissionId)
        : [...prev, permissionId]
    ));
  }

  function toggleModulePermissions(moduleKey: string) {
    const modulePermissionIds = bindingPermissions
      .filter((permission) => extractPermissionModule(permission) === moduleKey)
      .map((permission) => permission.id);
    if (modulePermissionIds.length === 0) return;

    setSelectedPermissionIds((prev) => {
      const selectedSet = new Set(prev);
      const allSelected = modulePermissionIds.every((permissionId) => selectedSet.has(permissionId));
      if (allSelected) {
        return prev.filter((permissionId) => !modulePermissionIds.includes(permissionId));
      }
      const next = new Set(prev);
      modulePermissionIds.forEach((permissionId) => next.add(permissionId));
      return Array.from(next);
    });
  }

  function openRoleCreateDrawer() {
    setRoleForm({ name: "", description: "" });
    setDrawer({ type: "role-create" });
  }

  function openRoleEditDrawer(role: RoleItem) {
    setRoleForm({ name: role.name, description: role.description ?? "" });
    setDrawer({ type: "role-edit", roleId: role.id, builtIn: role.builtIn });
  }

  function openRoleDetailDrawer(roleId: number) {
    setDrawer({ type: "role-detail", roleId });
  }

  function openPermissionCreateDrawer() {
    setPermissionForm(defaultPermissionForm());
    setDrawer({ type: "permission-create" });
  }

  function openPermissionEditDrawer(permission: PermissionItem) {
    setPermissionForm({
      name: permission.name,
      type: permission.type,
      key: permission.key ?? "",
      resource: permission.resource,
      action: permission.action,
      deptScope: permission.deptScope ?? "*",
      resourceTagScope: permission.resourceTagScope ?? "*",
      envScope: permission.envScope ?? "*",
      description: permission.description ?? "",
    });
    setDrawer({ type: "permission-edit", permissionId: permission.id });
  }

  function openPermissionDetailDrawer(permissionId: number) {
    setDrawer({ type: "permission-detail", permissionId });
  }

  function openRoleTableSettings() {
    setTableSettingsTarget("roles");
  }

  function openPermissionTableSettings() {
    setTableSettingsTarget("permissions");
  }

  function handleRoleFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRolePage(1);
    setRoleQuery({ ...roleFilters });
  }

  function handlePermissionFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPermissionPage(1);
    setPermissionQuery({ ...permissionFilters });
  }

  function handleRoleFilterReset() {
    const defaults = defaultRoleSearchState();
    setRoleFilters(defaults);
    setRoleQuery(defaults);
    setRolePage(1);
  }

  function handlePermissionFilterReset() {
    const defaults = defaultPermissionSearchState();
    setPermissionFilters(defaults);
    setPermissionQuery(defaults);
    setPermissionPage(1);
  }

  function toggleRoleVisibleColumn(columnKey: string) {
    const column = roleTableColumns.find((item) => item.key === columnKey);
    if (!column || column.required) return;
    setVisibleRoleColumnKeys((prev) => (
      prev.includes(columnKey)
        ? prev.filter((key) => key !== columnKey)
        : [...prev, columnKey]
    ));
  }

  function togglePermissionVisibleColumn(columnKey: string) {
    const column = permissionTableColumns.find((item) => item.key === columnKey);
    if (!column || column.required) return;
    setVisiblePermissionColumnKeys((prev) => (
      prev.includes(columnKey)
        ? prev.filter((key) => key !== columnKey)
        : [...prev, columnKey]
    ));
  }

  async function handleSaveBindings(roleId: number) {
    if (!roleId) {
      showToast("请选择角色");
      return;
    }
    setBindingSaving(true);
    try {
      await bindRolePermissions(roleId, selectedPermissionIds);
      showToast("角色权限绑定已保存");
      await loadRolePermissionDetail(roleId);
    } catch {
      showToast("角色权限绑定失败");
    } finally {
      setBindingSaving(false);
    }
  }

  async function handleSubmitRole(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = roleForm.name.trim();
    const description = roleForm.description.trim();
    if (!name) {
      showToast("角色名不能为空");
      return;
    }
    setSubmitting(true);
    try {
      if (drawer.type === "role-create") {
        await createRole({ name, description });
        showToast("角色创建成功");
        if (rolePage === 1) {
          await loadRolePage(1, rolePageSize);
        } else {
          setRolePage(1);
        }
      } else if (drawer.type === "role-edit") {
        await updateRole(drawer.roleId, { name, description });
        showToast("角色更新成功");
        await loadRolePage(rolePage, rolePageSize);
      }
      setDrawer({ type: "closed" });
    } catch {
      showToast("角色保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSubmitPermission(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = permissionForm.name.trim();
    const resource = permissionForm.resource.trim();
    const action = permissionForm.action.trim();
    if (!name || !resource || !action) {
      showToast("权限名、资源、动作必填");
      return;
    }
    setSubmitting(true);
    try {
      const payload: Partial<PermissionItem> = {
        name,
        type: permissionForm.type,
        key: permissionForm.key.trim(),
        resource,
        action,
        deptScope: permissionForm.deptScope.trim() || "*",
        resourceTagScope: permissionForm.resourceTagScope.trim() || "*",
        envScope: permissionForm.envScope.trim() || "*",
        description: permissionForm.description.trim(),
      };
      if (drawer.type === "permission-create") {
        await createPermission(payload);
        showToast("权限创建成功");
        if (permissionPage === 1) {
          await loadPermissionPage(1, permissionPageSize);
        } else {
          setPermissionPage(1);
        }
      } else if (drawer.type === "permission-edit") {
        await updatePermission(drawer.permissionId, payload);
        showToast("权限更新成功");
        await loadPermissionPage(permissionPage, permissionPageSize);
      }
      setDrawer({ type: "closed" });
    } catch {
      showToast("权限保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDeleteRole(role: RoleItem) {
    if (role.builtIn) {
      showToast("内置角色不允许删除");
      return;
    }
    try {
      await deleteRole(role.id);
      showToast("角色已删除");
      if (drawer.type === "role-detail" && drawer.roleId === role.id) {
        setDrawer({ type: "closed" });
      }
      if (roles.length === 1 && rolePage > 1) {
        setRolePage(rolePage - 1);
      } else {
        await loadRolePage(rolePage, rolePageSize);
      }
    } catch {
      showToast("角色删除失败");
    }
  }

  async function handleDeletePermission(permission: PermissionItem) {
    try {
      await deletePermission(permission.id);
      showToast("权限已删除");
      if (drawer.type === "permission-detail" && drawer.permissionId === permission.id) {
        setDrawer({ type: "closed" });
      }
      if (permissions.length === 1 && permissionPage > 1) {
        setPermissionPage(permissionPage - 1);
      } else {
        await loadPermissionPage(permissionPage, permissionPageSize);
      }
    } catch {
      showToast("权限删除失败");
    }
  }

  function requestDeleteRole(role: RoleItem) {
    if (role.builtIn) {
      showToast("内置角色不允许删除");
      return;
    }
    setDeleteRoleTarget(role);
  }

  function requestDeletePermission(permission: PermissionItem) {
    setDeletePermissionTarget(permission);
  }

  async function confirmDeleteRole() {
    if (!deleteRoleTarget) return;
    setDeleteSubmitting(true);
    await handleDeleteRole(deleteRoleTarget);
    setDeleteRoleTarget(null);
    setDeleteSubmitting(false);
  }

  async function confirmDeletePermission() {
    if (!deletePermissionTarget) return;
    setDeleteSubmitting(true);
    await handleDeletePermission(deletePermissionTarget);
    setDeletePermissionTarget(null);
    setDeleteSubmitting(false);
  }

  function formatDateTime(value?: string): string {
    if (!value) return "-";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "-";
    return date.toLocaleString();
  }

  function drawerTitle(): string {
    switch (drawer.type) {
      case "role-create":
        return "创建角色";
      case "role-edit":
        return "修改角色";
      case "role-detail":
        return "角色详情";
      case "permission-create":
        return "创建权限";
      case "permission-edit":
        return "修改权限";
      case "permission-detail":
        return "权限详情";
      default:
        return "";
    }
  }

  function totalPages(total: number, pageSize: number): number {
    return Math.max(1, Math.ceil(total / pageSize));
  }

  function clampPage(page: number, total: number, pageSize: number): number {
    const pages = totalPages(total, pageSize);
    return Math.min(Math.max(1, page), pages);
  }

  function handleRolePageChange(nextPage: number) {
    const target = clampPage(nextPage, roleTotal, rolePageSize);
    if (target !== rolePage) {
      setRolePage(target);
    }
  }

  function handlePermissionPageChange(nextPage: number) {
    const target = clampPage(nextPage, permissionTotal, permissionPageSize);
    if (target !== permissionPage) {
      setPermissionPage(target);
    }
  }

  function handleRoleJumpSubmit() {
    const target = Number.parseInt(roleJumpPageInput, 10);
    if (Number.isNaN(target)) {
      showToast("请输入正确页码");
      setRoleJumpPageInput(String(rolePage));
      return;
    }
    handleRolePageChange(target);
  }

  function handlePermissionJumpSubmit() {
    const target = Number.parseInt(permissionJumpPageInput, 10);
    if (Number.isNaN(target)) {
      showToast("请输入正确页码");
      setPermissionJumpPageInput(String(permissionPage));
      return;
    }
    handlePermissionPageChange(target);
  }

  function renderPagination(props: {
    page: number;
    total: number;
    pageSize: number;
    jumpValue: string;
    onPageChange: (pageNum: number) => void;
    onPageSizeChange: (pageSize: number) => void;
    onJumpValueChange: (value: string) => void;
    onJumpSubmit: () => void;
  }) {
    const {
      page,
      total,
      pageSize,
      jumpValue,
      onPageChange,
      onPageSizeChange,
      onJumpValueChange,
      onJumpSubmit,
    } = props;
    const pages = totalPages(total, pageSize);
    const prevDisabled = page <= 1;
    const nextDisabled = page >= pages;
    return (
      <div className="rbac-pagination">
        <div className="rbac-pagination-group">
          <span className="muted">共 {total} 条</span>
          <span className="muted">每页</span>
          <select
            className="rbac-pagination-select"
            value={pageSize}
            onChange={(event) => onPageSizeChange(Number(event.target.value))}
          >
            {pageSizeOptions.map((option) => (
              <option key={option} value={option}>{option}</option>
            ))}
          </select>
        </div>
        <div className="rbac-pagination-group">
          <button
            className="btn ghost cursor-pointer"
            type="button"
            onClick={() => onPageChange(page - 1)}
            disabled={prevDisabled}
          >
            上一页
          </button>
          <span className="rbac-pagination-text">
            第 {page} / {pages} 页
          </span>
          <button
            className="btn ghost cursor-pointer"
            type="button"
            onClick={() => onPageChange(page + 1)}
            disabled={nextDisabled}
          >
            下一页
          </button>
          <input
            className="rbac-pagination-input"
            type="number"
            min={1}
            max={pages}
            value={jumpValue}
            onChange={(event) => onJumpValueChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                onJumpSubmit();
              }
            }}
          />
          <button className="btn ghost cursor-pointer" type="button" onClick={onJumpSubmit}>
            跳转
          </button>
        </div>
      </div>
    );
  }

  const drawerVisible = drawer.type !== "closed";
  const showRoleForm = drawer.type === "role-create" || drawer.type === "role-edit";
  const showPermissionForm = drawer.type === "permission-create" || drawer.type === "permission-edit";
  const showRoleDetail = drawer.type === "role-detail";
  const showPermissionDetail = drawer.type === "permission-detail";
  const currentRoleId = drawer.type === "role-detail" ? drawer.roleId : 0;
  const currentRoleBuiltIn = drawer.type === "role-edit" ? drawer.builtIn : false;
  const filteredRoles = useMemo(() => roles.filter((role) => {
    const builtInValue = role.builtIn ? "true" : "false";
    if (!selectedRoleBuiltInValues.includes(builtInValue)) return false;
    if (roleQuery.builtIn && builtInValue !== roleQuery.builtIn) return false;
    const keyword = normalizeKeyword(roleQuery.keyword);
    if (!keyword) return true;
    const roleName = role.name ?? "";
    const roleDescription = role.description ?? "";
    return containsKeyword(roleName, keyword) || containsKeyword(roleDescription, keyword);
  }), [roles, selectedRoleBuiltInValues, roleQuery.builtIn, roleQuery.keyword]);
  const filteredPermissions = useMemo(() => permissions.filter((permission) => {
    if (!selectedPermissionTypeValues.includes(permission.type)) return false;
    if (permissionQuery.type && permission.type !== permissionQuery.type) return false;
    const keyword = normalizeKeyword(permissionQuery.keyword);
    if (!keyword) return true;
    const haystack = [
      permission.name,
      permission.key,
      permission.resource,
      permission.action,
      permission.description,
      permission.type,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return haystack.includes(keyword);
  }), [permissions, selectedPermissionTypeValues, permissionQuery.keyword, permissionQuery.type]);
  const roleRows = roleListLoading ? [] : filteredRoles;
  const permissionRows = permissionListLoading ? [] : filteredPermissions;
  const roleVisibleColumnSet = new Set(visibleRoleColumnKeys);
  const permissionVisibleColumnSet = new Set(visiblePermissionColumnKeys);
  const roleColSpan = Math.max(1, visibleRoleColumnKeys.length);
  const permissionColSpan = Math.max(1, visiblePermissionColumnKeys.length);
  const displayRole = showRoleDetail
    ? (roleDetail ?? roles.find((item) => item.id === currentRoleId) ?? null)
    : null;

  return (
    <section className="page">
      <h2>RBAC / ABAC 权限管理</h2>

      <div className="rbac-module-scroll">
        <div className="rbac-module-grid">
          <article className="card rbac-module-card rbac-compact-card">
          <div className="rbac-module-header">
            <div>
              <h3>角色子模块</h3>
              <p className="muted">角色列表、查看详情、权限绑定、创建、修改、删除</p>
            </div>
            <PermissionButton
              permissionKey="button.rbac.role.create"
              className="btn primary cursor-pointer"
              type="button"
              onClick={openRoleCreateDrawer}
            >
              创建角色
            </PermissionButton>
          </div>

          <form className="cloud-filter-bar" onSubmit={handleRoleFilterSubmit}>
            <input
              className="cloud-filter-control cloud-filter-keyword"
              value={roleFilters.keyword}
              onChange={(event) => setRoleFilters((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词：角色名/描述"
            />
            <select
              className="cloud-filter-control"
              value={roleFilters.builtIn}
              onChange={(event) => setRoleFilters((prev) => ({ ...prev, builtIn: event.target.value }))}
            >
              <option value="">内置状态：全部</option>
              <option value="true">内置角色</option>
              <option value="false">自定义角色</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={roleListLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={handleRoleFilterReset}>重置</button>
            </div>
          </form>

          <div className="rbac-table-wrapper">
            <table className="rbac-table">
              <thead>
                <tr>
                  {roleVisibleColumnSet.has("name") && <th>角色名</th>}
                  {roleVisibleColumnSet.has("description") && <th>描述</th>}
                  {roleVisibleColumnSet.has("builtIn") && (
                    <th>
                      <div className="table-actions-header">
                        <span>内置</span>
                        <FieldFilterPopover
                          ariaLabel="角色内置筛选"
                          options={[...roleBuiltInFilterOptions]}
                          selectedValues={selectedRoleBuiltInValues}
                          onChange={setSelectedRoleBuiltInValues}
                        />
                      </div>
                    </th>
                  )}
                  {roleVisibleColumnSet.has("updatedAt") && <th>更新时间</th>}
                  {roleVisibleColumnSet.has("actions") && (
                    <th>
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button className="table-settings-trigger cursor-pointer" type="button" onClick={openRoleTableSettings} aria-label="角色列表设置">
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {roleRows.length === 0 && (
                  <tr>
                    <td colSpan={roleColSpan} className="muted">{roleListLoading ? "加载中..." : "暂无角色"}</td>
                  </tr>
                )}
                {roleRows.map((role) => (
                  <tr key={role.id}>
                    {roleVisibleColumnSet.has("name") && <td>{role.name}</td>}
                    {roleVisibleColumnSet.has("description") && <td>{role.description || "-"}</td>}
                    {roleVisibleColumnSet.has("builtIn") && <td>{role.builtIn ? "是" : "否"}</td>}
                    {roleVisibleColumnSet.has("updatedAt") && <td>{formatDateTime(role.updatedAt)}</td>}
                    {roleVisibleColumnSet.has("actions") && (
                      <td className="rbac-row-actions">
                        <PermissionButton permissionKey="button.rbac.binding.save" className="btn ghost cursor-pointer" type="button" onClick={() => openRoleDetailDrawer(role.id)}>
                          权限绑定
                        </PermissionButton>
                        <PermissionButton permissionKey="button.rbac.role.detail" className="btn ghost cursor-pointer" type="button" onClick={() => openRoleDetailDrawer(role.id)}>
                          查看详情
                        </PermissionButton>
                        <PermissionButton permissionKey="button.rbac.role.update" className="btn ghost cursor-pointer" type="button" onClick={() => openRoleEditDrawer(role)}>
                          修改
                        </PermissionButton>
                        <PermissionButton
                          permissionKey="button.rbac.role.delete"
                          className="btn ghost cursor-pointer"
                          type="button"
                          onClick={() => requestDeleteRole(role)}
                          disabled={role.builtIn}
                        >
                          删除
                        </PermissionButton>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {renderPagination({
            page: rolePage,
            total: roleTotal,
            pageSize: rolePageSize,
            jumpValue: roleJumpPageInput,
            onPageChange: handleRolePageChange,
            onPageSizeChange: (size) => {
              if (size === rolePageSize) return;
              setRolePageSize(size);
              setRolePage(1);
            },
            onJumpValueChange: setRoleJumpPageInput,
            onJumpSubmit: handleRoleJumpSubmit,
          })}
        </article>

        <article className="card rbac-module-card rbac-compact-card">
          <div className="rbac-module-header">
            <div>
              <h3>权限子模块</h3>
              <p className="muted">权限列表、查看详情、创建、修改、删除</p>
            </div>
            <PermissionButton
              permissionKey="button.rbac.permission.create"
              className="btn primary cursor-pointer"
              type="button"
              onClick={openPermissionCreateDrawer}
            >
              创建权限
            </PermissionButton>
          </div>

          <form className="cloud-filter-bar" onSubmit={handlePermissionFilterSubmit}>
            <input
              className="cloud-filter-control cloud-filter-keyword"
              value={permissionFilters.keyword}
              onChange={(event) => setPermissionFilters((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="关键词：权限名/Key/资源/动作"
            />
            <select
              className="cloud-filter-control"
              value={permissionFilters.type}
              onChange={(event) => setPermissionFilters((prev) => ({ ...prev, type: event.target.value }))}
            >
              <option value="">权限类型：全部</option>
              <option value="api">api</option>
              <option value="menu">menu</option>
              <option value="button">button</option>
            </select>
            <div className="cloud-filter-actions">
              <button className="btn cursor-pointer" type="submit" disabled={permissionListLoading}>查询</button>
              <button className="btn cursor-pointer" type="button" onClick={handlePermissionFilterReset}>重置</button>
            </div>
          </form>

          <div className="rbac-table-wrapper">
            <table className="rbac-table">
              <thead>
                <tr>
                  {permissionVisibleColumnSet.has("name") && <th>权限名</th>}
                  {permissionVisibleColumnSet.has("type") && (
                    <th>
                      <div className="table-actions-header">
                        <span>类型</span>
                        <FieldFilterPopover
                          ariaLabel="权限类型筛选"
                          options={[...permissionTypeFilterOptions]}
                          selectedValues={selectedPermissionTypeValues}
                          onChange={setSelectedPermissionTypeValues}
                        />
                      </div>
                    </th>
                  )}
                  {permissionVisibleColumnSet.has("resourceAction") && <th>资源/动作</th>}
                  {permissionVisibleColumnSet.has("key") && <th>Key</th>}
                  {permissionVisibleColumnSet.has("actions") && (
                    <th>
                      <div className="table-actions-header">
                        <span>操作</span>
                        <button className="table-settings-trigger cursor-pointer" type="button" onClick={openPermissionTableSettings} aria-label="权限列表设置">
                          ⚙️
                        </button>
                      </div>
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {permissionRows.length === 0 && (
                  <tr>
                    <td colSpan={permissionColSpan} className="muted">{permissionListLoading ? "加载中..." : "暂无权限"}</td>
                  </tr>
                )}
                {permissionRows.map((permission) => (
                  <tr key={permission.id}>
                    {permissionVisibleColumnSet.has("name") && <td>{permission.name}</td>}
                    {permissionVisibleColumnSet.has("type") && <td>{permission.type}</td>}
                    {permissionVisibleColumnSet.has("resourceAction") && <td>{permission.resource} [{permission.action}]</td>}
                    {permissionVisibleColumnSet.has("key") && <td>{permission.key || "-"}</td>}
                    {permissionVisibleColumnSet.has("actions") && (
                      <td className="rbac-row-actions">
                        <PermissionButton permissionKey="button.rbac.permission.detail" className="btn ghost cursor-pointer" type="button" onClick={() => openPermissionDetailDrawer(permission.id)}>
                          查看详情
                        </PermissionButton>
                        <PermissionButton permissionKey="button.rbac.permission.update" className="btn ghost cursor-pointer" type="button" onClick={() => openPermissionEditDrawer(permission)}>
                          修改
                        </PermissionButton>
                        <PermissionButton permissionKey="button.rbac.permission.delete" className="btn ghost cursor-pointer" type="button" onClick={() => requestDeletePermission(permission)}>
                          删除
                        </PermissionButton>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {renderPagination({
            page: permissionPage,
            total: permissionTotal,
            pageSize: permissionPageSize,
            jumpValue: permissionJumpPageInput,
            onPageChange: handlePermissionPageChange,
            onPageSizeChange: (size) => {
              if (size === permissionPageSize) return;
              setPermissionPageSize(size);
              setPermissionPage(1);
            },
            onJumpValueChange: setPermissionJumpPageInput,
            onJumpSubmit: handlePermissionJumpSubmit,
          })}
        </article>
        </div>
      </div>

      <TableSettingsModal
        open={tableSettingsTarget === "roles"}
        title="角色列表设置"
        columns={roleTableColumns}
        visibleColumnKeys={visibleRoleColumnKeys}
        onToggleColumn={toggleRoleVisibleColumn}
        onReset={() => {
          setVisibleRoleColumnKeys(sanitizeVisibleColumnKeys(defaultRoleVisibleColumnKeys, roleTableColumns));
        }}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <TableSettingsModal
        open={tableSettingsTarget === "permissions"}
        title="权限列表设置"
        columns={permissionTableColumns}
        visibleColumnKeys={visiblePermissionColumnKeys}
        onToggleColumn={togglePermissionVisibleColumn}
        onReset={() => {
          setVisiblePermissionColumnKeys(sanitizeVisibleColumnKeys(defaultPermissionVisibleColumnKeys, permissionTableColumns));
        }}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <DeleteConfirmModal
        open={deleteRoleTarget !== null}
        title="删除角色确认"
        description={`将删除角色：${deleteRoleTarget?.name || "-"}`}
        confirming={deleteSubmitting}
        onCancel={() => setDeleteRoleTarget(null)}
        onConfirm={() => void confirmDeleteRole()}
      />

      <DeleteConfirmModal
        open={deletePermissionTarget !== null}
        title="删除权限确认"
        description={`将删除权限：${deletePermissionTarget?.name || "-"}`}
        confirming={deleteSubmitting}
        onCancel={() => setDeletePermissionTarget(null)}
        onConfirm={() => void confirmDeletePermission()}
      />

      {drawerVisible && (
        <div className="rbac-drawer-mask" onClick={() => setDrawer({ type: "closed" })}>
          <aside className="rbac-drawer" onClick={(event) => event.stopPropagation()}>
            <header className="rbac-drawer-header">
              <h3>{drawerTitle()}</h3>
              <button className="btn ghost cursor-pointer" type="button" onClick={() => setDrawer({ type: "closed" })}>
                关闭
              </button>
            </header>

            <div className="rbac-drawer-body">
              {drawerLoading && <p className="muted">加载中...</p>}

              {showRoleForm && !drawerLoading && (
                <form className="form-grid" onSubmit={handleSubmitRole}>
                  <label htmlFor="drawer-role-name">角色名</label>
                  <input
                    id="drawer-role-name"
                    value={roleForm.name}
                    onChange={(event) => setRoleForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="ops-engineer"
                    disabled={currentRoleBuiltIn}
                  />
                  {currentRoleBuiltIn && <small className="muted">内置角色不允许修改名称</small>}
                  <label htmlFor="drawer-role-desc">描述</label>
                  <input
                    id="drawer-role-desc"
                    value={roleForm.description}
                    onChange={(event) => setRoleForm((prev) => ({ ...prev, description: event.target.value }))}
                    placeholder="运维工程师角色"
                  />
                  <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>
                    {submitting ? "保存中..." : "保存"}
                  </button>
                </form>
              )}

              {showPermissionForm && !drawerLoading && (
                <form className="form-grid" onSubmit={handleSubmitPermission}>
                  <label htmlFor="drawer-perm-name">权限名</label>
                  <input
                    id="drawer-perm-name"
                    value={permissionForm.name}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="任务列表查看"
                  />
                  <label htmlFor="drawer-perm-type">类型</label>
                  <select
                    id="drawer-perm-type"
                    value={permissionForm.type}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, type: event.target.value as PermissionType }))}
                  >
                    <option value="api">api</option>
                    <option value="menu">menu</option>
                    <option value="button">button</option>
                  </select>
                  <label htmlFor="drawer-perm-key">前端标识 key</label>
                  <input
                    id="drawer-perm-key"
                    value={permissionForm.key}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, key: event.target.value }))}
                    placeholder="menu.tasks"
                  />
                  <label htmlFor="drawer-perm-resource">资源</label>
                  <input
                    id="drawer-perm-resource"
                    value={permissionForm.resource}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, resource: event.target.value }))}
                    placeholder="/api/v1/tasks"
                  />
                  <label htmlFor="drawer-perm-action">动作</label>
                  <input
                    id="drawer-perm-action"
                    value={permissionForm.action}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, action: event.target.value }))}
                    placeholder="GET|POST"
                  />
                  <label htmlFor="drawer-perm-dept">deptScope</label>
                  <input
                    id="drawer-perm-dept"
                    value={permissionForm.deptScope}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, deptScope: event.target.value }))}
                    placeholder="*"
                  />
                  <label htmlFor="drawer-perm-tag">resourceTagScope</label>
                  <input
                    id="drawer-perm-tag"
                    value={permissionForm.resourceTagScope}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, resourceTagScope: event.target.value }))}
                    placeholder="*"
                  />
                  <label htmlFor="drawer-perm-env">envScope</label>
                  <input
                    id="drawer-perm-env"
                    value={permissionForm.envScope}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, envScope: event.target.value }))}
                    placeholder="prod"
                  />
                  <label htmlFor="drawer-perm-desc">描述</label>
                  <input
                    id="drawer-perm-desc"
                    value={permissionForm.description}
                    onChange={(event) => setPermissionForm((prev) => ({ ...prev, description: event.target.value }))}
                    placeholder="ABAC 条件化权限"
                  />
                  <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>
                    {submitting ? "保存中..." : "保存"}
                  </button>
                </form>
              )}

              {showRoleDetail && !drawerLoading && (
                <div className="rbac-detail-stack">
                  <div className="rbac-kv-grid">
                    <div>
                      <span className="muted">角色名</span>
                      <strong>{displayRole?.name || `#${currentRoleId}`}</strong>
                    </div>
                    <div>
                      <span className="muted">描述</span>
                      <strong>{displayRole?.description || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">内置角色</span>
                      <strong>{displayRole?.builtIn ? "是" : "否"}</strong>
                    </div>
                    <div>
                      <span className="muted">更新时间</span>
                      <strong>{formatDateTime(displayRole?.updatedAt)}</strong>
                    </div>
                  </div>

                  <div className="rbac-header-row">
                    <h4>角色-权限绑定（多选）</h4>
                    <div className="rbac-binding-actions">
                      <input
                        className="rbac-binding-search"
                        type="search"
                        value={bindingKeyword}
                        onChange={(event) => setBindingKeyword(event.target.value)}
                        placeholder="搜索模块 / 权限名 / key / 资源 / 动作"
                      />
                      <label className="rbac-binding-toggle cursor-pointer">
                        <input
                          type="checkbox"
                          checked={bindingSelectedOnly}
                          onChange={(event) => setBindingSelectedOnly(event.target.checked)}
                        />
                        <span>只看已选</span>
                      </label>
                      <PermissionButton
                        permissionKey="button.rbac.binding.save"
                        className="btn primary cursor-pointer"
                        type="button"
                        onClick={() => void handleSaveBindings(currentRoleId)}
                        disabled={bindingSaving}
                      >
                        {bindingSaving ? "保存中..." : "保存绑定"}
                      </PermissionButton>
                    </div>
                  </div>

                  <p className="muted">
                    共 {bindingPermissions.length} 项，匹配 {filteredBindingPermissions.length} 项，已选 {selectedPermissionIds.length} 项
                  </p>

                  <div className="rbac-permission-groups">
                    {groupedPermissions.length === 0 && <p className="muted">未匹配到权限</p>}
                    {groupedPermissions.map((group) => (
                      <section className="permission-group" key={group.moduleKey}>
                        <div className="permission-group-header">
                          <h4>{formatModuleLabel(group.moduleKey)}（{group.list.length}）</h4>
                          <PermissionButton
                            permissionKey="button.rbac.binding.save"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => toggleModulePermissions(group.moduleKey)}
                          >
                            {bindingPermissions
                              .filter((permission) => extractPermissionModule(permission) === group.moduleKey)
                              .every((permission) => selectedPermissionIds.includes(permission.id))
                              ? "取消全选"
                              : "全选模块"}
                          </PermissionButton>
                        </div>
                        {group.list.map((permission) => (
                          <label className="permission-item cursor-pointer" key={permission.id}>
                            <input
                              type="checkbox"
                              checked={selectedPermissionIds.includes(permission.id)}
                              onChange={() => togglePermission(permission.id)}
                            />
                            <span>{permission.name}</span>
                            <small className="muted">
                              [{permission.type}]
                              {permission.key ? ` key=${permission.key} |` : ""}
                              {permission.resource} [{permission.action}] ({permission.deptScope || "*"}, {permission.resourceTagScope || "*"}, {permission.envScope || "*"})
                            </small>
                          </label>
                        ))}
                      </section>
                    ))}
                  </div>
                </div>
              )}

              {showPermissionDetail && !drawerLoading && permissionDetail && (
                <div className="rbac-detail-stack">
                  <div className="rbac-kv-grid">
                    <div>
                      <span className="muted">权限名</span>
                      <strong>{permissionDetail.name}</strong>
                    </div>
                    <div>
                      <span className="muted">类型</span>
                      <strong>{permissionDetail.type}</strong>
                    </div>
                    <div>
                      <span className="muted">Key</span>
                      <strong>{permissionDetail.key || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">资源</span>
                      <strong>{permissionDetail.resource}</strong>
                    </div>
                    <div>
                      <span className="muted">动作</span>
                      <strong>{permissionDetail.action}</strong>
                    </div>
                    <div>
                      <span className="muted">deptScope</span>
                      <strong>{permissionDetail.deptScope || "*"}</strong>
                    </div>
                    <div>
                      <span className="muted">resourceTagScope</span>
                      <strong>{permissionDetail.resourceTagScope || "*"}</strong>
                    </div>
                    <div>
                      <span className="muted">envScope</span>
                      <strong>{permissionDetail.envScope || "*"}</strong>
                    </div>
                    <div>
                      <span className="muted">描述</span>
                      <strong>{permissionDetail.description || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">更新时间</span>
                      <strong>{formatDateTime(permissionDetail.updatedAt)}</strong>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </aside>
        </div>
      )}
    </section>
  );
}
