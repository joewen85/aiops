import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";

import { usePermissionBundle } from "@/hooks/usePermissionBundle";
import { hasMenuPermission, isAdminUser } from "@/store/permission";

interface MenuRouteGuardProps {
  permissionKey: string;
  children: ReactNode;
}

export function MenuRouteGuard({ permissionKey, children }: MenuRouteGuardProps) {
  const bundle = usePermissionBundle();

  if (isAdminUser()) {
    return <>{children}</>;
  }
  if (!bundle) {
    return <Navigate to="/login" replace />;
  }
  if (!hasMenuPermission(permissionKey, bundle)) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
