import { createBrowserTokenStorage } from "../src/browser/storage";

describe("createBrowserTokenStorage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores namespaced tokens", () => {
    const storage = createBrowserTokenStorage({ namespace: "ponti" });
    storage.setAccessToken("a1");
    storage.setRefreshToken("r1");

    const host = window.location.host;
    expect(window.localStorage.getItem(`ponti:${host}:access_token`)).toBe("a1");
    expect(window.localStorage.getItem(`ponti:${host}:refresh_token`)).toBe("r1");
  });

  it("clears host-aware and legacy keys", () => {
    const host = window.location.host;
    window.localStorage.setItem("access_token", "legacy-access");
    window.localStorage.setItem("refresh_token", "legacy-refresh");
    window.localStorage.setItem(`ponti:${host}:access_token`, "scoped-access");
    window.localStorage.setItem(`ponti:${host}:refresh_token`, "scoped-refresh");

    const storage = createBrowserTokenStorage({ namespace: "ponti" });
    storage.clear();

    expect(window.localStorage.length).toBe(0);
  });
});
