import { FormEvent, useMemo, useState, useEffect } from "react";

import {
  bindDepartmentUsers,
  bindUserDepartments,
  bindUserRoles,
  createDepartment,
  createUser,
  deleteDepartment,
  deleteUser,
  getDepartment,
  getDepartmentUsers,
  getUser,
  getUserDepartments,
  getUserRoles,
  listDepartmentTree,
  listDepartments,
  listUsers,
  resetUserPassword,
  toggleUserStatus,
  updateDepartment,
  updateUser,
} from "@/api/users";
import { DeleteConfirmModal } from "@/components/DeleteConfirmModal";
import { FieldFilterPopover } from "@/components/FieldFilterPopover";
import { PermissionButton } from "@/components/PermissionButton";
import type { TableSettingsColumn } from "@/components/TableSettingsModal";
import { TableSettingsModal } from "@/components/TableSettingsModal";
import type { PageData } from "@/api/types";
import type { RoleItem } from "@/types/rbac";
import type { DepartmentListItem, DepartmentTreeNode, UserItem } from "@/types/users";
import {
  loadPersistedListSettings,
  sanitizeMultiFilterValues,
  sanitizeVisibleColumnKeys,
  savePersistedListSettings,
} from "@/utils/listSettings";
import { showToast } from "@/utils/toast";

const defaultPageSize = 10;
const pageSizeOptions = [10, 20, 50];
const USER_LIST_SETTINGS_KEY = "users.table.settings";
const DEPARTMENT_LIST_SETTINGS_KEY = "departments.table.settings";
const defaultUserVisibleColumnKeys = ["username", "displayName", "status", "updatedAt", "actions"];
const defaultDepartmentVisibleColumnKeys = ["name", "parentId", "updatedAt", "actions"];
const userStatusFilterOptions = [
  { value: "active", label: "启用" },
  { value: "inactive", label: "停用" },
] as const;
const userStatusFilterValues = userStatusFilterOptions.map((item) => item.value);

