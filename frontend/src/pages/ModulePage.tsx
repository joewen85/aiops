import { useMemo } from "react";
import { useLocation } from "react-router-dom";

const titleMap: Record<string, string> = {
  "/rbac": "RBAC / ABAC 权限管理",
  "/users": "用户与部门",
  "/cmdb": "CMDB",
  "/tasks": "任务中心",
  "/cloud": "多云管理",
  "/tickets": "工单管理",
  "/docker": "Docker 管理",
  "/middleware": "中间件管理",
  "/observability": "可观测性",
  "/kubernetes": "Kubernetes 管理",
  "/events": "事件中心",
  "/tools": "工具市场",
  "/aiops": "AIOps",
  "/messages": "站内消息",
  "/audit": "审计日志",
};

export function ModulePage() {
  const location = useLocation();
  const title = useMemo(() => titleMap[location.pathname] ?? "模块", [location.pathname]);

  return (
    <section className="page">
      <h2>{title}</h2>
      <article className="card">
        <p>该模块已接入统一骨架（鉴权、权限、分页、错误结构、审计）。</p>
        <p>下一步按 `PLAN.md` 逐项补齐业务详情页与联动能力。</p>
      </article>
    </section>
  );
}
