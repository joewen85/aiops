import { Navigate, Outlet, useLocation } from "react-router-dom";
import { getToken } from "@/store/auth";

export function ProtectedRoute() {
  const token = getToken();
  const location = useLocation();
  if (!token) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return <Outlet />;
}
