import { describe, expect, it } from "vitest";

import type { CmdbResourceItem } from "@/types/cmdb";
import { formatResourceBaseSpec, formatResourceExpiry } from "@/pages/cmdbResourceDisplay";

describe("cmdb resource display helpers", () => {
  it("formats VM base spec and expiry text", () => {
    const now = Date.parse("2026-04-20T00:00:00Z");
    const item: CmdbResourceItem = {
      id: 1,
      ciId: "aws:acc-a:ap-southeast-1:i-001",
      type: "VM",
      name: "api-vm-1",
      lastSeenAt: "2026-04-19T20:00:00Z",
      attributes: {
        cpu: "4",
        memory: "16Gi",
        disk: "200Gi",
        privateIp: "10.0.0.10",
        publicIp: "198.51.100.10",
        os: "Ubuntu 22.04",
        expiresAt: "2026-04-22T00:00:00Z",
      },
    };

    const baseSpec = formatResourceBaseSpec(item);
    expect(baseSpec).toContain("CPU 4");
    expect(baseSpec).toContain("内存 16Gi");
    expect(baseSpec).toContain("磁盘 200Gi");
    expect(baseSpec).toContain("IP 10.0.0.10/198.51.100.10");
    expect(baseSpec).toContain("OS Ubuntu 22.04");

    const expiry = formatResourceExpiry(item, now);
    expect(expiry).toContain("业务:");
    expect(expiry).toContain("数据:");
    expect(expiry).toContain("即将过期");
  });

  it("returns placeholder for non-VM base spec and missing expiry", () => {
    const now = Date.parse("2026-04-20T00:00:00Z");
    const item: CmdbResourceItem = {
      id: 2,
      ciId: "cmdb:service:live-api",
      type: "Service",
      name: "live-api",
      attributes: {},
    };

    expect(formatResourceBaseSpec(item)).toBe("-");
    expect(formatResourceExpiry(item, now)).toBe("业务:- | 数据:-");
  });

  it("adds G unit for numeric memory and converts memoryMB", () => {
    const vmWithNumericMemory: CmdbResourceItem = {
      id: 3,
      ciId: "tencent:acc-1:ap-guangzhou:ins-001",
      type: "VM",
      name: "vm-a",
      attributes: {
        cpu: 4,
        memory: 16,
      },
    };
    expect(formatResourceBaseSpec(vmWithNumericMemory)).toContain("内存 16G");

    const vmWithMemoryMB: CmdbResourceItem = {
      id: 4,
      ciId: "aliyun:acc-1:cn-hangzhou:i-001",
      type: "VM",
      name: "vm-b",
      attributes: {
        cpu: 4,
        memoryMB: 16384,
      },
    };
    expect(formatResourceBaseSpec(vmWithMemoryMB)).toContain("内存 16G");
  });

  it("reads memory from nested metadata fallback", () => {
    const vmNestedMB: CmdbResourceItem = {
      id: 5,
      ciId: "aliyun:acc-2:cn-beijing:i-002",
      type: "VM",
      name: "vm-c",
      attributes: {
        metadata: {
          memoryMB: 16384,
        },
      },
    };
    expect(formatResourceBaseSpec(vmNestedMB)).toContain("内存 16G");
  });
});
