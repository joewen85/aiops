import type { CmdbResourceItem } from "@/types/cmdb";

const defaultCMDBDataTTLHours = 24;

export function formatResourceBaseSpec(item: CmdbResourceItem): string {
  if ((item.type || "").toLowerCase() !== "vm") return "-";
  const cpu = readAttrString(item.attributes, "cpu", "vcpu", "cpuCore", "cpuCores");
  const memory = formatMemoryForDisplay(item.attributes);
  const disk = readAttrString(item.attributes, "disk", "diskGb", "diskGB", "diskSize");
  const privateIp = readAttrString(item.attributes, "privateIp", "private_ip", "innerIp", "privateIpAddress");
  const publicIp = readAttrString(item.attributes, "publicIp", "public_ip", "eip", "publicIpAddress");
  const operatingSystem = readAttrString(item.attributes, "os", "osName", "image", "imageName");

  const parts: string[] = [];
  if (cpu) parts.push(`CPU ${cpu}`);
  if (memory) parts.push(`内存 ${memory}`);
  if (disk) parts.push(`磁盘 ${disk}`);
  if (privateIp || publicIp) parts.push(`IP ${privateIp || "-"}${publicIp ? `/${publicIp}` : ""}`);
  if (operatingSystem) parts.push(`OS ${operatingSystem}`);
  return parts.length > 0 ? parts.join(" | ") : "-";
}

export function formatResourceExpiry(item: CmdbResourceItem, nowMs = Date.now(), ttlHours = defaultCMDBDataTTLHours): string {
  const businessExpireAt = parseDateTime(readAttrString(
    item.attributes,
    "expiresAt",
    "expireAt",
    "expireTime",
    "expiredAt",
    "expirationTime",
  ));
  const dataExpireAt = parseDateTime(item.lastSeenAt);
  if (dataExpireAt) {
    dataExpireAt.setHours(dataExpireAt.getHours() + ttlHours);
  }

  const business = formatExpirySegment("业务", businessExpireAt, nowMs);
  const data = formatExpirySegment("数据", dataExpireAt, nowMs);
  return `${business} | ${data}`;
}

function formatExpirySegment(label: string, value: Date | undefined, nowMs: number): string {
  if (!value) return `${label}:-`;
  return `${label}:${value.toLocaleString()}(${getExpiryStatus(value, nowMs)})`;
}

function getExpiryStatus(expireAt: Date, nowMs: number): string {
  const ttl = expireAt.getTime() - nowMs;
  if (ttl <= 0) return "已过期";
  if (ttl <= 72 * 60 * 60 * 1000) return "即将过期";
  return "正常";
}

function parseDateTime(value?: string): Date | undefined {
  if (!value) return undefined;
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed;
}

function readAttrString(attrs: Record<string, unknown> | undefined, ...keys: string[]): string {
  if (!attrs) return "";
  for (const key of keys) {
    if (!(key in attrs)) continue;
    const raw = attrs[key];
    const text = String(raw ?? "").trim();
    if (!text) continue;
    if (text === "[object Object]") continue;
    return text;
  }
  return "";
}

function formatMemoryForDisplay(attrs: Record<string, unknown> | undefined): string {
  const memory = readAttrString(attrs, "memory", "memoryGb", "memoryGB", "mem");
  if (memory) {
    if (containsAlphabet(memory)) return memory;
    const parsed = Number(memory);
    if (!Number.isNaN(parsed) && parsed > 0) return `${trimNumber(parsed)}G`;
    return memory;
  }

  const memoryMB = readAttrString(attrs, "memoryMB", "memoryMb", "memMB");
  if (memoryMB) {
    const normalized = memoryMB.toLowerCase().replace(/\s+/g, "");
    const withoutUnit = normalized.endsWith("mb") ? normalized.slice(0, -2) : normalized;
    const parsed = Number(withoutUnit);
    if (!Number.isNaN(parsed) && parsed > 0) return `${trimNumber(parsed / 1024)}G`;
  }

  const metadata = readMetadata(attrs);
  if (metadata) {
    const nestedMemory = readAttrString(metadata, "memory", "memoryGb", "memoryGB", "mem");
    if (nestedMemory) {
      if (containsAlphabet(nestedMemory)) return nestedMemory;
      const parsed = Number(nestedMemory);
      if (!Number.isNaN(parsed) && parsed > 0) return `${trimNumber(parsed)}G`;
      return nestedMemory;
    }
    const nestedMemoryMB = readAttrString(metadata, "memoryMB", "memoryMb", "memMB");
    if (nestedMemoryMB) {
      const normalized = nestedMemoryMB.toLowerCase().replace(/\s+/g, "");
      const withoutUnit = normalized.endsWith("mb") ? normalized.slice(0, -2) : normalized;
      const parsed = Number(withoutUnit);
      if (!Number.isNaN(parsed) && parsed > 0) return `${trimNumber(parsed / 1024)}G`;
    }
  }
  return "";
}

function containsAlphabet(value: string): boolean {
  return /[a-zA-Z]/.test(value);
}

function trimNumber(value: number): string {
  return value.toFixed(2).replace(/\.?0+$/, "");
}

function readMetadata(attrs: Record<string, unknown> | undefined): Record<string, unknown> | undefined {
  if (!attrs) return undefined;
  const raw = attrs.metadata;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  return raw as Record<string, unknown>;
}
