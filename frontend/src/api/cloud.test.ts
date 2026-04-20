import { beforeEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "@/api/client";
import { listCloudAccounts } from "@/api/cloud";

describe("cloud api account list query", () => {
  const getSpy = vi.spyOn(apiClient, "get");

  beforeEach(() => {
    getSpy.mockReset();
    getSpy.mockResolvedValue({
      data: {
        code: 0,
        message: "ok",
        data: {
          list: [],
          total: 0,
          page: 1,
          pageSize: 10,
        },
      },
    } as never);
  });

  it("uses default paging query", async () => {
    await listCloudAccounts();
    expect(getSpy).toHaveBeenCalledTimes(1);
    expect(getSpy.mock.calls[0]?.[0]).toBe("/cloud/accounts?page=1&pageSize=10");
  });

  it("appends account filters into query string", async () => {
    await listCloudAccounts({
      page: 2,
      pageSize: 20,
      keyword: "prod",
      provider: "aws",
      region: "ap-southeast-1",
      verified: "true",
    });
    expect(getSpy).toHaveBeenCalledTimes(1);
    expect(getSpy.mock.calls[0]?.[0]).toBe(
      "/cloud/accounts?page=2&pageSize=20&keyword=prod&provider=aws&region=ap-southeast-1&verified=true",
    );
  });
});
