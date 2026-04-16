import axios from "axios";
import { getToken, logout } from "@/store/auth";
import { showToast } from "@/utils/toast";

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "/api/v1",
  timeout: 30000,
});

apiClient.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const status = error?.response?.status;
    const message = error?.response?.data?.message ?? error.message ?? "request failed";

    if (status === 401) {
      logout();
      showToast("登录已失效，请重新登录");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
      return Promise.reject(error);
    }

    showToast(message);
    return Promise.reject(error);
  },
);
