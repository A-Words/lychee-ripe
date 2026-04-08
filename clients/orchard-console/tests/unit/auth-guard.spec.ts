import { describe, expect, it } from "vitest";
import { resolveAuthGuardDecision } from "../../app/utils/auth-guard";

describe("auth guard decisions", () => {
  it("skips redirect decisions during SSR", () => {
    expect(
      resolveAuthGuardDecision({
        path: "/dashboard",
        fullPath: "/dashboard",
        isServer: true,
        isAuthenticated: false,
        isAdmin: false,
      }),
    ).toEqual({ kind: "allow" });
  });

  it("redirects unauthenticated browser requests to login", () => {
    expect(
      resolveAuthGuardDecision({
        path: "/batch/create",
        fullPath: "/batch/create?from=deep-link",
        isServer: false,
        isAuthenticated: false,
        isAdmin: false,
      }),
    ).toEqual({
      kind: "login",
      redirect: "/batch/create?from=deep-link",
    });
  });

  it("allows authenticated browser refreshes on protected routes", () => {
    expect(
      resolveAuthGuardDecision({
        path: "/dashboard",
        fullPath: "/dashboard",
        isServer: false,
        isAuthenticated: true,
        isAdmin: false,
      }),
    ).toEqual({ kind: "allow" });
  });

  it("redirects non-admin users away from admin routes", () => {
    expect(
      resolveAuthGuardDecision({
        path: "/admin",
        fullPath: "/admin",
        isServer: false,
        isAuthenticated: true,
        isAdmin: false,
      }),
    ).toEqual({ kind: "dashboard" });
  });

  it("allows public routes without a session", () => {
    expect(
      resolveAuthGuardDecision({
        path: "/trace/TRC-9A7X-11QF",
        fullPath: "/trace/TRC-9A7X-11QF",
        isServer: false,
        isAuthenticated: false,
        isAdmin: false,
      }),
    ).toEqual({ kind: "allow" });
  });
});
