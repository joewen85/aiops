import type { ButtonHTMLAttributes, ReactNode } from "react";

import { usePermissionBundle } from "@/hooks/usePermissionBundle";
import { hasButtonPermission } from "@/store/permission";

interface PermissionButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  permissionKey: string;
  children: ReactNode;
}

export function PermissionButton({ permissionKey, children, ...props }: PermissionButtonProps) {
  const bundle = usePermissionBundle();
  const allowed = hasButtonPermission(permissionKey, bundle);
  if (!allowed) return null;
  return <button {...props}>{children}</button>;
}
