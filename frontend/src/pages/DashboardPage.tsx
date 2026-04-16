import { useWebSocket } from "@/hooks/useWebSocket";

export function DashboardPage() {
  const { messages } = useWebSocket(true);

  return (
    <section className="page">
      <h2>平台概览</h2>
      <div className="grid cards">
        <article className="card">
          <h3>任务执行</h3>
          <p>统一追踪 Ansible 任务与执行日志。</p>
        </article>
        <article className="card">
          <h3>CMDB 资源</h3>
          <p>支持动态属性与标签体系管理。</p>
        </article>
        <article className="card">
          <h3>权限治理</h3>
          <p>RBAC + ABAC 分层控制接口与菜单。</p>
        </article>
      </div>
      <section className="card">
        <h3>实时消息（WebSocket）</h3>
        <div className="message-list">
          {messages.length === 0 && <p className="muted">暂无实时消息</p>}
          {messages.map((msg, idx) => (
            <div key={`${msg.title}-${idx}`} className="message-item">
              <strong>{msg.title || "消息"}</strong>
              <span>{msg.content}</span>
            </div>
          ))}
        </div>
      </section>
    </section>
  );
}
