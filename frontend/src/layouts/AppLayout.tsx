import { useEffect, useMemo, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { fetchMyPermissions } from "@/api/auth";
import { ThemeToggle } from "@/components/ThemeToggle";
import { usePermissionBundle } from "@/hooks/usePermissionBundle";
import { useTheme } from "@/hooks/useTheme";
import { getUser, logout } from "@/store/auth";
import { clearPermissionBundle, hasMenuPermission, isAdminUser, setPermissionBundle } from "@/store/permission";
import { showToast } from "@/utils/toast";

const menu = [
  { to: "/", label: "概览", key: "menu.dashboard" },
  { to: "/rbac", label: "权限", key: "menu.rbac" },
  { to: "/users", label: "用户部门", key: "menu.users" },
  { to: "/cmdb", label: "CMDB", key: "menu.cmdb" },
  { to: "/tasks", label: "任务中心", key: "menu.tasks" },
  { to: "/messages", label: "站内消息", key: "menu.messages" },
  { to: "/cloud", label: "多云", key: "menu.cloud" },
  { to: "/tickets", label: "工单", key: "menu.tickets" },
  { to: "/docker", label: "Docker", key: "menu.docker" },
  { to: "/middleware", label: "中间件", key: "menu.middleware" },
  { to: "/observability", label: "可观测性", key: "menu.observability" },
  { to: "/kubernetes", label: "K8s", key: "menu.kubernetes" },
  { to: "/events", label: "事件", key: "menu.events" },
  { to: "/tools", label: "工具市场", key: "menu.tools" },
  { to: "/aiops", label: "AIOps", key: "menu.aiops" },
  { to: "/audit", label: "审计日志", key: "menu.audit" },
];

export function AppLayout() {
  const { mode, toggle } = useTheme();
  const navigate = useNavigate();
  const user = getUser();
  const permissionBundle = usePermissionBundle();
  const [permissionReady, setPermissionReady] = useState(false);
  const adminUser = isAdminUser();

  useEffect(() => {
    let mounted = true;
    async function syncPermissions() {
      try {
        const bundle = await fetchMyPermissions();
        setPermissionBundle(bundle);
      } catch (error: any) {
        clearPermissionBundle();
        if (error?.response?.status !== 401) {
          showToast("权限加载失败，请重新登录");
        }
        logout();
        navigate("/login", { replace: true });
      } finally {
        if (mounted) setPermissionReady(true);
      }
    }
    void syncPermissions();
    return () => {
      mounted = false;
    };
  }, [navigate]);

  const visibleMenu = useMemo(() => {
    if (adminUser) return menu;
    if (!permissionReady) return [];
    if (!permissionBundle) return [];
    return menu.filter((item) => hasMenuPermission(item.key, permissionBundle));
  }, [adminUser, permissionBundle, permissionReady]);

  function handleLogout() {
    logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>SME DevOps</h1>
        <nav>
          {visibleMenu.map((item) => (
            <NavLink key={item.to} className="menu-item cursor-pointer" to={item.to} end={item.to === "/"}>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="main">
        <header className="topbar">
          <div className="topbar-left">
            <span className="muted">欢迎，{user?.displayName ?? user?.username ?? "User"}</span>
          </div>
          <div className="topbar-right">
            <ThemeToggle mode={mode} onToggle={toggle} />
            <button className="btn cursor-pointer" onClick={handleLogout}>退出</button>
          </div>
        </header>
        <main className="content">
          {!adminUser && !permissionReady
            ? <p className="muted">权限加载中...</p>
            : <Outlet />}
        </main>
      </div>
    </div>
  );
}
