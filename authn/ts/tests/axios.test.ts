import axios from "axios";
import MockAdapter from "axios-mock-adapter";
import { AUTH_FORCE_LOGOUT_EVENT } from "../src/browser/events";
import { createAuthenticatedAxiosClient } from "../src/http/axios";

function createMemoryTokenStorage() {
  let accessToken: string | null = "expired-access";
  let refreshToken: string | null = "refresh-1";

  return {
    getAccessToken: () => accessToken,
    getRefreshToken: () => refreshToken,
    setAccessToken: (token: string) => {
      accessToken = token;
    },
    setRefreshToken: (token: string) => {
      refreshToken = token;
    },
    clear: () => {
      accessToken = null;
      refreshToken = null;
    },
  };
}

describe("createAuthenticatedAxiosClient", () => {
  it("refreshes once and retries the original request", async () => {
    const storage = createMemoryTokenStorage();
    const client = createAuthenticatedAxiosClient({
      tokenStorage: storage,
      refreshRequest: {
        path: "/auth/access-token",
        mapResponse(data) {
          const payload = data as { access_token: string; refresh_token: string };
          return { accessToken: payload.access_token, refreshToken: payload.refresh_token };
        },
      },
    });

    const mock = new MockAdapter(client.raw());
    mock.onGet("/protected").replyOnce(401);
    mock.onGet("/auth/access-token").replyOnce(200, {
      access_token: "new-access",
      refresh_token: "new-refresh",
    });
    mock.onGet("/protected").replyOnce(200, { ok: true });

    await expect(client.get("/protected")).resolves.toEqual({ ok: true });
    expect(storage.getAccessToken()).toBe("new-access");
    expect(storage.getRefreshToken()).toBe("new-refresh");
  });

  it("dispatches force logout when refresh fails", async () => {
    const storage = createMemoryTokenStorage();
    const client = createAuthenticatedAxiosClient({
      tokenStorage: storage,
      refreshRequest: {
        path: "/auth/access-token",
        mapResponse(data) {
          const payload = data as { access_token: string; refresh_token: string };
          return { accessToken: payload.access_token, refreshToken: payload.refresh_token };
        },
      },
    });

    const listener = vi.fn();
    window.addEventListener(AUTH_FORCE_LOGOUT_EVENT, listener);

    const mock = new MockAdapter(client.raw());
    mock.onGet("/protected").replyOnce(401);
    mock.onGet("/auth/access-token").replyOnce(401, { error: "expired" });

    await expect(client.get("/protected")).rejects.toBeTruthy();
    expect(listener).toHaveBeenCalledTimes(1);
    expect(storage.getAccessToken()).toBeNull();
    expect(storage.getRefreshToken()).toBeNull();

    window.removeEventListener(AUTH_FORCE_LOGOUT_EVENT, listener);
  });
});
