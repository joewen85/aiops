import { describe, expect, it } from "vitest";

import { formatBindingNameList } from "@/utils/bindingDisplay";

describe("formatBindingNameList", () => {
  it("formats object names from normal API shape", () => {
    expect(formatBindingNameList([{ id: 1, name: "管理员" }, { id: 2, name: "运维" }], [], "角色")).toBe("管理员、运维");
  });

  it("supports legacy uppercase field names", () => {
    expect(formatBindingNameList([{ ID: 1, Name: "研发部" }], [], "部门")).toBe("研发部");
  });

  it("falls back to ids when names are unavailable", () => {
    expect(formatBindingNameList([], [3, "4"], "角色")).toBe("角色#3、角色#4");
  });

  it("returns dash when no binding data exists", () => {
    expect(formatBindingNameList([], [], "部门")).toBe("-");
  });
});
