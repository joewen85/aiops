import { useEffect, useRef, useState } from "react";
import { getToken } from "@/store/auth";
import { showToast } from "@/utils/toast";

export interface WsMessage {
  channel: string;
  target: string;
  title: string;
  content: string;
  data?: unknown;
}

type WsListener = (message: WsMessage) => void;

const messageListeners = new Set<WsListener>();
let sharedSocket: WebSocket | null = null;
let sharedSocketUrl = "";
let sharedConnectTimer: number | null = null;
let subscriberCount = 0;
let manualClosing = false;
const wsDebugEnabled = parseDebugFlag(import.meta.env.VITE_WS_DEBUG);

function parseDebugFlag(raw: unknown): boolean {
  const value = String(raw ?? "").trim().toLowerCase();
  return value === "1" || value === "true" || value === "on" || value === "yes";
}

function wsDebugLog(event: string, detail?: unknown) {
  if (!wsDebugEnabled) return;
  if (detail === undefined) {
    console.debug(`[ws-debug] ${event}`);
    return;
  }
  console.debug(`[ws-debug] ${event}`, detail);
}

function emitMessage(message: WsMessage) {
  messageListeners.forEach((listener) => listener(message));
}

function closeSharedSocket() {
  if (sharedConnectTimer !== null) {
    window.clearTimeout(sharedConnectTimer);
    sharedConnectTimer = null;
    wsDebugLog("cancel pending connection timer");
  }
  if (!sharedSocket) return;
  wsDebugLog("close shared socket");
  manualClosing = true;
  sharedSocket.close();
  sharedSocket = null;
  sharedSocketUrl = "";
}

function ensureSharedSocket(url: string) {
  if (sharedSocket && sharedSocketUrl !== url) {
    wsDebugLog("socket url changed, recreate socket", { previous: sharedSocketUrl, next: url });
    closeSharedSocket();
  }
  if (sharedSocket || sharedConnectTimer !== null) {
    wsDebugLog("reuse existing socket or pending timer", {
      hasSocket: Boolean(sharedSocket),
      hasPendingTimer: sharedConnectTimer !== null,
    });
    return;
  }

  wsDebugLog("schedule shared socket connect", { subscriberCount });
  sharedConnectTimer = window.setTimeout(() => {
    sharedConnectTimer = null;
    if (subscriberCount <= 0 || sharedSocket) {
      wsDebugLog("skip connect due to no subscribers or existing socket", { subscriberCount, hasSocket: Boolean(sharedSocket) });
      return;
    }

    const ws = new WebSocket(url);
    sharedSocket = ws;
    sharedSocketUrl = url;
    wsDebugLog("create websocket", { url });

    ws.onopen = () => {
      wsDebugLog("socket open", { url: sharedSocketUrl });
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as WsMessage;
        wsDebugLog("socket message", {
          channel: msg.channel,
          target: msg.target,
          title: msg.title,
        });
        emitMessage(msg);
        if (msg.title || msg.content) {
          showToast(`${msg.title ?? "消息"}: ${msg.content ?? ""}`);
        }
      } catch {
        wsDebugLog("socket message parse failed");
        showToast("收到无法解析的实时消息");
      }
    };

    ws.onerror = () => {
      wsDebugLog("socket error");
      showToast("实时通道连接异常");
    };

    ws.onclose = (event) => {
      const closedByManual = manualClosing;
      manualClosing = false;
      if (sharedSocket === ws) {
        sharedSocket = null;
        sharedSocketUrl = "";
      }
      wsDebugLog("socket close", {
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean,
        closedByManual,
        subscriberCount,
      });
      if (!closedByManual && subscriberCount > 0) {
        showToast("实时通道已断开");
      }
    };
  }, 0);
}

export function useWebSocket(enabled: boolean) {
  const [messages, setMessages] = useState<WsMessage[]>([]);
  const listenerRef = useRef<WsListener | null>(null);

  useEffect(() => {
    if (!enabled) {
      wsDebugLog("hook disabled, skip subscribe");
      return;
    }
    const token = getToken();
    if (!token) {
      wsDebugLog("missing token, skip subscribe");
      return;
    }

    const base = import.meta.env.VITE_WS_URL ?? "ws://localhost:8080/ws";
    const url = `${base}?token=${encodeURIComponent(token)}`;

    const listener: WsListener = (msg) => {
      setMessages((prev) => [msg, ...prev].slice(0, 100));
    };
    listenerRef.current = listener;
    messageListeners.add(listener);
    subscriberCount += 1;
    wsDebugLog("subscribe hook listener", { subscriberCount });

    ensureSharedSocket(url);

    return () => {
      if (listenerRef.current) {
        messageListeners.delete(listenerRef.current);
        listenerRef.current = null;
      }
      subscriberCount = Math.max(0, subscriberCount - 1);
      wsDebugLog("unsubscribe hook listener", { subscriberCount });
      if (subscriberCount === 0) {
        closeSharedSocket();
      }
    };
  }, [enabled]);

  return { messages };
}
