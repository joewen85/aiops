import { useEffect, useState } from "react";

import type { PermissionBundle } from "@/types/permission";
import { getPermissionBundle, permissionEventName } from "@/store/permission";

export function usePermissionBundle() {
  const [bundle, setBundle] = useState<PermissionBundle | null>(() => getPermissionBundle());

  useEffect(() => {
    const eventName = permissionEventName();
    const handler = () => setBundle(getPermissionBundle());
    window.addEventListener(eventName, handler);
    return () => window.removeEventListener(eventName, handler);
  }, []);

  return bundle;
}
