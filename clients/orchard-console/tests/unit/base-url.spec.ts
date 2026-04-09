import { describe, expect, it } from "vitest";
import { buildURLUnderBase } from "../../app/utils/base-url";

describe("base url builder", () => {
  it("preserves gateway base-path prefixes", () => {
    expect(
      buildURLUnderBase(
        "https://example.com/gateway",
        "/v1/auth/login",
      ).toString(),
    ).toBe("https://example.com/gateway/v1/auth/login");
  });

  it("keeps root-based deployments unchanged", () => {
    expect(
      buildURLUnderBase("https://example.com", "/v1/auth/login").toString(),
    ).toBe("https://example.com/v1/auth/login");
  });

  it("preserves query strings under a prefixed base path", () => {
    expect(
      buildURLUnderBase(
        "https://example.com/console",
        "/admin?tab=users",
      ).toString(),
    ).toBe("https://example.com/console/admin?tab=users");
  });
});
