import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";

import { fetchMyPermissions } from "@/api/auth";
import { apiClient } from "@/api/client";
import type { ApiResponse } from "@/api/types";
import { setToken, setUser } from "@/store/auth";
import { setPermissionBundle } from "@/store/permission";
import type { LoginResponse } from "@/types/auth";
import { showToast } from "@/utils/toast";

export function LoginPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const username = String(formData.get("username") ?? "");
    const password = String(formData.get("password") ?? "");
    if (!username || !password) {
      showToast("请输入用户名和密码");
      return;
    }
    setLoading(true);
    try {
      const { data } = await apiClient.post<ApiResponse<LoginResponse>>("/auth/login", {
        username,
        password,
      });
      setToken(data.data.token);
      setUser(data.data.user);
      try {
        const bundle = await fetchMyPermissions();
        setPermissionBundle(bundle);
      } catch {
        showToast("权限初始化失败，将按默认权限展示");
      }
      showToast("登录成功");
      navigate("/", { replace: true });
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <form className="auth-card" onSubmit={onSubmit}>
        <h1>SME DevOps</h1>
        <p>统一运维与平台治理控制台</p>
        <label htmlFor="username">Username</label>
        <input id="username" name="username" placeholder="admin" />
        <label htmlFor="password">Password</label>
        <input id="password" name="password" type="password" placeholder="Admin@123" />
        <button className="btn primary cursor-pointer" type="submit" disabled={loading}>
          {loading ? "登录中..." : "登录"}
        </button>
      </form>
    </main>
  );
}