const userTableColumns: TableSettingsColumn[] = [
  { key: "username", label: "用户名" },
  { key: "displayName", label: "姓名" },
  { key: "email", label: "邮箱" },
  { key: "status", label: "状态" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];

const departmentTableColumns: TableSettingsColumn[] = [
  { key: "name", label: "部门名称" },
  { key: "parentId", label: "父部门ID" },
  { key: "updatedAt", label: "更新时间" },
  { key: "actions", label: "操作", required: true },
];

type DrawerState =
  | { type: "closed" }
  | { type: "user-create" }
  | { type: "user-edit"; userId: number }
  | { type: "user-detail"; userId: number }
  | { type: "user-reset-password"; userId: number }
  | { type: "user-bind-roles"; userId: number }
  | { type: "user-bind-departments"; userId: number }
  | { type: "department-create" }
  | { type: "department-edit"; departmentId: number }
  | { type: "department-detail"; departmentId: number }
  | { type: "department-bind-members"; departmentId: number };

type TableSettingsTarget = "closed" | "users" | "departments";

interface UserFormState {
  username: string;
  password: string;
  displayName: string;
  email: string;
  isActive: boolean;
}

interface DepartmentFormState {
  name: string;
  parentIdInput: string;
}

function defaultUserForm(): UserFormState {
  return {
    username: "",
    password: "",
    displayName: "",
    email: "",
    isActive: true,
  };
}

function defaultDepartmentForm(): DepartmentFormState {
  return {
    name: "",
    parentIdInput: "",
  };
}

export function UsersPage() {
  const [users, setUsers] = useState<UserItem[]>([]);
  const [departments, setDepartments] = useState<DepartmentListItem[]>([]);
  const [departmentTree, setDepartmentTree] = useState<DepartmentTreeNode[]>([]);
  const [userTotal, setUserTotal] = useState(0);
  const [departmentTotal, setDepartmentTotal] = useState(0);
  const [userPage, setUserPage] = useState(1);
  const [departmentPage, setDepartmentPage] = useState(1);
  const [userPageSize, setUserPageSize] = useState(defaultPageSize);
  const [departmentPageSize, setDepartmentPageSize] = useState(defaultPageSize);
  const [userJumpPageInput, setUserJumpPageInput] = useState("1");
  const [departmentJumpPageInput, setDepartmentJumpPageInput] = useState("1");
  const [drawer, setDrawer] = useState<DrawerState>({ type: "closed" });
  const [userForm, setUserForm] = useState<UserFormState>(defaultUserForm);
  const [departmentForm, setDepartmentForm] = useState<DepartmentFormState>(defaultDepartmentForm);
  const [resetPasswordInput, setResetPasswordInput] = useState("");
  const [userDetail, setUserDetail] = useState<UserItem | null>(null);
  const [departmentDetail, setDepartmentDetail] = useState<DepartmentListItem | null>(null);
  const [bindingRoles, setBindingRoles] = useState<RoleItem[]>([]);
  const [bindingDepartments, setBindingDepartments] = useState<DepartmentListItem[]>([]);
  const [bindingUsers, setBindingUsers] = useState<UserItem[]>([]);
  const [selectedRoleIds, setSelectedRoleIds] = useState<number[]>([]);
  const [selectedDepartmentIds, setSelectedDepartmentIds] = useState<number[]>([]);
  const [selectedMemberUserIds, setSelectedMemberUserIds] = useState<number[]>([]);
  const [memberSearchKeyword, setMemberSearchKeyword] = useState("");
  const [onlySelectedMembers, setOnlySelectedMembers] = useState(false);
  const [userListLoading, setUserListLoading] = useState(false);
  const [departmentListLoading, setDepartmentListLoading] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [bindingSaving, setBindingSaving] = useState(false);
  const [statusChangingUserId, setStatusChangingUserId] = useState<number | null>(null);
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteUserTarget, setDeleteUserTarget] = useState<UserItem | null>(null);
  const [deleteDepartmentTarget, setDeleteDepartmentTarget] = useState<DepartmentListItem | null>(null);
  const [tableSettingsTarget, setTableSettingsTarget] = useState<TableSettingsTarget>("closed");
  const [visibleUserColumnKeys, setVisibleUserColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(USER_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultUserVisibleColumnKeys, userTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, userTableColumns);
  });
  const [visibleDepartmentColumnKeys, setVisibleDepartmentColumnKeys] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(DEPARTMENT_LIST_SETTINGS_KEY);
    const defaults = sanitizeVisibleColumnKeys(defaultDepartmentVisibleColumnKeys, departmentTableColumns);
    return sanitizeVisibleColumnKeys(persisted?.visibleColumnKeys ?? defaults, departmentTableColumns);
  });
  const [selectedUserStatusValues, setSelectedUserStatusValues] = useState<string[]>(() => {
    const persisted = loadPersistedListSettings(USER_LIST_SETTINGS_KEY);
    return sanitizeMultiFilterValues(
      persisted?.userStatusValues,
      userStatusFilterValues,
      userStatusFilterValues,
    );
  });

  useEffect(() => {
    void loadUserPage(userPage, userPageSize);
  }, [userPage, userPageSize]);

  useEffect(() => {
    void loadDepartmentPage(departmentPage, departmentPageSize);
  }, [departmentPage, departmentPageSize]);

  useEffect(() => {
    void loadDepartmentTree();
  }, []);

  useEffect(() => {
    setUserJumpPageInput(String(userPage));
  }, [userPage]);

  useEffect(() => {
    setDepartmentJumpPageInput(String(departmentPage));
  }, [departmentPage]);

  useEffect(() => {
    savePersistedListSettings(USER_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleUserColumnKeys,
      userStatusValues: selectedUserStatusValues,
    });
  }, [selectedUserStatusValues, visibleUserColumnKeys]);

  useEffect(() => {
    savePersistedListSettings(DEPARTMENT_LIST_SETTINGS_KEY, {
      visibleColumnKeys: visibleDepartmentColumnKeys,
    });
  }, [visibleDepartmentColumnKeys]);

  useEffect(() => {
    if (drawer.type === "user-edit" || drawer.type === "user-detail" || drawer.type === "user-reset-password") {
      void loadUserDetail(drawer.userId);
      return;
    }
    if (drawer.type === "user-bind-roles") {
      void loadUserRoleBinding(drawer.userId);
      return;
    }
    if (drawer.type === "user-bind-departments") {
      void loadUserDepartmentBinding(drawer.userId);
      return;
    }
    if (drawer.type === "department-edit" || drawer.type === "department-detail") {
      void loadDepartmentDetail(drawer.departmentId);
      return;
    }
    if (drawer.type === "department-bind-members") {
      void loadDepartmentMemberBinding(drawer.departmentId);
      return;
    }

    setUserDetail(null);
    setDepartmentDetail(null);
    setBindingRoles([]);
    setBindingDepartments([]);
    setBindingUsers([]);
    setSelectedRoleIds([]);
    setSelectedDepartmentIds([]);
    setSelectedMemberUserIds([]);
    setMemberSearchKeyword("");
    setOnlySelectedMembers(false);
    setResetPasswordInput("");
  }, [drawer]);

  async function loadUserPage(page: number, pageSize: number) {
    setUserListLoading(true);
    try {
      const data = await listUsers(page, pageSize);
      syncPagedList(
        data,
        pageSize,
        page,
        setUserPage,
        setUsers,
        setUserTotal,
      );
    } catch {
      showToast("用户列表加载失败");
    } finally {
      setUserListLoading(false);
    }
  }

  async function loadDepartmentPage(page: number, pageSize: number) {
    setDepartmentListLoading(true);
    try {
      const data = await listDepartments(page, pageSize);
      syncPagedList(
        data,
        pageSize,
        page,
        setDepartmentPage,
        setDepartments,
        setDepartmentTotal,
      );
    } catch {
      showToast("部门列表加载失败");
    } finally {
      setDepartmentListLoading(false);
    }
  }

  async function loadDepartmentTree() {
    try {
      const tree = await listDepartmentTree();
      setDepartmentTree(tree);
    } catch {
      showToast("部门树加载失败");
    }
  }

  async function loadUserDetail(userId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getUser(userId);
      setUserDetail(detail);
      if (drawer.type === "user-edit") {
        setUserForm({
          username: detail.username,
          password: "",
          displayName: detail.displayName ?? "",
          email: detail.email ?? "",
          isActive: detail.isActive,
        });
      }
    } catch {
      showToast("用户详情加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  async function loadDepartmentDetail(departmentId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getDepartment(departmentId);
      setDepartmentDetail(detail);
      if (drawer.type === "department-edit") {
        setDepartmentForm({
          name: detail.name,
          parentIdInput: detail.parentId ? String(detail.parentId) : "",
        });
      }
    } catch {
      showToast("部门详情加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  async function loadUserRoleBinding(userId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getUserRoles(userId);
      setUserDetail(detail.user);
      setBindingRoles(detail.roles ?? []);
      setSelectedRoleIds((detail.roleIds ?? []).map(Number));
    } catch {
      showToast("用户角色绑定加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  async function loadUserDepartmentBinding(userId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getUserDepartments(userId);
      setUserDetail(detail.user);
      setBindingDepartments(detail.departments ?? []);
      setSelectedDepartmentIds((detail.departmentIds ?? []).map(Number));
    } catch {
      showToast("用户部门绑定加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  async function loadDepartmentMemberBinding(departmentId: number) {
    setDrawerLoading(true);
    try {
      const detail = await getDepartmentUsers(departmentId);
      setDepartmentDetail(detail.department);
      setBindingUsers(detail.users ?? []);
      setSelectedMemberUserIds((detail.userIds ?? []).map(Number));
    } catch {
      showToast("部门成员绑定加载失败");
    } finally {
      setDrawerLoading(false);
    }
  }

  function openUserCreateDrawer() {
    setUserForm(defaultUserForm());
    setDrawer({ type: "user-create" });
  }

  function openUserEditDrawer(userId: number) {
    setUserForm(defaultUserForm());
    setDrawer({ type: "user-edit", userId });
  }

  function openUserDetailDrawer(userId: number) {
    setDrawer({ type: "user-detail", userId });
  }

  function openUserResetPasswordDrawer(userId: number) {
    setResetPasswordInput("");
    setDrawer({ type: "user-reset-password", userId });
  }

  function openUserBindRolesDrawer(userId: number) {
    setDrawer({ type: "user-bind-roles", userId });
  }

  function openUserBindDepartmentsDrawer(userId: number) {
    setDrawer({ type: "user-bind-departments", userId });
  }

  function openDepartmentCreateDrawer() {
    setDepartmentForm(defaultDepartmentForm());
    setDrawer({ type: "department-create" });
  }

  function openDepartmentEditDrawer(departmentId: number) {
    setDepartmentForm(defaultDepartmentForm());
    setDrawer({ type: "department-edit", departmentId });
  }

  function openDepartmentDetailDrawer(departmentId: number) {
    setDrawer({ type: "department-detail", departmentId });
  }

  function openDepartmentBindMembersDrawer(departmentId: number) {
    setMemberSearchKeyword("");
    setOnlySelectedMembers(false);
    setSelectedMemberUserIds([]);
    setBindingUsers([]);
    setDrawer({ type: "department-bind-members", departmentId });
  }

  function openUserTableSettings() {
    setTableSettingsTarget("users");
  }

  function openDepartmentTableSettings() {
    setTableSettingsTarget("departments");
  }

  function toggleUserVisibleColumn(columnKey: string) {
    const column = userTableColumns.find((item) => item.key === columnKey);
    if (!column || column.required) return;
    setVisibleUserColumnKeys((prev) => (
      prev.includes(columnKey)
        ? prev.filter((key) => key !== columnKey)
        : [...prev, columnKey]
    ));
  }

  function toggleDepartmentVisibleColumn(columnKey: string) {
    const column = departmentTableColumns.find((item) => item.key === columnKey);
    if (!column || column.required) return;
    setVisibleDepartmentColumnKeys((prev) => (
      prev.includes(columnKey)
        ? prev.filter((key) => key !== columnKey)
        : [...prev, columnKey]
    ));
  }

  function toggleRoleSelection(roleId: number) {
    setSelectedRoleIds((prev) => (
      prev.includes(roleId) ? prev.filter((item) => item !== roleId) : [...prev, roleId]
    ));
  }

  function toggleDepartmentSelection(departmentId: number) {
    setSelectedDepartmentIds((prev) => (
      prev.includes(departmentId) ? prev.filter((item) => item !== departmentId) : [...prev, departmentId]
    ));
  }

  function toggleMemberUserSelection(userId: number) {
    setSelectedMemberUserIds((prev) => (
      prev.includes(userId) ? prev.filter((item) => item !== userId) : [...prev, userId]
    ));
  }

  async function handleSubmitUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const username = userForm.username.trim();
    const displayName = userForm.displayName.trim();
    const email = userForm.email.trim();

    if (drawer.type === "user-create") {
      const password = userForm.password.trim();
      if (!username || !password) {
        showToast("用户名和密码必填");
        return;
      }
      setSubmitting(true);
      try {
        await createUser({
          username,
          password,
          displayName,
          email,
          isActive: userForm.isActive,
        });
        showToast("用户创建成功");
        if (userPage === 1) {
          await loadUserPage(1, userPageSize);
        } else {
          setUserPage(1);
        }
        setDrawer({ type: "closed" });
      } catch {
        showToast("用户创建失败");
      } finally {
        setSubmitting(false);
      }
      return;
    }

    if (drawer.type === "user-edit") {
      setSubmitting(true);
      try {
        await updateUser(drawer.userId, {
          displayName,
          email,
          isActive: userForm.isActive,
        });
        showToast("用户更新成功");
        await loadUserPage(userPage, userPageSize);
        setDrawer({ type: "closed" });
      } catch {
        showToast("用户更新失败");
      } finally {
        setSubmitting(false);
      }
    }
  }

  async function handleSubmitDepartment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = departmentForm.name.trim();
    if (!name) {
      showToast("部门名称必填");
      return;
    }

    const parentId = parseOptionalPositiveInt(departmentForm.parentIdInput);
    if (departmentForm.parentIdInput.trim() !== "" && parentId === undefined) {
      showToast("父部门ID必须为正整数");
      return;
    }

    setSubmitting(true);
    try {
      if (drawer.type === "department-create") {
        const payload = parentId === undefined ? { name } : { name, parentId };
        await createDepartment(payload);
        showToast("部门创建成功");
        if (departmentPage === 1) {
          await loadDepartmentPage(1, departmentPageSize);
        } else {
          setDepartmentPage(1);
        }
      } else if (drawer.type === "department-edit") {
        await updateDepartment(drawer.departmentId, {
          name,
          parentId: parentId ?? null,
        });
        showToast("部门更新成功");
        await loadDepartmentPage(departmentPage, departmentPageSize);
      }
      await loadDepartmentTree();
      setDrawer({ type: "closed" });
    } catch {
      showToast("部门保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSubmitResetPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (drawer.type !== "user-reset-password") return;
    const password = resetPasswordInput.trim();
    if (!password) {
      showToast("新密码不能为空");
      return;
    }

    setSubmitting(true);
    try {
      await resetUserPassword(drawer.userId, password);
      showToast("密码已重置");
      setDrawer({ type: "closed" });
    } catch {
      showToast("密码重置失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSaveUserRoles(userId: number) {
    setBindingSaving(true);
    try {
      await bindUserRoles(userId, selectedRoleIds);
      showToast("用户角色绑定已保存");
      await loadUserRoleBinding(userId);
    } catch {
      showToast("用户角色绑定失败");
    } finally {
      setBindingSaving(false);
    }
  }

  async function handleSaveUserDepartments(userId: number) {
    setBindingSaving(true);
    try {
      await bindUserDepartments(userId, selectedDepartmentIds);
      showToast("用户部门绑定已保存");
      await loadUserDepartmentBinding(userId);
    } catch {
      showToast("用户部门绑定失败");
    } finally {
      setBindingSaving(false);
    }
  }

  async function handleSaveDepartmentUsers(departmentId: number) {
    setBindingSaving(true);
    try {
      await bindDepartmentUsers(departmentId, selectedMemberUserIds);
      showToast("部门成员绑定已保存");
      await loadDepartmentMemberBinding(departmentId);
    } catch {
      showToast("部门成员绑定失败");
    } finally {
      setBindingSaving(false);
    }
  }

  async function handleDeleteUser(user: UserItem) {
    try {
      await deleteUser(user.id);
      showToast("用户已删除");
      if (users.length === 1 && userPage > 1) {
        setUserPage(userPage - 1);
      } else {
        await loadUserPage(userPage, userPageSize);
      }
      if (
        (drawer.type === "user-detail" || drawer.type === "user-edit" || drawer.type === "user-reset-password" || drawer.type === "user-bind-roles" || drawer.type === "user-bind-departments")
        && drawer.userId === user.id
      ) {
        setDrawer({ type: "closed" });
      }
    } catch {
      showToast("用户删除失败");
    }
  }

  async function handleDeleteDepartment(department: DepartmentListItem) {
    try {
      await deleteDepartment(department.id);
      showToast("部门已删除");
      if (departments.length === 1 && departmentPage > 1) {
        setDepartmentPage(departmentPage - 1);
      } else {
        await loadDepartmentPage(departmentPage, departmentPageSize);
      }
      await loadDepartmentTree();
      if ((drawer.type === "department-detail" || drawer.type === "department-edit" || drawer.type === "department-bind-members") && drawer.departmentId === department.id) {
        setDrawer({ type: "closed" });
      }
    } catch {
      showToast("部门删除失败");
    }
  }

  function requestDeleteUser(user: UserItem) {
    setDeleteUserTarget(user);
  }

  function requestDeleteDepartment(department: DepartmentListItem) {
    setDeleteDepartmentTarget(department);
  }

  async function confirmDeleteUser() {
    if (!deleteUserTarget) return;
    setDeleteSubmitting(true);
    await handleDeleteUser(deleteUserTarget);
    setDeleteUserTarget(null);
    setDeleteSubmitting(false);
  }

  async function confirmDeleteDepartment() {
    if (!deleteDepartmentTarget) return;
    setDeleteSubmitting(true);
    await handleDeleteDepartment(deleteDepartmentTarget);
    setDeleteDepartmentTarget(null);
    setDeleteSubmitting(false);
  }

  async function handleToggleUserActive(user: UserItem) {
    setStatusChangingUserId(user.id);
    try {
      await toggleUserStatus(user.id, !user.isActive);
      showToast(user.isActive ? "用户已停用" : "用户已启用");
      await loadUserPage(userPage, userPageSize);
      if (drawer.type === "user-detail" && drawer.userId === user.id) {
        await loadUserDetail(user.id);
      }
    } catch {
      showToast("用户状态更新失败");
    } finally {
      setStatusChangingUserId(null);
    }
  }

  function drawerTitle(): string {
    switch (drawer.type) {
      case "user-create":
        return "创建用户";
      case "user-edit":
        return "修改用户";
      case "user-detail":
        return "用户详情";
      case "user-reset-password":
        return "重置用户密码";
      case "user-bind-roles":
        return "绑定用户角色";
      case "user-bind-departments":
        return "绑定用户部门";
      case "department-create":
        return "创建部门";
      case "department-edit":
        return "修改部门";
      case "department-detail":
        return "部门详情";
      case "department-bind-members":
        return "绑定部门成员";
      default:
        return "";
    }
  }

  function renderPagination(props: {
    page: number;
    total: number;
    pageSize: number;
    jumpValue: string;
    onPageChange: (nextPage: number) => void;
    onPageSizeChange: (nextPageSize: number) => void;
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
          <button className="btn ghost cursor-pointer" type="button" onClick={() => onPageChange(page - 1)} disabled={page <= 1}>
            上一页
          </button>
          <span className="rbac-pagination-text">第 {page} / {pages} 页</span>
          <button className="btn ghost cursor-pointer" type="button" onClick={() => onPageChange(page + 1)} disabled={page >= pages}>
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
  const showUserForm = drawer.type === "user-create" || drawer.type === "user-edit";
  const showDepartmentForm = drawer.type === "department-create" || drawer.type === "department-edit";
  const showUserDetail = drawer.type === "user-detail";
  const showDepartmentDetail = drawer.type === "department-detail";
  const showResetPasswordForm = drawer.type === "user-reset-password";
  const showBindRoles = drawer.type === "user-bind-roles";
  const showBindDepartments = drawer.type === "user-bind-departments";
  const showBindDepartmentMembers = drawer.type === "department-bind-members";
  const currentBindingUserId = showBindRoles || showBindDepartments ? drawer.userId : 0;
  const currentBindingDepartmentId = showBindDepartmentMembers ? drawer.departmentId : 0;
  const parentDepartmentOptions = flattenDepartmentTreeOptions(departmentTree).filter((option) => (
    drawer.type !== "department-edit" || option.id !== drawer.departmentId
  ));
  const filteredBindingUsers = useMemo(() => bindingUsers.filter((user) => {
    if (onlySelectedMembers && !selectedMemberUserIds.includes(user.id)) return false;
    const keyword = memberSearchKeyword.trim().toLowerCase();
    if (!keyword) return true;
    const text = `${user.username} ${user.displayName ?? ""} ${user.email ?? ""}`.toLowerCase();
    return text.includes(keyword);
  }), [bindingUsers, memberSearchKeyword, onlySelectedMembers, selectedMemberUserIds]);
  const filteredUsers = useMemo(() => users.filter((user) => {
    const status = user.isActive ? "active" : "inactive";
    return selectedUserStatusValues.includes(status);
  }), [selectedUserStatusValues, users]);
  const filteredDepartments = useMemo(() => departments, [departments]);
  const userRows = userListLoading ? [] : filteredUsers;
  const departmentRows = departmentListLoading ? [] : filteredDepartments;
  const userVisibleColumnSet = new Set(visibleUserColumnKeys);
  const departmentVisibleColumnSet = new Set(visibleDepartmentColumnKeys);
  const userColSpan = Math.max(1, visibleUserColumnKeys.length);
  const departmentColSpan = Math.max(1, visibleDepartmentColumnKeys.length);

  return (
    <section className="page">
      <h2>用户与部门(组)管理</h2>

      <div className="rbac-module-scroll">
        <div className="rbac-module-grid">
          <article className="card rbac-module-card">
            <div className="rbac-module-header">
              <div>
                <h3>用户子模块</h3>
                <p className="muted">用户列表、详情、创建、修改、启停、重置密码、角色绑定、部门绑定</p>
              </div>
              <PermissionButton
                permissionKey="button.users.user.create"
                className="btn primary cursor-pointer"
                type="button"
                onClick={openUserCreateDrawer}
              >
                创建用户
              </PermissionButton>
            </div>

            <div className="rbac-table-wrapper">
              <table className="rbac-table">
                <thead>
                  <tr>
                    {userVisibleColumnSet.has("username") && <th>用户名</th>}
                    {userVisibleColumnSet.has("displayName") && <th>姓名</th>}
                    {userVisibleColumnSet.has("email") && <th>邮箱</th>}
                    {userVisibleColumnSet.has("status") && (
                      <th>
                        <div className="table-actions-header">
                          <span>状态</span>
                          <FieldFilterPopover
                            ariaLabel="用户状态筛选"
                            options={[...userStatusFilterOptions]}
                            selectedValues={selectedUserStatusValues}
                            onChange={setSelectedUserStatusValues}
                          />
                        </div>
                      </th>
                    )}
                    {userVisibleColumnSet.has("updatedAt") && <th>更新时间</th>}
                    {userVisibleColumnSet.has("actions") && (
                      <th>
                        <div className="table-actions-header">
                          <span>操作</span>
                          <button className="table-settings-trigger cursor-pointer" type="button" onClick={openUserTableSettings} aria-label="用户列表设置">
                            ⚙️
                          </button>
                        </div>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {userRows.length === 0 && (
                    <tr>
                      <td colSpan={userColSpan} className="muted">{userListLoading ? "加载中..." : "暂无用户"}</td>
                    </tr>
                  )}
                  {userRows.map((user) => (
                    <tr key={user.id}>
                      {userVisibleColumnSet.has("username") && <td>{user.username}</td>}
                      {userVisibleColumnSet.has("displayName") && <td>{user.displayName || "-"}</td>}
                      {userVisibleColumnSet.has("email") && <td>{user.email || "-"}</td>}
                      {userVisibleColumnSet.has("status") && <td>{user.isActive ? "启用" : "停用"}</td>}
                      {userVisibleColumnSet.has("updatedAt") && <td>{formatDateTime(user.updatedAt)}</td>}
                      {userVisibleColumnSet.has("actions") && (
                        <td className="rbac-row-actions">
                          <button className="btn ghost cursor-pointer" type="button" onClick={() => openUserDetailDrawer(user.id)}>
                            查看详情
                          </button>
                          <PermissionButton permissionKey="button.users.user.update" className="btn ghost cursor-pointer" type="button" onClick={() => openUserEditDrawer(user.id)}>
                            修改
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.user.toggle_status"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => void handleToggleUserActive(user)}
                            disabled={statusChangingUserId === user.id}
                          >
                            {statusChangingUserId === user.id ? "处理中..." : user.isActive ? "停用" : "启用"}
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.user.reset_password"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => openUserResetPasswordDrawer(user.id)}
                          >
                            重置密码
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.user.bind_roles"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => openUserBindRolesDrawer(user.id)}
                          >
                            绑定角色
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.user.bind_departments"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => openUserBindDepartmentsDrawer(user.id)}
                          >
                            绑定部门
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.user.delete"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => requestDeleteUser(user)}
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
              page: userPage,
              total: userTotal,
              pageSize: userPageSize,
              jumpValue: userJumpPageInput,
              onPageChange: (nextPage) => handleUserPageChange(nextPage, userTotal, userPageSize, userPage, setUserPage),
              onPageSizeChange: (nextPageSize) => {
                if (nextPageSize === userPageSize) return;
                setUserPageSize(nextPageSize);
                setUserPage(1);
              },
              onJumpValueChange: setUserJumpPageInput,
              onJumpSubmit: () => handleUserJumpSubmit(userJumpPageInput, userPage, userTotal, userPageSize, setUserPage, setUserJumpPageInput),
            })}
          </article>

          <article className="card rbac-module-card">
            <div className="rbac-module-header">
              <div>
                <h3>部门(组)子模块</h3>
                <p className="muted">部门列表、详情、创建、修改、删除、成员绑定</p>
              </div>
              <PermissionButton
                permissionKey="button.users.department.create"
                className="btn primary cursor-pointer"
                type="button"
                onClick={openDepartmentCreateDrawer}
              >
                创建部门
              </PermissionButton>
            </div>

            <div className="rbac-table-wrapper">
              <table className="rbac-table">
                <thead>
                  <tr>
                    {departmentVisibleColumnSet.has("name") && <th>部门名称</th>}
                    {departmentVisibleColumnSet.has("parentId") && <th>父部门ID</th>}
                    {departmentVisibleColumnSet.has("updatedAt") && <th>更新时间</th>}
                    {departmentVisibleColumnSet.has("actions") && (
                      <th>
                        <div className="table-actions-header">
                          <span>操作</span>
                          <button className="table-settings-trigger cursor-pointer" type="button" onClick={openDepartmentTableSettings} aria-label="部门列表设置">
                            ⚙️
                          </button>
                        </div>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {departmentRows.length === 0 && (
                    <tr>
                      <td colSpan={departmentColSpan} className="muted">{departmentListLoading ? "加载中..." : "暂无部门"}</td>
                    </tr>
                  )}
                  {departmentRows.map((department) => (
                    <tr key={department.id}>
                      {departmentVisibleColumnSet.has("name") && <td>{department.name}</td>}
                      {departmentVisibleColumnSet.has("parentId") && <td>{department.parentId || "-"}</td>}
                      {departmentVisibleColumnSet.has("updatedAt") && <td>{formatDateTime(department.updatedAt)}</td>}
                      {departmentVisibleColumnSet.has("actions") && (
                        <td className="rbac-row-actions">
                          <button className="btn ghost cursor-pointer" type="button" onClick={() => openDepartmentDetailDrawer(department.id)}>
                            查看详情
                          </button>
                          <PermissionButton permissionKey="button.users.department.update" className="btn ghost cursor-pointer" type="button" onClick={() => openDepartmentEditDrawer(department.id)}>
                            修改
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.department.bind_members"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => openDepartmentBindMembersDrawer(department.id)}
                          >
                            绑定成员
                          </PermissionButton>
                          <PermissionButton
                            permissionKey="button.users.department.delete"
                            className="btn ghost cursor-pointer"
                            type="button"
                            onClick={() => requestDeleteDepartment(department)}
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
              page: departmentPage,
              total: departmentTotal,
              pageSize: departmentPageSize,
              jumpValue: departmentJumpPageInput,
              onPageChange: (nextPage) => handleDepartmentPageChange(nextPage, departmentTotal, departmentPageSize, departmentPage, setDepartmentPage),
              onPageSizeChange: (nextPageSize) => {
                if (nextPageSize === departmentPageSize) return;
                setDepartmentPageSize(nextPageSize);
                setDepartmentPage(1);
              },
              onJumpValueChange: setDepartmentJumpPageInput,
              onJumpSubmit: () => handleDepartmentJumpSubmit(departmentJumpPageInput, departmentPage, departmentTotal, departmentPageSize, setDepartmentPage, setDepartmentJumpPageInput),
            })}
          </article>
        </div>
      </div>

      <TableSettingsModal
        open={tableSettingsTarget === "users"}
        title="用户列表设置"
        columns={userTableColumns}
        visibleColumnKeys={visibleUserColumnKeys}
        onToggleColumn={toggleUserVisibleColumn}
        onReset={() => {
          setVisibleUserColumnKeys(sanitizeVisibleColumnKeys(defaultUserVisibleColumnKeys, userTableColumns));
        }}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <TableSettingsModal
        open={tableSettingsTarget === "departments"}
        title="部门列表设置"
        columns={departmentTableColumns}
        visibleColumnKeys={visibleDepartmentColumnKeys}
        onToggleColumn={toggleDepartmentVisibleColumn}
        onReset={() => {
          setVisibleDepartmentColumnKeys(sanitizeVisibleColumnKeys(defaultDepartmentVisibleColumnKeys, departmentTableColumns));
        }}
        onClose={() => setTableSettingsTarget("closed")}
      />

      <DeleteConfirmModal
        open={deleteUserTarget !== null}
        title="删除用户确认"
        description={`将删除用户：${deleteUserTarget?.username || "-"}`}
        confirming={deleteSubmitting}
        onCancel={() => setDeleteUserTarget(null)}
        onConfirm={() => void confirmDeleteUser()}
      />

      <DeleteConfirmModal
        open={deleteDepartmentTarget !== null}
        title="删除部门确认"
        description={`将删除部门：${deleteDepartmentTarget?.name || "-"}`}
        confirming={deleteSubmitting}
        onCancel={() => setDeleteDepartmentTarget(null)}
        onConfirm={() => void confirmDeleteDepartment()}
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

              {showUserForm && !drawerLoading && (
                <form className="form-grid" onSubmit={handleSubmitUser}>
                  <label htmlFor="users-username">用户名</label>
                  <input
                    id="users-username"
                    value={userForm.username}
                    onChange={(event) => setUserForm((prev) => ({ ...prev, username: event.target.value }))}
                    placeholder="ops-user"
                    disabled={drawer.type === "user-edit"}
                  />
                  {drawer.type === "user-create" && (
                    <>
                      <label htmlFor="users-password">密码</label>
                      <input
                        id="users-password"
                        type="password"
                        value={userForm.password}
                        onChange={(event) => setUserForm((prev) => ({ ...prev, password: event.target.value }))}
                        placeholder="请输入密码"
                      />
                    </>
                  )}
                  <label htmlFor="users-display-name">姓名</label>
                  <input
                    id="users-display-name"
                    value={userForm.displayName}
                    onChange={(event) => setUserForm((prev) => ({ ...prev, displayName: event.target.value }))}
                    placeholder="运维同学"
                  />
                  <label htmlFor="users-email">邮箱</label>
                  <input
                    id="users-email"
                    value={userForm.email}
                    onChange={(event) => setUserForm((prev) => ({ ...prev, email: event.target.value }))}
                    placeholder="user@example.com"
                  />
                  <label htmlFor="users-active">状态</label>
                  <select
                    id="users-active"
                    value={userForm.isActive ? "1" : "0"}
                    onChange={(event) => setUserForm((prev) => ({ ...prev, isActive: event.target.value === "1" }))}
                  >
                    <option value="1">启用</option>
                    <option value="0">停用</option>
                  </select>
                  <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>
                    {submitting ? "保存中..." : "保存"}
                  </button>
                </form>
              )}

              {showDepartmentForm && !drawerLoading && (
                <form className="form-grid" onSubmit={handleSubmitDepartment}>
                  <label htmlFor="department-name">部门名称</label>
                  <input
                    id="department-name"
                    value={departmentForm.name}
                    onChange={(event) => setDepartmentForm((prev) => ({ ...prev, name: event.target.value }))}
                    placeholder="运维部"
                  />
                  <label htmlFor="department-parent-id">父部门ID（可空）</label>
                  <select
                    id="department-parent-id"
                    value={departmentForm.parentIdInput}
                    onChange={(event) => setDepartmentForm((prev) => ({ ...prev, parentIdInput: event.target.value }))}
                  >
                    <option value="">无</option>
                    {parentDepartmentOptions.map((option) => (
                      <option key={option.id} value={String(option.id)}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                  <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>
                    {submitting ? "保存中..." : "保存"}
                  </button>
                </form>
              )}

              {showResetPasswordForm && !drawerLoading && (
                <form className="form-grid" onSubmit={handleSubmitResetPassword}>
                  <label htmlFor="reset-password-input">新密码</label>
                  <input
                    id="reset-password-input"
                    type="password"
                    value={resetPasswordInput}
                    onChange={(event) => setResetPasswordInput(event.target.value)}
                    placeholder="请输入新密码"
                  />
                  <button className="btn primary cursor-pointer" type="submit" disabled={submitting}>
                    {submitting ? "保存中..." : "确认重置"}
                  </button>
                </form>
              )}

              {showUserDetail && !drawerLoading && userDetail && (
                <div className="rbac-detail-stack">
                  <div className="rbac-kv-grid">
                    <div>
                      <span className="muted">用户名</span>
                      <strong>{userDetail.username}</strong>
                    </div>
                    <div>
                      <span className="muted">姓名</span>
                      <strong>{userDetail.displayName || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">邮箱</span>
                      <strong>{userDetail.email || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">状态</span>
                      <strong>{userDetail.isActive ? "启用" : "停用"}</strong>
                    </div>
                    <div>
                      <span className="muted">更新时间</span>
                      <strong>{formatDateTime(userDetail.updatedAt)}</strong>
                    </div>
                  </div>
                </div>
              )}

              {showDepartmentDetail && !drawerLoading && departmentDetail && (
                <div className="rbac-detail-stack">
                  <div className="rbac-kv-grid">
                    <div>
                      <span className="muted">部门名称</span>
                      <strong>{departmentDetail.name}</strong>
                    </div>
                    <div>
                      <span className="muted">父部门ID</span>
                      <strong>{departmentDetail.parentId || "-"}</strong>
                    </div>
                    <div>
                      <span className="muted">更新时间</span>
                      <strong>{formatDateTime(departmentDetail.updatedAt)}</strong>
                    </div>
                  </div>
                </div>
              )}

              {showBindRoles && !drawerLoading && (
                <div className="rbac-detail-stack">
                  <div className="rbac-header-row">
                    <h4>用户角色绑定（多选）</h4>
                    <PermissionButton
                      permissionKey="button.users.user.bind_roles"
                      className="btn primary cursor-pointer"
                      type="button"
                      onClick={() => void handleSaveUserRoles(currentBindingUserId)}
                      disabled={bindingSaving}
                    >
                      {bindingSaving ? "保存中..." : "保存绑定"}
                    </PermissionButton>
                  </div>
                  <p className="muted">
                    用户：{userDetail?.username || `#${currentBindingUserId}`}，共 {bindingRoles.length} 个角色，已选 {selectedRoleIds.length} 个
                  </p>
                  <div className="rbac-kv-grid">
                    {bindingRoles.length === 0 && <p className="muted">暂无角色</p>}
                    {bindingRoles.map((role) => (
                      <label className="permission-item cursor-pointer" key={role.id}>
                        <input
                          type="checkbox"
                          checked={selectedRoleIds.includes(role.id)}
                          onChange={() => toggleRoleSelection(role.id)}
                        />
                        <span>{role.name}</span>
                        <small className="muted">{role.description || "-"}</small>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {showBindDepartments && !drawerLoading && (
                <div className="rbac-detail-stack">
                  <div className="rbac-header-row">
                    <h4>用户部门绑定（多选）</h4>
                    <PermissionButton
                      permissionKey="button.users.user.bind_departments"
                      className="btn primary cursor-pointer"
                      type="button"
                      onClick={() => void handleSaveUserDepartments(currentBindingUserId)}
                      disabled={bindingSaving}
                    >
                      {bindingSaving ? "保存中..." : "保存绑定"}
                    </PermissionButton>
                  </div>
                  <p className="muted">
                    用户：{userDetail?.username || `#${currentBindingUserId}`}，共 {bindingDepartments.length} 个部门，已选 {selectedDepartmentIds.length} 个
                  </p>
                  <div className="rbac-kv-grid">
                    {bindingDepartments.length === 0 && <p className="muted">暂无部门</p>}
                    {bindingDepartments.map((department) => (
                      <label className="permission-item cursor-pointer" key={department.id}>
                        <input
                          type="checkbox"
                          checked={selectedDepartmentIds.includes(department.id)}
                          onChange={() => toggleDepartmentSelection(department.id)}
                        />
                        <span>{department.name}</span>
                        <small className="muted">parentId: {department.parentId || "-"}</small>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {showBindDepartmentMembers && !drawerLoading && (
                <div className="rbac-detail-stack">
                  <div className="rbac-header-row">
                    <h4>部门成员绑定（多选）</h4>
                    <PermissionButton
                      permissionKey="button.users.department.bind_members"
                      className="btn primary cursor-pointer"
                      type="button"
                      onClick={() => void handleSaveDepartmentUsers(currentBindingDepartmentId)}
                      disabled={bindingSaving}
                    >
                      {bindingSaving ? "保存中..." : "保存绑定"}
                    </PermissionButton>
                  </div>
                  <div className="rbac-binding-actions">
                    <input
                      className="rbac-binding-search"
                      placeholder="搜索用户名/姓名/邮箱"
                      value={memberSearchKeyword}
                      onChange={(event) => setMemberSearchKeyword(event.target.value)}
                    />
                    <label className="rbac-binding-toggle cursor-pointer">
                      <input
                        type="checkbox"
                        checked={onlySelectedMembers}
                        onChange={(event) => setOnlySelectedMembers(event.target.checked)}
                      />
                      只看已选
                    </label>
                  </div>
                  <p className="muted">
                    部门：{departmentDetail?.name || `#${currentBindingDepartmentId}`}，共 {bindingUsers.length} 个用户，已选 {selectedMemberUserIds.length} 个
                  </p>
                  <div className="rbac-kv-grid">
                    {filteredBindingUsers.length === 0 && <p className="muted">暂无可选用户</p>}
                    {filteredBindingUsers.map((user) => (
                      <label className="permission-item cursor-pointer" key={user.id}>
                        <input
                          type="checkbox"
                          checked={selectedMemberUserIds.includes(user.id)}
                          onChange={() => toggleMemberUserSelection(user.id)}
                        />
                        <span>{user.username}</span>
                        <small className="muted">{user.displayName || user.email || "-"}</small>
                      </label>
                    ))}
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

function totalPages(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

function clampPage(page: number, total: number, pageSize: number): number {
  return Math.min(Math.max(1, page), totalPages(total, pageSize));
}

function handleUserPageChange(
  nextPage: number,
  total: number,
  pageSize: number,
  currentPage: number,
  setPage: (page: number) => void,
) {
  const target = clampPage(nextPage, total, pageSize);
  if (target !== currentPage) setPage(target);
}

function handleDepartmentPageChange(
  nextPage: number,
  total: number,
  pageSize: number,
  currentPage: number,
  setPage: (page: number) => void,
) {
  const target = clampPage(nextPage, total, pageSize);
  if (target !== currentPage) setPage(target);
}

function handleUserJumpSubmit(
  input: string,
  currentPage: number,
  total: number,
  pageSize: number,
  setPage: (page: number) => void,
  setInput: (value: string) => void,
) {
  const target = Number.parseInt(input, 10);
  if (Number.isNaN(target)) {
    showToast("请输入正确页码");
    setInput(String(currentPage));
    return;
  }
  handleUserPageChange(target, total, pageSize, currentPage, setPage);
}

function handleDepartmentJumpSubmit(
  input: string,
  currentPage: number,
  total: number,
  pageSize: number,
  setPage: (page: number) => void,
  setInput: (value: string) => void,
) {
  const target = Number.parseInt(input, 10);
  if (Number.isNaN(target)) {
    showToast("请输入正确页码");
    setInput(String(currentPage));
    return;
  }
  handleDepartmentPageChange(target, total, pageSize, currentPage, setPage);
}

function syncPagedList<T>(
  data: PageData<T>,
  pageSize: number,
  page: number,
  setPage: (nextPage: number) => void,
  setList: (nextList: T[]) => void,
  setTotal: (nextTotal: number) => void,
) {
  const pages = totalPages(data.total, pageSize);
  if (page > pages) {
    setPage(pages);
    return;
  }
  setList(data.list);
  setTotal(data.total);
}

function parseOptionalPositiveInt(raw: string): number | undefined {
  const value = raw.trim();
  if (!value) return undefined;
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed) || parsed <= 0) return undefined;
  return parsed;
}

function flattenDepartmentTreeOptions(nodes: DepartmentTreeNode[], level = 0): Array<{ id: number; label: string }> {
  const prefix = level <= 0 ? "" : `${"— ".repeat(level)}`;
  return nodes.flatMap((node) => {
    const current = { id: node.id, label: `${prefix}${node.name}` };
    const children = flattenDepartmentTreeOptions(node.children ?? [], level + 1);
    return [current, ...children];
  });
}

function formatDateTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString();
}
