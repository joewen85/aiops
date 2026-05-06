import type { ReactElement } from "react";
import { createBrowserRouter } from "react-router-dom";

import { MenuRouteGuard } from "@/components/MenuRouteGuard";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { AppLayout } from "@/layouts/AppLayout";
import { CMDBPage } from "@/pages/CMDBPage";
import { CloudPage } from "@/pages/CloudPage";
import { DashboardPage } from "@/pages/DashboardPage";
import { LoginPage } from "@/pages/LoginPage";
import { MessagesPage } from "@/pages/MessagesPage";
import { ModulePage } from "@/pages/ModulePage";
import { RBACPage } from "@/pages/RBACPage";
import { UsersPage } from "@/pages/UsersPage";

function withMenuGuard(permissionKey: string, element: ReactElement) {
  return <MenuRouteGuard permissionKey={permissionKey}>{element}</MenuRouteGuard>;
}

export const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  {
    path: "/",
    element: <ProtectedRoute />,
    children: [
      {
        path: "/",
        element: <AppLayout />,
        children: [
          { index: true, element: withMenuGuard("menu.dashboard", <DashboardPage />) },
          { path: "rbac", element: withMenuGuard("menu.rbac", <RBACPage />) },
          { path: "users", element: withMenuGuard("menu.users", <UsersPage />) },
          { path: "cmdb", element: withMenuGuard("menu.cmdb", <CMDBPage />) },
          { path: "tasks", element: withMenuGuard("menu.tasks", <ModulePage />) },
          { path: "messages", element: withMenuGuard("menu.messages", <MessagesPage />) },
          { path: "cloud", element: withMenuGuard("menu.cloud", <CloudPage />) },
          { path: "tickets", element: withMenuGuard("menu.tickets", <ModulePage />) },
          { path: "docker", element: withMenuGuard("menu.docker", <ModulePage />) },
          { path: "middleware", element: withMenuGuard("menu.middleware", <ModulePage />) },
          { path: "observability", element: withMenuGuard("menu.observability", <ModulePage />) },
          { path: "kubernetes", element: withMenuGuard("menu.kubernetes", <ModulePage />) },
          { path: "events", element: withMenuGuard("menu.events", <ModulePage />) },
          { path: "tools", element: withMenuGuard("menu.tools", <ModulePage />) },
          { path: "aiops", element: withMenuGuard("menu.aiops", <ModulePage />) },
          { path: "audit", element: withMenuGuard("menu.audit", <ModulePage />) },
        ],
      },
    ],
  },
]);
